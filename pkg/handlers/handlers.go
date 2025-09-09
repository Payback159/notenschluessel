package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/csrf"
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

// getCSRFField returns CSRF field for production, empty string for development
func getCSRFField(r *http.Request) template.HTML {
	if os.Getenv("ENV") == "production" {
		return csrf.TemplateField(r)
	}
	return template.HTML("")
}

// HandleHome handles the main page requests (GET and POST)
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		pageData := models.PageData{
			CSRFField: getCSRFField(r),
		}
		h.Templates.ExecuteTemplate(w, "index.html", pageData)
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

	logging.LogInfo("HTTP request processing initiated",
		"method", r.Method,
		"content_length", r.ContentLength,
		"ip", ip)

	// Parse form data
	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
	if err != nil {
		logging.LogError("Form parsing failed", err,
			"content_length", r.ContentLength,
			"content_type", r.Header.Get("Content-Type"),
			"ip", ip)
		http.Error(w, "Fehler beim Parsen des Formulars", http.StatusBadRequest)
		return
	}

	pageData := models.PageData{
		CSRFField: getCSRFField(r),
	}

	// Parse input parameters
	maxPointsStr := r.FormValue("maxPoints")
	minPointsStr := r.FormValue("minPoints")
	breakPointPercentStr := r.FormValue("breakPointPercent")

	maxPoints, err := strconv.Atoi(maxPointsStr)
	if err != nil || maxPoints <= 0 || maxPoints > 1000 {
		logging.LogWarn("Invalid form input detected",
			"field", "maxPoints",
			"validation_error", "out_of_range_or_invalid",
			"valid_range", "1-1000",
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültige maximale Punktzahl (1-1000 erlaubt)",
		}
		h.Templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}

	minPoints, err := strconv.ParseFloat(minPointsStr, 64)
	if err != nil || minPoints <= 0 || minPoints > float64(maxPoints) {
		logging.LogWarn("Invalid form input detected",
			"field", "minPoints",
			"validation_error", "negative_or_exceeds_max",
			"constraint", "positive_and_below_max",
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültige Punkteschrittweite",
		}
		h.Templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}

	breakPointPercent, err := strconv.ParseFloat(breakPointPercentStr, 64)
	if err != nil || breakPointPercent < 1 || breakPointPercent > 99 {
		logging.LogWarn("Invalid form input detected",
			"field", "breakPointPercent",
			"validation_error", "out_of_range",
			"valid_range", "1-99",
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Ungültiger Knickpunkt (1-99% erlaubt)",
		}
		h.Templates.ExecuteTemplate(w, "index.html", pageData)
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

		logging.LogInfo("File upload processing initiated",
			"content_type", fileHeader.Header.Get("Content-Type"),
			"content_length", fileHeader.Size,
			"processing_stage", "parsing",
			"ip", ip)

		students, err := calculator.ParseCSVFile(fileHeader)
		if err != nil {
			logging.LogError("CSV parsing operation failed", err,
				"content_length", fileHeader.Size,
				"parse_stage", "file_processing",
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

			logging.LogInfo("CSV processing completed successfully",
				"records_processed", len(students),
				"calculation_result", averageGrade,
				"processing_stage", "completed",
				"ip", ip)
		}
	} else if err != http.ErrMissingFile {
		logging.LogError("File access operation failed", err,
			"operation", "file_upload_access",
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Fehler beim Zugriff auf die hochgeladene Datei",
		}
	}

	// Generate session ID and store data
	sessionID, err := session.GenerateSessionID()
	if err != nil {
		logging.LogError("Session ID generation failed", err,
			"operation", "session_management",
			"ip", ip)
		pageData.Message = &models.Message{
			Type: models.MessageError,
			Text: "Systemfehler bei der Session-Erstellung",
		}
	} else {
		h.SessionStore.Set(sessionID, pageData)
		pageData.SessionID = sessionID

		logging.LogDebug("Session management completed",
			"session_id_length", len(sessionID),
			"data_cached", pageData.HasStudents,
			"cache_size", len(pageData.Students),
			"session_status", "active")
	}

	// Log successful completion
	duration := time.Since(start)
	logging.LogCalculation(maxPoints, minPoints, breakPointPercent, len(pageData.Students), duration, true)

	logging.LogInfo("Request processing completed",
		"session_id_length", len(sessionID),
		"parameters_processed", 3,
		"data_processed", pageData.HasStudents,
		"records_count", len(pageData.Students),
		"processing_time_ms", duration.Milliseconds(),
		"status", "success",
		"ip", ip)

	// Render template with results
	h.Templates.ExecuteTemplate(w, "index.html", pageData)
}
