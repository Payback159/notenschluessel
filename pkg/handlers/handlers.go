package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/payback159/notenschluessel/pkg/calculator"
	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"github.com/payback159/notenschluessel/pkg/security"
	"github.com/payback159/notenschluessel/pkg/session"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	Templates    *template.Template
	SessionStore *session.Store
}

// NewHandler creates a new handler with dependencies
func NewHandler(templates *template.Template, sessionStore *session.Store) *Handler {
	return &Handler{
		Templates:    templates,
		SessionStore: sessionStore,
	}
}

// Go 1.25+ native cross-origin protection - no CSRF token functions needed

func (h *Handler) executeTemplateSafe(w http.ResponseWriter, templateName string, data interface{}, sessionID, ip string) {
	if err := h.Templates.ExecuteTemplate(w, templateName, data); err != nil {
		logging.LogError("Failed to execute template", err,
			"template", templateName,
			"session_id", sessionID,
			"ip", ip)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleHome handles the main page requests (GET and POST)
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		pageData := models.PageData{
			InputMode: models.InputModeCSV,
			// Go 1.25+ native cross-origin protection - no CSRF field needed
		}
		h.executeTemplateSafe(w, "index.html", pageData, "", security.GetClientIP(r))
		return
	}

	if r.Method == http.MethodPost {
		h.HandleCalculation(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleCalculation processes the grade calculation
func (h *Handler) HandleCalculation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ip := security.GetClientIP(r)

	logging.LogInfo("Processing calculation request",
		"ip", ip,
		"user_agent", r.UserAgent())

	// Parse form data
	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
	if err != nil {
		logging.LogError("Failed to parse multipart form", err, "ip", ip)
		http.Error(w, "Fehler beim Parsen des Formulars", http.StatusBadRequest)
		return
	}

	pageData := models.PageData{}

	// Generate session ID early for error handling
	sessionID, err := session.GenerateSessionID()
	if err != nil {
		logging.LogError("Failed to generate session ID", err, "ip", ip)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse input parameters
	maxPointsStr := r.FormValue("maxPoints")
	minPointsStr := r.FormValue("minPoints")
	breakPointPercentStr := r.FormValue("breakPointPercent")
	inputMode := models.InputMode(r.FormValue("inputMode"))
	if inputMode == "" {
		inputMode = models.InputModeCSV
	}
	pageData.InputMode = inputMode
	pageData.ManualEntries = parseManualEntries(r)

	maxPoints, err := strconv.Atoi(maxPointsStr)
	if err != nil || maxPoints <= 0 || maxPoints > 1000 {
		logging.LogWarn("Invalid max points value",
			"max_points_str", maxPointsStr,
			"error", err,
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültige maximale Punktzahl (1-1000 erlaubt)",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	minPoints, err := strconv.ParseFloat(minPointsStr, 64)
	if err != nil || minPoints <= 0 || minPoints > float64(maxPoints) {
		logging.LogWarn("Invalid min points value",
			"min_points_str", minPointsStr,
			"error", err,
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültige Punkteschrittweite",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	breakPointPercent, err := strconv.ParseFloat(breakPointPercentStr, 64)
	if err != nil || breakPointPercent < 1 || breakPointPercent > 99 {
		logging.LogWarn("Invalid break point value",
			"break_point_str", breakPointPercentStr,
			"error", err,
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültiger Knickpunkt (1-99% erlaubt)",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	// Calculate grade bounds
	gradeBounds := calculator.CalculateGradeBounds(maxPoints, minPoints, breakPointPercent)
	if valid, reason := calculator.ValidateGradeBounds(gradeBounds); !valid {
		logging.LogWarn("Invalid grade scale configuration",
			"max_points", maxPoints,
			"min_points", minPoints,
			"break_point_percent", breakPointPercent,
			"reason", reason,
			"ip", ip)

		pageData.MaxPoints = maxPoints
		pageData.MinPoints = minPoints
		pageData.BreakPointPercent = breakPointPercent
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Diese Kombination aus maximaler Punktzahl, Schrittweite und Knickpunkt ergibt keine gültige Notenskala. Bitte Schrittweite verkleinern oder Punktzahl erhöhen.",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	// Set basic page data
	pageData.MaxPoints = maxPoints
	pageData.MinPoints = minPoints
	pageData.BreakPointPercent = breakPointPercent
	pageData.GradeBounds = gradeBounds
	pageData.HasResults = true
	pageData.CalculationSuccess = true

	// Detect if a CSV file was uploaded.
	csvFile, fileHeader, fileErr := r.FormFile("csvFile")
	csvProvided := false
	if fileErr == nil {
		csvProvided = true
		defer csvFile.Close()
	} else if fileErr != http.ErrMissingFile {
		logging.LogError("Error accessing uploaded file", fileErr, "ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Fehler beim Zugriff auf die hochgeladene Datei",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	manualProvided := hasNonEmptyManualEntries(pageData.ManualEntries)

	// Input methods are offered as alternatives and may not be combined.
	if csvProvided && manualProvided {
		logging.LogWarn("Both CSV and manual input provided",
			"ip", ip,
			"manual_entries", len(pageData.ManualEntries))
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Bitte entweder CSV-Import oder manuelle Eingabe verwenden. Eine Kombination ist nicht erlaubt.",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	if inputMode != models.InputModeCSV && inputMode != models.InputModeManual {
		logging.LogWarn("Invalid input mode",
			"input_mode", inputMode,
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültiger Eingabemodus",
		}
		h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
		return
	}

	var students []models.Student
	switch inputMode {
	case models.InputModeCSV:
		if csvProvided {
			logging.LogInfo("Processing uploaded CSV file",
				"filename", fileHeader.Filename,
				"size", fileHeader.Size,
				"ip", ip)

			students, err = calculator.ParseCSVFile(fileHeader)
			if err != nil {
				logging.LogError("Failed to parse CSV file", err,
					"filename", fileHeader.Filename,
					"size", fileHeader.Size,
					"ip", ip)
				pageData.Message = &models.Message{
					Type: models.MessageError,
					Text: fmt.Sprintf("Fehler beim Verarbeiten der CSV-Datei: %v", err),
				}
			}
		}
	case models.InputModeManual:
		if manualProvided {
			students, err = parseManualStudents(pageData.ManualEntries)
			if err != nil {
				logging.LogWarn("Invalid manual student entry",
					"error", err.Error(),
					"ip", ip)
				pageData.Message = &models.Message{
					Type: models.MessageError,
					Text: err.Error(),
				}
			}
		}
	}

	if pageData.Message == nil && len(students) > 0 {
		students = calculator.ProcessStudents(students, gradeBounds)
		averageGrade := calculator.CalculateAverageGrade(students)

		pageData.Students = students
		pageData.AverageGrade = averageGrade
		pageData.HasStudents = true

		logging.LogInfo("Students processed successfully",
			"input_mode", inputMode,
			"student_count", len(students),
			"average_grade", averageGrade,
			"ip", ip)
	}

	// Store data in session if no errors occurred
	if pageData.Message == nil || pageData.Message.Type != models.MessageError {
		h.SessionStore.Set(sessionID, pageData)
		pageData.SessionID = sessionID

		// Set session cookie (HttpOnly, SameSite=Strict, Secure) to prevent IDOR
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   models.SessionTimeout,
		})

		logging.LogDebug("Session created",
			"session_id", sessionID,
			"has_students", pageData.HasStudents,
			"student_count", len(pageData.Students))
	}

	// Log successful completion
	duration := time.Since(start)
	logging.LogCalculation(maxPoints, minPoints, breakPointPercent, len(pageData.Students), duration, true)

	logging.LogInfo("Calculation completed successfully",
		"session_id", sessionID,
		"input_mode", inputMode,
		"max_points", maxPoints,
		"min_points", minPoints,
		"break_point_percent", breakPointPercent,
		"has_students", pageData.HasStudents,
		"student_count", len(pageData.Students),
		"average_grade", pageData.AverageGrade,
		"duration_ms", duration.Milliseconds(),
		"ip", ip)

	// Render template with results
	h.executeTemplateSafe(w, "index.html", pageData, sessionID, ip)
}

// HandlePrivacy renders the privacy information page.
func (h *Handler) HandlePrivacy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.executeTemplateSafe(w, "privacy.html", models.PageData{}, "", security.GetClientIP(r))
}

// HandleDeleteSession removes the current user's session and clears the cookie.
func (h *Handler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := security.GetClientIP(r)
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		h.SessionStore.Delete(cookie.Value)
		logging.LogInfo("Session deleted by user request",
			"session_id", cookie.Value,
			"ip", ip)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	pageData := models.PageData{
		InputMode: models.InputModeCSV,
		Message: &models.Message{
			Type: models.MessageSuccess,
			Text: "Sitzung und temporäre Daten wurden gelöscht.",
		},
	}

	h.executeTemplateSafe(w, "index.html", pageData, "", ip)
}

func parseManualEntries(r *http.Request) []models.ManualEntry {
	if r.MultipartForm == nil {
		return nil
	}

	names := r.MultipartForm.Value["manualName"]
	points := r.MultipartForm.Value["manualPoints"]
	maxLen := len(names)
	if len(points) > maxLen {
		maxLen = len(points)
	}

	entries := make([]models.ManualEntry, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		entry := models.ManualEntry{}
		if i < len(names) {
			entry.Name = strings.TrimSpace(names[i])
		}
		if i < len(points) {
			entry.Points = strings.TrimSpace(points[i])
		}
		entries = append(entries, entry)
	}

	return entries
}

func hasNonEmptyManualEntries(entries []models.ManualEntry) bool {
	for _, entry := range entries {
		if entry.Name != "" || entry.Points != "" {
			return true
		}
	}
	return false
}

func parseManualStudents(entries []models.ManualEntry) ([]models.Student, error) {
	students := make([]models.Student, 0, len(entries))
	for i, entry := range entries {
		name := strings.TrimSpace(entry.Name)
		pointsStr := strings.TrimSpace(entry.Points)

		if name == "" && pointsStr == "" {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("Zeile %d: Name fehlt", i+1)
		}
		if pointsStr == "" {
			return nil, fmt.Errorf("Zeile %d: Punkte fehlen", i+1)
		}

		pointsStr = strings.ReplaceAll(pointsStr, ",", ".")
		points, err := strconv.ParseFloat(pointsStr, 64)
		if err != nil {
			return nil, fmt.Errorf("Zeile %d: Ungültige Punktzahl", i+1)
		}
		if points < 0 || points > 1000 {
			return nil, fmt.Errorf("Zeile %d: Punktzahl außerhalb des erlaubten Bereichs (0-1000)", i+1)
		}

		students = append(students, models.Student{
			Name:   security.SanitizeName(name),
			Points: points,
		})

		if len(students) > models.MaxStudents {
			return nil, fmt.Errorf("Zu viele Schülerdaten (maximal %d)", models.MaxStudents)
		}
	}

	return students, nil
}
