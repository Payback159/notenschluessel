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

// HandleGradeScaleCSV handles CSV download of grade scale
func HandleGradeScaleCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := r.URL.Query().Get("id")
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
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel.csv")

	// Write content to response
	w.Write(buffer.Bytes())

	duration := time.Since(start)
	logging.LogFileOperation("csv_download", "notenschluessel.csv", int64(buffer.Len()), duration, true,
		"session_id", sessionID,
		"ip", ip,
		"grade_count", len(data.GradeBounds))
}

// HandleStudentResultsCSV handles CSV download of student results
func HandleStudentResultsCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := r.URL.Query().Get("id")
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
		// Escape names that might contain commas
		escapedName := student.Name
		if strings.Contains(escapedName, ",") {
			escapedName = fmt.Sprintf("\"%s\"", escapedName)
		}
		line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
		buffer.WriteString(line)
	}

	// Add average at the bottom
	buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))

	// Set headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=schueler_ergebnisse.csv")

	// Write content to response
	w.Write(buffer.Bytes())

	duration := time.Since(start)
	logging.LogFileOperation("csv_download", "schueler_ergebnisse.csv", int64(buffer.Len()), duration, true,
		"session_id", sessionID,
		"ip", ip,
		"student_count", len(data.Students))
}

// HandleCombinedCSV handles CSV download of combined results
func HandleCombinedCSV(w http.ResponseWriter, r *http.Request, sessionStore *session.Store) {
	start := time.Now()
	sessionID := r.URL.Query().Get("id")
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
			escapedName := student.Name
			if strings.Contains(escapedName, ",") {
				escapedName = fmt.Sprintf("\"%s\"", escapedName)
			}
			line := fmt.Sprintf("%s,%.1f,%d\n", escapedName, student.Points, student.Grade)
			buffer.WriteString(line)
		}
		buffer.WriteString(fmt.Sprintf("Durchschnitt,,%.2f\n", data.AverageGrade))
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel_komplett.csv")

	// Write content to response
	w.Write(buffer.Bytes())

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
	sessionID := r.URL.Query().Get("id")
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
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")

	// Add headers
	f.SetCellValue(sheetName, "A1", "Note")
	f.SetCellValue(sheetName, "B1", "Punktebereich von")
	f.SetCellValue(sheetName, "C1", "Punktebereich bis")

	// Define grade styles with colors
	gradeStyles := make(map[int]int)

	style1, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#c6f6d5"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[1] = style1

	style2, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#d4edda"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[2] = style2

	style3, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#fff3cd"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[3] = style3

	style4, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#ffe8cc"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[4] = style4

	style5, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f8d7da"}, Pattern: 1},
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
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), bound.Grade)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), bound.LowerBound)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), bound.UpperBound)

		if style, exists := gradeStyles[bound.Grade]; exists {
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style)
		}
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel.xlsx")

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
	sessionID := r.URL.Query().Get("id")
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
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")

	// Add headers
	f.SetCellValue(sheetName, "A1", "Name")
	f.SetCellValue(sheetName, "B1", "Punkte")
	f.SetCellValue(sheetName, "C1", "Note")

	// Add data
	for i, student := range data.Students {
		row := i + 2
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
	sessionID := r.URL.Query().Get("id")
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
	gradeStyles := make(map[int]int)

	style1, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#c6f6d5"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[1] = style1

	style2, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#d4edda"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[2] = style2

	style3, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#fff3cd"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[3] = style3

	style4, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#ffe8cc"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[4] = style4

	style5, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#f8d7da"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	gradeStyles[5] = style5

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
	f.NewSheet(gradeSheetName)
	f.DeleteSheet("Sheet1")

	// Add grade scale headers
	f.SetCellValue(gradeSheetName, "A1", "Note")
	f.SetCellValue(gradeSheetName, "B1", "Punktebereich von")
	f.SetCellValue(gradeSheetName, "C1", "Punktebereich bis")
	f.SetCellStyle(gradeSheetName, "A1", "C1", headerStyle)

	// Add grade scale data
	for i, bound := range data.GradeBounds {
		row := i + 2
		f.SetCellValue(gradeSheetName, fmt.Sprintf("A%d", row), bound.Grade)
		f.SetCellValue(gradeSheetName, fmt.Sprintf("B%d", row), bound.LowerBound)
		f.SetCellValue(gradeSheetName, fmt.Sprintf("C%d", row), bound.UpperBound)

		if style, exists := gradeStyles[bound.Grade]; exists {
			f.SetCellStyle(gradeSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), style)
		}
	}

	// Create student results sheet if students are available
	if data.HasStudents {
		studentSheetName := "Schülerergebnisse"
		f.NewSheet(studentSheetName)

		// Add student headers
		f.SetCellValue(studentSheetName, "A1", "Name")
		f.SetCellValue(studentSheetName, "B1", "Punkte")
		f.SetCellValue(studentSheetName, "C1", "Note")
		f.SetCellStyle(studentSheetName, "A1", "C1", headerStyle)

		// Add student data
		for i, student := range data.Students {
			row := i + 2
			f.SetCellValue(studentSheetName, fmt.Sprintf("A%d", row), student.Name)
			f.SetCellValue(studentSheetName, fmt.Sprintf("B%d", row), student.Points)
			f.SetCellValue(studentSheetName, fmt.Sprintf("C%d", row), student.Grade)
		}

		// Add average
		lastRow := len(data.Students) + 2
		f.SetCellValue(studentSheetName, fmt.Sprintf("A%d", lastRow), "Durchschnitt")
		f.SetCellValue(studentSheetName, fmt.Sprintf("C%d", lastRow), data.AverageGrade)
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=notenschluessel_komplett.xlsx")

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
