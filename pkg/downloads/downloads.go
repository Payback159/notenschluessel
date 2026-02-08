package downloads

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/security"
	"github.com/payback159/notenschluessel/pkg/session"
	"github.com/xuri/excelize/v2"
)

// setCellValueSafe safely sets a cell value with error handling
func setCellValueSafe(f *excelize.File, sheet, axis string, value interface{}, sessionID, ip string) error {
	if err := f.SetCellValue(sheet, axis, value); err != nil {
		logging.LogError("Failed to set cell value", err,
			"sheet", sheet,
			"axis", axis,
			"session_id", sessionID,
			"ip", ip)
		return err
	}
	return nil
}

// setCellStyleSafe safely sets a cell style with error handling
func setCellStyleSafe(f *excelize.File, sheet, hCell, vCell string, styleID int, sessionID, ip string) {
	if err := f.SetCellStyle(sheet, hCell, vCell, styleID); err != nil {
		logging.LogError("Failed to set cell style", err,
			"sheet", sheet,
			"range", fmt.Sprintf("%s:%s", hCell, vCell),
			"session_id", sessionID,
			"ip", ip)
		// Non-critical error for styles, continue execution
	}
} // createSheetSafe safely creates a new sheet with error handling
func createSheetSafe(f *excelize.File, name, sessionID, ip string) error {
	if _, err := f.NewSheet(name); err != nil {
		logging.LogError("Failed to create sheet", err,
			"sheet_name", name,
			"session_id", sessionID,
			"ip", ip)
		return err
	}
	return nil
}

// deleteSheetSafe safely deletes a sheet with error handling
func deleteSheetSafe(f *excelize.File, name, sessionID, ip string) {
	if err := f.DeleteSheet(name); err != nil {
		logging.LogError("Failed to delete sheet", err,
			"sheet_name", name,
			"session_id", sessionID,
			"ip", ip)
		// Non-critical error, continue
	}
}

// writeResponseSafe safely writes response with error handling
func writeResponseSafe(w http.ResponseWriter, buffer *bytes.Buffer, sessionID, ip string) {
	if _, err := w.Write(buffer.Bytes()); err != nil {
		logging.LogError("Failed to write response", err,
			"session_id", sessionID,
			"ip", ip)
		// Response already started, can't send error status
	}
}

// getSessionIDFromCookie reads the session ID from an HttpOnly cookie
func getSessionIDFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// sanitizeCSVField prevents CSV injection and properly escapes fields
func sanitizeCSVField(field string) string {
	// Prevent formula injection: prefix dangerous first characters
	if len(field) > 0 {
		first := field[0]
		if first == '=' || first == '+' || first == '-' || first == '@' || first == '\t' || first == '\r' {
			field = "'" + field
		}
	}
	// Properly quote fields containing commas, quotes, or newlines
	if strings.ContainsAny(field, ",\"\n") {
		field = "\"" + strings.ReplaceAll(field, "\"", "\"\"") + "\""
	}
	return field
}

// createGradeStyles creates colored Excel styles for each grade
func createGradeStyles(f *excelize.File) map[int]int {
	gradeStyles := make(map[int]int)
	colors := map[int]string{
		1: "#c6f6d5",
		2: "#d4edda",
		3: "#fff3cd",
		4: "#ffe8cc",
		5: "#f8d7da",
	}
	for grade, color := range colors {
		style, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
			Border: []excelize.Border{
				{Type: "left", Color: "#000000", Style: 1},
				{Type: "top", Color: "#000000", Style: 1},
				{Type: "right", Color: "#000000", Style: 1},
				{Type: "bottom", Color: "#000000", Style: 1},
			},
		})
		gradeStyles[grade] = style
	}
	return gradeStyles
}

// setDownloadHeaders sets common security and caching headers for downloads
func setDownloadHeaders(w http.ResponseWriter, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

// HandleGradeScaleCSV handles CSV download of grade scale
func HandleGradeScaleCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Grade scale CSV download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasResults {
		logging.LogWarn("Grade scale download requested but no data available",
			"session_id", sessionID,
			"ip", ip,
			"session_exists", exists,
			"has_results", exists && data.HasResults)
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
	setDownloadHeaders(w, "text/csv", "notenschluessel.csv")

	// Write content to response
	writeResponseSafe(w, &buffer, sessionID, ip)

	duration := time.Since(start)
	logging.LogFileOperation("csv_download", "notenschluessel.csv", int64(buffer.Len()), duration, true,
		"session_id", sessionID,
		"ip", ip,
		"grade_count", len(data.GradeBounds))
}

// HandleStudentResultsCSV handles CSV download of student results
func HandleStudentResultsCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Student results CSV download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasStudents {
		logging.LogWarn("Student results download requested but no data available",
			"session_id", sessionID,
			"ip", ip,
			"session_exists", exists,
			"has_students", exists && data.HasStudents)
		http.Error(w, "Keine Schülerdaten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Generate CSV content
	var buffer bytes.Buffer
	buffer.WriteString("Name,Punkte,Note\n")

	for _, student := range data.Students {
		escapedName := sanitizeCSVField(student.Name)
		line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
		buffer.WriteString(line)
	}

	// Add average at the bottom
	buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))

	// Set headers for file download
	setDownloadHeaders(w, "text/csv", "schueler_ergebnisse.csv")

	// Write content to response
	writeResponseSafe(w, &buffer, sessionID, ip)

	duration := time.Since(start)
	logging.LogFileOperation("csv_download", "schueler_ergebnisse.csv", int64(buffer.Len()), duration, true,
		"session_id", sessionID,
		"ip", ip,
		"student_count", len(data.Students))
}

// HandleCombinedCSV handles CSV download of combined results
func HandleCombinedCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Combined CSV download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasResults {
		logging.LogWarn("Combined download requested but no data available",
			"session_id", sessionID,
			"ip", ip,
			"session_exists", exists,
			"has_results", exists && data.HasResults)
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Generate CSV content
	var buffer bytes.Buffer

	// Grade scale section
	buffer.WriteString("NOTENSCHLÜSSEL\n")
	buffer.WriteString("Note,Punktebereich von,Punktebereich bis\n")
	for _, bound := range data.GradeBounds {
		line := fmt.Sprintf("%d,%.1f,%.1f\n", bound.Grade, bound.LowerBound, bound.UpperBound)
		buffer.WriteString(line)
	}

	// Empty line separator
	buffer.WriteString("\n")

	// Student results section (if available)
	if data.HasStudents {
		buffer.WriteString("SCHÜLERERGEBNISSE\n")
		buffer.WriteString("Name,Punkte,Note\n")
		for _, student := range data.Students {
			escapedName := sanitizeCSVField(student.Name)
			line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
			buffer.WriteString(line)
		}
		buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))
	}

	// Set headers for file download
	setDownloadHeaders(w, "text/csv", "notenschluessel_komplett.csv")

	// Write content to response
	writeResponseSafe(w, &buffer, sessionID, ip)

	duration := time.Since(start)
	logging.LogFileOperation("csv_download", "notenschluessel_komplett.csv", int64(buffer.Len()), duration, true,
		"session_id", sessionID,
		"ip", ip,
		"has_students", data.HasStudents,
		"grade_count", len(data.GradeBounds),
		"student_count", len(data.Students))
}

// HandleGradeScaleExcel handles Excel download of grade scale
func HandleGradeScaleExcel(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Grade scale Excel download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasResults {
		logging.LogWarn("Excel grade scale download requested but no data available",
			"session_id", sessionID,
			"ip", ip)
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logging.LogError("Failed to close Excel file", err, "session_id", sessionID, "ip", ip)
		}
	}()

	sheetName := "Notenschlüssel"
	if err := createSheetSafe(f, sheetName, sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	deleteSheetSafe(f, "Sheet1", sessionID, ip)

	// Add headers
	if err := setCellValueSafe(f, sheetName, "A1", "Note", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, sheetName, "B1", "Punktebereich von", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, sheetName, "C1", "Punktebereich bis", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}

	// Define grade styles with colors
	gradeStyles := createGradeStyles(f)

	// Add data with styling
	for i, bound := range data.GradeBounds {
		row := i + 2
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("A%d", row), bound.Grade, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("B%d", row), bound.LowerBound, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("C%d", row), bound.UpperBound, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}

		if style, exists := gradeStyles[bound.Grade]; exists {
			setCellStyleSafe(f, sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style, sessionID, ip)
		}
	}

	// Set headers for file download
	setDownloadHeaders(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "notenschluessel.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		logging.LogError("Failed to write Excel file", err, "session_id", sessionID, "ip", ip)
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	logging.LogFileOperation("excel_download", "notenschluessel.xlsx", 0, duration, true,
		"session_id", sessionID,
		"ip", ip,
		"grade_count", len(data.GradeBounds))
}

// HandleStudentResultsExcel handles Excel download of student results
func HandleStudentResultsExcel(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Student results Excel download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasStudents {
		logging.LogWarn("Excel student download requested but no data available",
			"session_id", sessionID,
			"ip", ip)
		http.Error(w, "Keine Schülerdaten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logging.LogError("Failed to close Excel file", err, "session_id", sessionID, "ip", ip)
		}
	}()

	sheetName := "Schülerergebnisse"
	if err := createSheetSafe(f, sheetName, sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	deleteSheetSafe(f, "Sheet1", sessionID, ip)

	// Add headers
	if err := setCellValueSafe(f, sheetName, "A1", "Name", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, sheetName, "B1", "Punkte", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, sheetName, "C1", "Note", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}

	// Add data
	for i, student := range data.Students {
		row := i + 2
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("A%d", row), student.Name, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("B%d", row), student.Points, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, sheetName, fmt.Sprintf("C%d", row), student.Grade, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
	}

	// Add average at the bottom
	lastRow := len(data.Students) + 2
	if err := setCellValueSafe(f, sheetName, fmt.Sprintf("A%d", lastRow), "Durchschnitt", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, sheetName, fmt.Sprintf("C%d", lastRow), data.AverageGrade, sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	setDownloadHeaders(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "schueler_ergebnisse.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		logging.LogError("Failed to write Excel file to response", err, "session_id", sessionID, "ip", ip)
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	logging.LogFileOperation("excel_download", "schueler_ergebnisse.xlsx", 0, duration, true,
		"session_id", sessionID,
		"ip", ip,
		"student_count", len(data.Students))
}

// HandleCombinedExcel handles Excel download of combined results
func HandleCombinedExcel(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := getSessionIDFromCookie(r)
	ip := security.GetClientIP(r)

	logging.LogInfo("Combined Excel download requested",
		"session_id", sessionID,
		"ip", ip)

	data, exists := sessionStore.Get(sessionID)
	if !exists || !data.HasResults {
		logging.LogWarn("Excel combined download requested but no data available",
			"session_id", sessionID,
			"ip", ip)
		http.Error(w, "Keine Daten zum Herunterladen verfügbar", http.StatusBadRequest)
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logging.LogError("Failed to close Excel file", err, "session_id", sessionID, "ip", ip)
		}
	}()

	// Define grade styles
	gradeStyles := createGradeStyles(f)

	// Header style
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
	if err := createSheetSafe(f, gradeSheetName, sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	deleteSheetSafe(f, "Sheet1", sessionID, ip)

	// Add grade scale headers
	if err := setCellValueSafe(f, gradeSheetName, "A1", "Note", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, gradeSheetName, "B1", "Punktebereich von", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	if err := setCellValueSafe(f, gradeSheetName, "C1", "Punktebereich bis", sessionID, ip); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
	setCellStyleSafe(f, gradeSheetName, "A1", "C1", headerStyle, sessionID, ip)

	// Add grade scale data
	for i, bound := range data.GradeBounds {
		row := i + 2
		if err := setCellValueSafe(f, gradeSheetName, fmt.Sprintf("A%d", row), bound.Grade, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, gradeSheetName, fmt.Sprintf("B%d", row), bound.LowerBound, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, gradeSheetName, fmt.Sprintf("C%d", row), bound.UpperBound, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}

		if style, exists := gradeStyles[bound.Grade]; exists {
			setCellStyleSafe(f, gradeSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style, sessionID, ip)
		}
	}

	// Create student results sheet if students are available
	if data.HasStudents {
		studentSheetName := "Schülerergebnisse"
		if err := createSheetSafe(f, studentSheetName, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}

		// Add student headers
		if err := setCellValueSafe(f, studentSheetName, "A1", "Name", sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, studentSheetName, "B1", "Punkte", sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, studentSheetName, "C1", "Note", sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		setCellStyleSafe(f, studentSheetName, "A1", "C1", headerStyle, sessionID, ip)

		// Add student data
		for i, student := range data.Students {
			row := i + 2
			if err := setCellValueSafe(f, studentSheetName, fmt.Sprintf("A%d", row), student.Name, sessionID, ip); err != nil {
				http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
				return
			}
			if err := setCellValueSafe(f, studentSheetName, fmt.Sprintf("B%d", row), student.Points, sessionID, ip); err != nil {
				http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
				return
			}
			if err := setCellValueSafe(f, studentSheetName, fmt.Sprintf("C%d", row), student.Grade, sessionID, ip); err != nil {
				http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
				return
			}
		}

		// Add average
		lastRow := len(data.Students) + 2
		if err := setCellValueSafe(f, studentSheetName, fmt.Sprintf("A%d", lastRow), "Durchschnitt", sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
		if err := setCellValueSafe(f, studentSheetName, fmt.Sprintf("C%d", lastRow), data.AverageGrade, sessionID, ip); err != nil {
			http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
			return
		}
	}

	// Set headers for file download
	setDownloadHeaders(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "notenschluessel_komplett.xlsx")

	// Write the file to response
	if err := f.Write(w); err != nil {
		logging.LogError("Failed to write combined Excel file", err, "session_id", sessionID, "ip", ip)
		http.Error(w, "Fehler beim Erstellen der Excel-Datei", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	logging.LogFileOperation("excel_download", "notenschluessel_komplett.xlsx", 0, duration, true,
		"session_id", sessionID,
		"ip", ip,
		"has_students", data.HasStudents,
		"grade_count", len(data.GradeBounds),
		"student_count", len(data.Students))
}
