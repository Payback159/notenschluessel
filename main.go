package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
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
}

type GradeBound struct {
	Grade      int
	LowerBound float64
	UpperBound float64
}

// Session storage to keep track of calculation results
var sessionResults = make(map[string]PageData)

func main() {
	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Register handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if r.Method == http.MethodGet {
			templates.ExecuteTemplate(w, "index.html", PageData{})
			return
		}

		if r.Method == http.MethodPost {
			handleCalculation(w, r, templates)
			return
		}
	})

	// Add download handlers
	http.HandleFunc("/download/grade-scale", handleGradeScaleDownload)
	http.HandleFunc("/download/student-results", handleStudentResultsDownload)
	http.HandleFunc("/download/combined", handleCombinedDownload)

	// Add Excel download handlers
	http.HandleFunc("/download/grade-scale-excel", handleGradeScaleExcelDownload)
	http.HandleFunc("/download/student-results-excel", handleStudentResultsExcelDownload)
	http.HandleFunc("/download/combined-excel", handleCombinedExcelDownload)

	// Start server
	fmt.Println("Server läuft auf http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleCalculation(w http.ResponseWriter, r *http.Request, templates *template.Template) {
	// Parse form data
	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
	if err != nil {
		http.Error(w, "Fehler beim Parsen des Formulars", http.StatusBadRequest)
		return
	}

	pageData := PageData{}

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

		// Create temporary file to store the upload
		tempFile, err := os.CreateTemp("", "upload-*.csv")
		if err != nil {
			pageData.Message = &Message{
				Text: "Fehler bei der Verarbeitung des CSV-Files",
				Type: MessageTypeError,
			}
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}
		defer os.Remove(tempFile.Name())
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
			Text: fmt.Sprintf("CSV-Datei '%s' erfolgreich geladen", handler.Filename),
			Type: MessageTypeSuccess,
		}
	}

	// Store results in session for later download
	sessionID := generateSessionID()
	sessionResults[sessionID] = pageData

	// Add session ID to the template data
	pageData.SessionID = sessionID

	// Render template with results
	templates.ExecuteTemplate(w, "index.html", pageData)
}

// Function to generate a simple session ID
func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
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
