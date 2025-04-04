package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
)

type Student struct {
	Name   string
	Points float64
	Grade  int
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
	ErrorMessage       string
}

type GradeBound struct {
	Grade      int
	LowerBound float64
	UpperBound float64
}

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
		pageData.ErrorMessage = "Ungültige Eingabe für maximale Punktzahl"
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.MaxPoints = maxPoints

	minPoints, err := strconv.ParseFloat(minPointsStr, 64)
	if err != nil {
		pageData.ErrorMessage = "Ungültige Eingabe für minimale Punktevergabe"
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.MinPoints = minPoints

	breakPointPercent, err := strconv.ParseFloat(breakPointPercentStr, 64)
	if err != nil {
		pageData.ErrorMessage = "Ungültige Eingabe für Knickpunkt"
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}
	pageData.BreakPointPercent = breakPointPercent

	// Validate input
	if float64(maxPoints) <= minPoints {
		pageData.ErrorMessage = "Maximale Punktzahl muss größer als minimale Punktevergabe sein"
		templates.ExecuteTemplate(w, "index.html", pageData)
		return
	}

	// Calculate breaking point in points
	breakPointPercent = math.Max(0, math.Min(100, breakPointPercent))
	breakPointPoints := minPoints + float64(maxPoints-int(minPoints))*(breakPointPercent/100)

	// Round the breaking point to the nearest multiple of minPoints
	if minPoints > 0 {
		breakPointPoints = math.Round(breakPointPoints/minPoints) * minPoints
	}

	// Ensure breaking point is within valid range
	if breakPointPoints <= minPoints {
		breakPointPoints = minPoints + minPoints
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
	lowerBound5 := minPoints

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
			pageData.ErrorMessage = "Fehler bei der Verarbeitung des CSV-Files"
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Copy uploaded file to temp file
		_, err = io.Copy(tempFile, file)
		if err != nil {
			pageData.ErrorMessage = "Fehler beim Speichern des CSV-Files"
			templates.ExecuteTemplate(w, "index.html", pageData)
			return
		}

		// Load students from temp file
		students, err := loadStudentsFromCSV(tempFile.Name())
		if err != nil {
			pageData.ErrorMessage = fmt.Sprintf("Fehler beim Laden des CSV-Files: %v", err)
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

		pageData.Students = students
		pageData.HasStudents = len(students) > 0

		if pageData.HasStudents {
			pageData.AverageGrade = gradeSum / float64(len(students))
		}

		pageData.ErrorMessage = fmt.Sprintf("CSV-Datei '%s' erfolgreich geladen", handler.Filename)
	}

	// Render template with results
	templates.ExecuteTemplate(w, "index.html", pageData)
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
