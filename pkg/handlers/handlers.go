package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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

	// Set basic page data
	pageData.MaxPoints = maxPoints
	pageData.MinPoints = minPoints
	pageData.BreakPointPercent = breakPointPercent
	pageData.GradeBounds = gradeBounds
	pageData.HasResults = true
	pageData.CalculationSuccess = true

	// Process uploaded CSV file if present
	file, fileHeader, err := r.FormFile("csvFile")
	if err == nil {
		defer file.Close()

		logging.LogInfo("Processing uploaded CSV file",
			"filename", fileHeader.Filename,
			"size", fileHeader.Size,
			"ip", ip)

		students, err := calculator.ParseCSVFile(fileHeader)
		if err != nil {
			logging.LogError("Failed to parse CSV file", err,
				"filename", fileHeader.Filename,
				"size", fileHeader.Size,
				"ip", ip)
			pageData.Message = &models.Message{
				Type: models.MessageError,
				Text: fmt.Sprintf("Fehler beim Verarbeiten der CSV-Datei: %v", err),
			}
		} else {
			// Calculate grades for students
			students = calculator.ProcessStudents(students, gradeBounds)
			averageGrade := calculator.CalculateAverageGrade(students)

			pageData.Students = students
			pageData.AverageGrade = averageGrade
			pageData.HasStudents = true

			logging.LogInfo("Students processed successfully",
				"student_count", len(students),
				"average_grade", averageGrade,
				"filename", fileHeader.Filename,
				"ip", ip)
		}
	} else if err != http.ErrMissingFile {
		logging.LogError("Error accessing uploaded file", err, "ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Fehler beim Zugriff auf die hochgeladene Datei",
		}
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
