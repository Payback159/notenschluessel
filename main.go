package main

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/csrf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/time/rate"
)

type Student struct {
	Name   string
	Points float64
	Grade  int
}

// MessageType definiert die erlaubten Nachrichtentypen
type MessageType string

const (
	MessageTypeError   MessageType = "error"
	MessageTypeInfo    MessageType = "info"
	MessageTypeWarning MessageType = "warning"
	MessageTypeSuccess MessageType = "success"
)

type Message struct {
	Text string
	Type MessageType
}

type PageData struct {
	MaxPoints          int
	MinPoints          float64
	BreakPointPercent  float64
	GradeBounds        []GradeBound
	Students           []Student
	AverageGrade       float64
	HasResults         bool
	HasStudents        bool
	CalculationSuccess bool
	Message            *Message
	SessionID          string
	GitHubConfigured   bool
	CSRFField          template.HTML
}

type GradeBound struct {
	Grade      int
	LowerBound float64
	UpperBound float64
}

// Bug report structures
type BugReport struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Steps          string `json:"steps"`
	Expected       string `json:"expected"`
	Browser        string `json:"browser"`
	OS             string `json:"os"`
	MaxPoints      string `json:"maxPoints"`
	MinPoints      string `json:"minPoints"`
	BreakPoint     string `json:"breakPoint"`
	CSVUsed        string `json:"csvUsed"`
	AdditionalInfo string `json:"additionalInfo"`
}

type BugReportResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GitHubIssue struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

type GitHubClient struct {
	token string
	repo  string
}

// Constants for security limits
const (
	maxFileSize     = 5 * 1024 * 1024 // 5MB
	maxStudents     = 1000
	sessionTimeout  = time.Hour
	cleanupInterval = 15 * time.Minute
	rateLimit       = 10              // requests per minute
	rateLimitPeriod = 6 * time.Second // 60s / 10 requests
)

// Secure session storage
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

type SessionData struct {
	PageData  PageData
	CreatedAt time.Time
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*SessionData),
	}
	store.startCleanup()
	return store
}

func (s *SessionStore) Set(id string, data PageData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = &SessionData{
		PageData:  data,
		CreatedAt: time.Now(),
	}
}

func (s *SessionStore) Get(id string) (PageData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return PageData{}, false
	}

	// Check if session expired
	if time.Since(session.CreatedAt) > sessionTimeout {
		return PageData{}, false
	}

	return session.PageData, true
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *SessionStore) startCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	go func() {
		for range ticker.C {
			s.mu.Lock()
			for id, session := range s.sessions {
				if time.Since(session.CreatedAt) > sessionTimeout {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}()
}

// Rate limiter
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(rateLimitPeriod), rateLimit)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// Global instances
var (
	sessionStore = NewSessionStore()
	rateLimiter  = NewRateLimiter()
)

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxy)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP if multiple are present
		if idx := strings.Index(forwarded, ","); idx != -1 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func validateCSVFile(file multipart.File, header *multipart.FileHeader) error {
	if header.Size > maxFileSize {
		return fmt.Errorf("Datei zu groß (max. 5MB)")
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		return fmt.Errorf("Nur CSV-Dateien erlaubt")
	}

	return nil
}

func sanitizeName(name string) string {
	// Remove control characters and limit length
	name = strings.TrimSpace(name)
	if len(name) > 100 {
		name = name[:100]
	}
	// Remove any HTML/Script tags
	name = strings.ReplaceAll(name, "<", "&lt;")
	name = strings.ReplaceAll(name, ">", "&gt;")
	name = strings.ReplaceAll(name, "\"", "&quot;")
	name = strings.ReplaceAll(name, "'", "&#39;")
	return name
}

// Middleware functions
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")

		next(w, r)
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		limiter := rateLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Zu viele Anfragen. Bitte versuchen Sie es später erneut.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// Security utility functions
func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// Removed deprecated sessionResults map. All session data is now managed using SessionStore.

// Check if GitHub is configured
func isGitHubConfigured() bool {
	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPO")
	return token != "" && repo != ""
}

// GitHub client for creating issues
func newGitHubClient() *GitHubClient {
	return &GitHubClient{
		token: os.Getenv("GITHUB_TOKEN"),
		repo:  os.Getenv("GITHUB_REPO"),
	}
}

func (g *GitHubClient) isConfigured() bool {
	return g.token != "" && g.repo != ""
}

func (g *GitHubClient) createIssue(report BugReport) error {
	if !g.isConfigured() {
		return fmt.Errorf("GitHub client not configured")
	}

	// Don't log the token - only log repo for debugging
	log.Printf("Creating GitHub issue in repo: %s", g.repo)

	// Create issue body from bug report
	body := fmt.Sprintf(`## 🐛 Fehlerbeschreibung
%s

## 🔄 Schritte zur Reproduktion
%s

## ✅ Erwartetes Verhalten
%s

## 💻 Systeminformationen
- Browser: %s
- Betriebssystem: %s

## 📋 Eingabedaten
- Maximale Punktzahl: %s
- Punkteschrittweite: %s
- Knickpunkt: %s
- CSV-Datei verwendet: %s

## 📝 Zusätzlicher Kontext
%s

---
*Automatisch erstellt über das Bug-Report-Formular*`,
		report.Description, report.Steps, report.Expected,
		report.Browser, report.OS,
		report.MaxPoints, report.MinPoints, report.BreakPoint, report.CSVUsed,
		report.AdditionalInfo)

	issue := GitHubIssue{
		Title:  "[BUG] " + report.Title,
		Body:   body,
		Labels: []string{"bug", "user-report"},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(issue)
	if err != nil {
		return err
	}

	// Create HTTP request
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", g.repo)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

func main() {
	// Check for health check flag
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		// Simple health check - just exit with 0 if we can start
		fmt.Println("OK")
		os.Exit(0)
	}

	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Generate CSRF key
	csrfKey := make([]byte, 32)
	if _, err := rand.Read(csrfKey); err != nil {
		log.Fatal("Failed to generate CSRF key:", err)
	}

	// Configure CSRF protection (only in production)
	var csrfMiddleware func(http.Handler) http.Handler
	if os.Getenv("ENV") == "production" {
		csrfMiddleware = csrf.Protect(csrfKey, csrf.Secure(true))
	} else {
		csrfMiddleware = csrf.Protect(csrfKey, csrf.Secure(false))
	}

	// Create multiplexer
	mux := http.NewServeMux()

	// Register handlers with middleware
	mux.HandleFunc("/", securityHeaders(rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if r.Method == http.MethodGet {
			pageData := PageData{
				GitHubConfigured: isGitHubConfigured(),
				CSRFField:        csrf.TemplateField(r),
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		if r.Method == http.MethodPost {
			handleCalculation(w, r, templates)
			return
		}
	})))

	// Add download handlers (no rate limiting for downloads)
	mux.HandleFunc("/download/grade-scale", securityHeaders(handleGradeScaleDownload))
	mux.HandleFunc("/download/student-results", securityHeaders(handleStudentResultsDownload))
	mux.HandleFunc("/download/combined", securityHeaders(handleCombinedDownload))

	// Add Excel download handlers
	mux.HandleFunc("/download/grade-scale-excel", securityHeaders(handleGradeScaleExcelDownload))
	mux.HandleFunc("/download/student-results-excel", securityHeaders(handleStudentResultsExcelDownload))
	mux.HandleFunc("/download/combined-excel", securityHeaders(handleCombinedExcelDownload))

	// Add bug report handler with rate limiting
	mux.HandleFunc("/api/bug-report", securityHeaders(rateLimitMiddleware(handleBugReport)))

	// Apply CSRF middleware to the entire mux
	handler := csrfMiddleware(mux)

	// Start server
	fmt.Println("Server läuft auf http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func handleCalculation(w http.ResponseWriter, r *http.Request, templates *template.Template) {
	// Parse form data
	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
	if err != nil {
		http.Error(w, "Fehler beim Parsen des Formulars", http.StatusBadRequest)
		return
	}

	pageData := PageData{
		GitHubConfigured: isGitHubConfigured(),
		CSRFField:        csrf.TemplateField(r),
	}

	// Get form values
	maxPointsStr := r.FormValue("maxPoints")
	minPointsStr := r.FormValue("minPoints")
	breakPointPercentStr := r.FormValue("breakPointPercent")

	// Convert string values to appropriate types
	maxPoints, err := strconv.Atoi(maxPointsStr)
	if err != nil {
		pageData.Message = &Message{
			Text: "Ungültige Eingabe für maximale Punktzahl",
			Type: MessageTypeError,
		}
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.MaxPoints = maxPoints

	minPoints, err := strconv.ParseFloat(minPointsStr, 64)
	if err != nil {
		pageData.Message = &Message{
			Text: "Ungültige Eingabe für Punkteschrittweite",
			Type: MessageTypeError,
		}
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.MinPoints = minPoints

	breakPointPercent, err := strconv.ParseFloat(breakPointPercentStr, 64)
	if err != nil {
		pageData.Message = &Message{
			Text: "Ungültige Eingabe für Knickpunkt",
			Type: MessageTypeError,
		}
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.BreakPointPercent = breakPointPercent

	// Validate input
	if float64(maxPoints) <= minPoints {
		pageData.Message = &Message{
			Text: "Maximale Punktzahl muss größer als Punkteschrittweite sein",
			Type: MessageTypeError,
		}
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}

	// Calculate breaking point in points
	breakPointPercent = math.Max(0, math.Min(100, breakPointPercent))
	breakPointPoints := float64(maxPoints) * (breakPointPercent / 100)

	// Round the breaking point to the nearest multiple of minPoints
	if minPoints > 0 {
		// Round down to ensure the boundary is inclusive for grade 4
		breakPointPoints = math.Floor(breakPointPoints/minPoints) * minPoints
	}

	// Ensure breaking point is within valid range
	if breakPointPoints < minPoints {
		breakPointPoints = minPoints
	}
	if breakPointPoints >= float64(maxPoints) {
		breakPointPoints = float64(maxPoints) - minPoints
	}

	// Calculate the range available for grades 1-4
	rangeForGrades1to4 := float64(maxPoints) - breakPointPoints + minPoints

	// Calculate the size of each grade range
	pointsPerGrade := rangeForGrades1to4 / 4.0

	// Calculate and round the starting point for each grade to respect minPoints as step size
	lowerBound1 := roundToStepSize(breakPointPoints+3*pointsPerGrade, minPoints)
	lowerBound2 := roundToStepSize(breakPointPoints+2*pointsPerGrade, minPoints)
	lowerBound3 := roundToStepSize(breakPointPoints+pointsPerGrade, minPoints)
	lowerBound4 := breakPointPoints
	lowerBound5 := 0.0 // Set to 0 as the absolute minimum for grade 5

	// Calculate the upper bound of each grade
	upperBound1 := float64(maxPoints)
	upperBound2 := lowerBound1 - minPoints
	upperBound3 := lowerBound2 - minPoints
	upperBound4 := lowerBound3 - minPoints
	upperBound5 := lowerBound4 - minPoints

	// Store grade bounds for display
	pageData.GradeBounds = []GradeBound{
		{Grade: 1, LowerBound: lowerBound1, UpperBound: upperBound1},
		{Grade: 2, LowerBound: lowerBound2, UpperBound: upperBound2},
		{Grade: 3, LowerBound: lowerBound3, UpperBound: upperBound3},
		{Grade: 4, LowerBound: lowerBound4, UpperBound: upperBound4},
		{Grade: 5, LowerBound: lowerBound5, UpperBound: upperBound5},
	}

	pageData.HasResults = true
	pageData.CalculationSuccess = true

	// Process CSV file if uploaded
	file, handler, err := r.FormFile("csvFile")
	if err == nil && file != nil {
		defer file.Close()

		// Validate CSV file
		if err := validateCSVFile(file, handler); err != nil {
			pageData.Message = &Message{
				Text: fmt.Sprintf("Fehler beim Validieren der CSV-Datei: %v", err),
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		// Create secure temp directory
		tempDir, err := os.MkdirTemp("", "notenschluessel-*")
		if err != nil {
			pageData.Message = &Message{
				Text: "Fehler bei der Verarbeitung des CSV-Files",
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}
		defer os.RemoveAll(tempDir) // Clean up entire directory

		// Create temp file in secure directory
		tempFile, err := os.CreateTemp(tempDir, "upload-*.csv")
		if err != nil {
			pageData.Message = &Message{
				Text: "Fehler bei der Verarbeitung des CSV-Files",
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}
		defer tempFile.Close()

		// Copy uploaded file to temp file
		_, err = io.Copy(tempFile, file)
		if err != nil {
			pageData.Message = &Message{
				Text: "Fehler beim Speichern des CSV-Files",
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		// Load students from temp file
		students, err := loadStudentsFromCSV(tempFile.Name())
		if err != nil {
			pageData.Message = &Message{
				Text: fmt.Sprintf("Fehler beim Laden des CSV-Files: %v", err),
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		// Validate number of students
		if len(students) > maxStudents {
			pageData.Message = &Message{
				Text: fmt.Sprintf("Zu viele Schüler in der CSV-Datei (max. %d)", maxStudents),
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		// Sanitize student names
		for i := range students {
			students[i].Name = sanitizeName(students[i].Name)
		}

		// Calculate grades for students
		var gradeSum float64
		for i := range students {
			students[i].Grade = calculateGrade(students[i].Points,
				lowerBound1, lowerBound2, lowerBound3, lowerBound4, lowerBound5)
			gradeSum += float64(students[i].Grade)
		}

		// Sort students alphabetically by name
		sort.Slice(students, func(i, j int) bool {
			return students[i].Name < students[j].Name
		})

		pageData.Students = students
		pageData.HasStudents = len(students) > 0

		if pageData.HasStudents {
			pageData.AverageGrade = gradeSum / float64(len(students))
		}

		pageData.Message = &Message{
			Text: fmt.Sprintf("CSV-Datei '%s' erfolgreich geladen", sanitizeName(handler.Filename)),
			Type: MessageTypeSuccess,
		}
	}

	// Store results in secure session store
	sessionID := generateSessionID()
	sessionStore.Set(sessionID, pageData)

	// Also store in legacy sessionResults for backwards compatibility with download handlers
	sessionResults[sessionID] = pageData

	// Add session ID to the template data
	pageData.SessionID = sessionID

	// Render template with results
	templates.ExecuteTemplate(w, "index.html", pageData)
}

// Handler for grade scale download
func handleGradeScaleDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasResults {
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Generate CSV content
	var buffer bytes.Buffer
	buffer.WriteString("Note,Punktebereich von,Punktebereich bis\n")

	for _, bound := range data.GradeBounds {
		line := fmt.Sprintf("%d,%.1f,%.1f\n", bound.Grade, bound.LowerBound, bound.UpperBound)
		buffer.WriteString(line)
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel.csv")

	// Write content to response
	w.Write(buffer.Bytes())
}

// Handler for student results download
func handleStudentResultsDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasStudents {
		http.Error(w, "Keine Schülerdaten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Generate CSV content
	var buffer bytes.Buffer
	buffer.WriteString("Name,Punkte,Note\n")

	for _, student := range data.Students {
		// Escape names that might contain commas
		escapedName := student.Name
		if strings.Contains(escapedName, ",") {
			escapedName = fmt.Sprintf("\"%s\"", escapedName)
		}

		line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
		buffer.WriteString(line)
	}

	// Add average at the end
	buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))

	// Set headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=schueler_ergebnisse.csv")

	// Write content to response
	w.Write(buffer.Bytes())
}

// Handler for combined download of grade scale and student results
func handleCombinedDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasResults {
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Generate combined CSV content
	var buffer bytes.Buffer

	// Add grade scale section
	buffer.WriteString("NOTENSCHLÜSSEL\n")
	buffer.WriteString("Note,Punktebereich von,Punktebereich bis\n")

	for _, bound := range data.GradeBounds {
		line := fmt.Sprintf("%d,%.1f,%.1f\n", bound.Grade, bound.LowerBound, bound.UpperBound)
		buffer.WriteString(line)
	}

	// Add empty line as separator between sections
	buffer.WriteString("\n")

	// Add student results section if available
	if data.HasStudents {
		buffer.WriteString("SCHÜLERERGEBNISSE\n")
		buffer.WriteString("Name,Punkte,Note\n")

		for _, student := range data.Students {
			// Escape names that might contain commas
			escapedName := student.Name
			if strings.Contains(escapedName, ",") {
				escapedName = fmt.Sprintf("\"%s\"", escapedName)
			}

			line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
			buffer.WriteString(line)
		}

		// Add average at the end
		buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel_und_ergebnisse.csv")

	// Write content to response
	w.Write(buffer.Bytes())
}

// Handler for grade scale Excel download
func handleGradeScaleExcelDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasResults {
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create a new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Create a new sheet
	sheetName := "Notenschlüssel"
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1") // Remove default sheet

	// Add headers
	f.SetCellValue(sheetName, "A1", "Note")
	f.SetCellValue(sheetName, "B1", "Punktebereich von")
	f.SetCellValue(sheetName, "C1", "Punktebereich bis")

	// Style for headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f2f2f2"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	f.SetCellStyle(sheetName, "A1", "C1", headerStyle)

	// Define cell styles for each grade
	gradeStyles := make(map[int]int)

	style1, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#c6f6d5"}, Pattern: 1}, // Light green
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[1] = style1

	style2, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#d4edda"}, Pattern: 1}, // Pale green
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[2] = style2

	style3, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#fff3cd"}, Pattern: 1}, // Light yellow
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[3] = style3

	style4, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#ffe8cc"}, Pattern: 1}, // Light orange
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[4] = style4

	style5, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f8d7da"}, Pattern: 1}, // Light red
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[5] = style5

	// Add data with styling
	for i, bound := range data.GradeBounds {
		row := i + 2 // Start from row 2 (after headers)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), bound.Grade)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), bound.LowerBound)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), bound.UpperBound)

		// Apply the style based on grade
		if style, exists := gradeStyles[bound.Grade]; exists {
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style)
		}
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}
}

// Handler for student results Excel download
func handleStudentResultsExcelDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasStudents {
		http.Error(w, "Keine Schülerdaten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create a new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Create a new sheet
	sheetName := "Schülerergebnisse"
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1") // Remove default sheet

	// Add headers
	f.SetCellValue(sheetName, "A1", "Name")
	f.SetCellValue(sheetName, "B1", "Punkte")
	f.SetCellValue(sheetName, "C1", "Note")

	// Add data
	for i, student := range data.Students {
		row := i + 2 // Start from row 2 (after headers)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), student.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), student.Points)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), student.Grade)
	}

	// Add average at the bottom
	lastRow := len(data.Students) + 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", lastRow), "Durchschnitt")
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", lastRow), data.AverageGrade)

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=schueler_ergebnisse.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}
}

// Handler for combined Excel download
func handleCombinedExcelDownload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	data, exists := sessionResults[sessionID]

	if !exists || !data.HasResults {
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create a new Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Define cell styles for each grade
	gradeStyles := make(map[int]int)

	style1, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#c6f6d5"}, Pattern: 1}, // Light green
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[1] = style1

	style2, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#d4edda"}, Pattern: 1}, // Pale green
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[2] = style2

	style3, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#fff3cd"}, Pattern: 1}, // Light yellow
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[3] = style3

	style4, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#ffe8cc"}, Pattern: 1}, // Light orange
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[4] = style4

	style5, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f8d7da"}, Pattern: 1}, // Light red
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[5] = style5

	// Style for headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f2f2f2"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})

	// Create grade scale sheet
	gradeSheetName := "Notenschlüssel"
	f.NewSheet(gradeSheetName)
	f.DeleteSheet("Sheet1") // Remove default sheet

	// Add headers to grade scale sheet
	f.SetCellValue(gradeSheetName, "A1", "Note")
	f.SetCellValue(gradeSheetName, "B1", "Punktebereich von")
	f.SetCellValue(gradeSheetName, "C1", "Punktebereich bis")
	f.SetCellStyle(gradeSheetName, "A1", "C1", headerStyle)

	// Add grade data with styling
	for i, bound := range data.GradeBounds {
		row := i + 2 // Start from row 2 (after headers)
		f.SetCellValue(gradeSheetName, fmt.Sprintf("A%d", row), bound.Grade)
		f.SetCellValue(gradeSheetName, fmt.Sprintf("B%d", row), bound.LowerBound)
		f.SetCellValue(gradeSheetName, fmt.Sprintf("C%d", row), bound.UpperBound)

		// Apply the style based on grade
		if style, exists := gradeStyles[bound.Grade]; exists {
			f.SetCellStyle(gradeSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style)
		}
	}

	// Create student results sheet if data is available
	if data.HasStudents {
		studentSheetName := "Schülerergebnisse"
		f.NewSheet(studentSheetName)

		// Add headers to student results sheet
		f.SetCellValue(studentSheetName, "A1", "Name")
		f.SetCellValue(studentSheetName, "B1", "Punkte")
		f.SetCellValue(studentSheetName, "C1", "Note")
		f.SetCellStyle(studentSheetName, "A1", "C1", headerStyle)

		// Add student data with styling
		for i, student := range data.Students {
			row := i + 2 // Start from row 2 (after headers)
			f.SetCellValue(studentSheetName, fmt.Sprintf("A%d", row), student.Name)
			f.SetCellValue(studentSheetName, fmt.Sprintf("B%d", row), student.Points)
			f.SetCellValue(studentSheetName, fmt.Sprintf("C%d", row), student.Grade)

			// Apply the style based on grade
			if style, exists := gradeStyles[student.Grade]; exists {
				f.SetCellStyle(studentSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style)
			}
		}

		// Add average at the bottom with special formatting
		lastRow := len(data.Students) + 2
		f.SetCellValue(studentSheetName, fmt.Sprintf("A%d", lastRow), "Durchschnitt")
		f.SetCellValue(studentSheetName, fmt.Sprintf("C%d", lastRow), data.AverageGrade)

		// Create a summary style
		summaryStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Border: []excelize.Border{
				{Type: "top", Color: "#000000", Style: 1},
			},
		})
		f.SetCellStyle(studentSheetName, fmt.Sprintf("A%d", lastRow), fmt.Sprintf("C%d", lastRow), summaryStyle)
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel_und_ergebnisse.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}
}

// Function to round a value to the nearest multiple of step size
func roundToStepSize(value, stepSize float64) float64 {
	if stepSize <= 0 {
		return value
	}
	return math.Round(value/stepSize) * stepSize
}

// Function to load students from a CSV file
func loadStudentsFromCSV(filePath string) ([]Student, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ',' // Set delimiter to comma

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("fehler beim Lesen der CSV-Kopfzeile: %v", err)
	}

	var students []Student
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fehler beim Lesen einer CSV-Zeile: %v", err)
		}

		// Expecting each row to have at least two columns: name and points
		if len(record) < 2 {
			continue
		}

		name := record[0]
		points, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue // Skip records with invalid point values
		}

		students = append(students, Student{
			Name:   name,
			Points: points,
		})
	}

	return students, nil
}

// Function to calculate grade based on points
func calculateGrade(points, lowerBound1, lowerBound2, lowerBound3, lowerBound4, lowerBound5 float64) int {
	switch {
	case points >= lowerBound1:
		return 1
	case points >= lowerBound2:
		return 2
	case points >= lowerBound3:
		return 3
	case points >= lowerBound4:
		return 4
	default:
		return 5
	}
}

// Handler for bug reports
func handleBugReport(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set CORS headers for frontend
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Limit request body size (1MB)
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	// Parse JSON request
	var bugReport BugReport
	err := json.NewDecoder(r.Body).Decode(&bugReport)
	if err != nil {
		response := BugReportResponse{
			Success: false,
			Message: "Ungültige Anfrage: " + err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate and sanitize input fields
	bugReport.Title = strings.TrimSpace(bugReport.Title)
	bugReport.Description = strings.TrimSpace(bugReport.Description)
	bugReport.Steps = strings.TrimSpace(bugReport.Steps)
	bugReport.Expected = strings.TrimSpace(bugReport.Expected)
	bugReport.Browser = strings.TrimSpace(bugReport.Browser)
	bugReport.OS = strings.TrimSpace(bugReport.OS)
	bugReport.AdditionalInfo = strings.TrimSpace(bugReport.AdditionalInfo)

	// Validate required fields
	if bugReport.Title == "" || bugReport.Description == "" {
		response := BugReportResponse{
			Success: false,
			Message: "Titel und Beschreibung sind erforderlich",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate field lengths
	if len(bugReport.Title) > 200 {
		response := BugReportResponse{
			Success: false,
			Message: "Titel ist zu lang (max. 200 Zeichen)",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(bugReport.Description) > 2000 {
		response := BugReportResponse{
			Success: false,
			Message: "Beschreibung ist zu lang (max. 2000 Zeichen)",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Sanitize all text fields
	bugReport.Title = sanitizeName(bugReport.Title)
	bugReport.Description = sanitizeName(bugReport.Description)
	bugReport.Steps = sanitizeName(bugReport.Steps)
	bugReport.Expected = sanitizeName(bugReport.Expected)
	bugReport.Browser = sanitizeName(bugReport.Browser)
	bugReport.OS = sanitizeName(bugReport.OS)
	bugReport.AdditionalInfo = sanitizeName(bugReport.AdditionalInfo)

	// Check if GitHub is configured
	if !isGitHubConfigured() {
		response := BugReportResponse{
			Success: false,
			Message: "Bug-Report-Funktion ist nicht verfügbar",
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create GitHub client and try to create issue
	gitHubClient := newGitHubClient()

	// Try to create GitHub issue
	err = gitHubClient.createIssue(bugReport)
	if err != nil {
		log.Printf("Failed to create GitHub issue: %v", err)
		response := BugReportResponse{
			Success: false,
			Message: "Fehler beim Übermitteln des Bug-Reports. Bitte versuchen Sie es später erneut.",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success
	response := BugReportResponse{
		Success: true,
		Message: "Bug-Report wurde erfolgreich übermittelt. Vielen Dank!",
	}
	json.NewEncoder(w).Encode(response)
}
