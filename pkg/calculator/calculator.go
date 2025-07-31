package calculator

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"github.com/payback159/notenschluessel/pkg/security"
)

// CalculateGradeBounds calculates the grade boundaries based on parameters
func CalculateGradeBounds(maxPoints int, minPoints, breakPointPercent float64) []models.GradeBound {
	// Calculate breakpoint in absolute points
	breakPointAbsolute := float64(maxPoints) * (breakPointPercent / 100.0)

	// Calculate points needed for each grade
	// Grade 1 (very good): 100% to ~85% of max points
	lowerBound1 := float64(maxPoints) * 0.85

	// Grade 2 (good): ~85% to breakpoint
	lowerBound2 := breakPointAbsolute

	// Grade 3 (satisfactory): breakpoint to ~60% of breakpoint
	lowerBound3 := breakPointAbsolute * 0.6

	// Grade 4 (adequate): ~60% of breakpoint to ~33% of breakpoint
	lowerBound4 := breakPointAbsolute * 0.33

	// Grade 5 (inadequate): below ~33% of breakpoint
	lowerBound5 := 0.0

	// Round to nearest minPoints increment
	lowerBound1 = math.Round(lowerBound1/minPoints) * minPoints
	lowerBound2 = math.Round(lowerBound2/minPoints) * minPoints
	lowerBound3 = math.Round(lowerBound3/minPoints) * minPoints
	lowerBound4 = math.Round(lowerBound4/minPoints) * minPoints

	gradeBounds := []models.GradeBound{
		{Grade: 1, LowerBound: lowerBound1, UpperBound: float64(maxPoints)},
		{Grade: 2, LowerBound: lowerBound2, UpperBound: lowerBound1 - minPoints},
		{Grade: 3, LowerBound: lowerBound3, UpperBound: lowerBound2 - minPoints},
		{Grade: 4, LowerBound: lowerBound4, UpperBound: lowerBound3 - minPoints},
		{Grade: 5, LowerBound: lowerBound5, UpperBound: lowerBound4 - minPoints},
	}

	logging.LogInfo("Grade bounds calculated",
		"max_points", maxPoints,
		"min_points", minPoints,
		"break_point_percent", breakPointPercent,
		"break_point_absolute", breakPointAbsolute,
		"grade_1_lower", lowerBound1,
		"grade_2_lower", lowerBound2,
		"grade_3_lower", lowerBound3,
		"grade_4_lower", lowerBound4)

	return gradeBounds
}

// CalculateGrade determines the grade based on points and grade boundaries
func CalculateGrade(points, lowerBound1, lowerBound2, lowerBound3, lowerBound4, lowerBound5 float64) int {
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

// ProcessStudents calculates grades for a list of students
func ProcessStudents(students []models.Student, gradeBounds []models.GradeBound) []models.Student {
	if len(gradeBounds) < 5 {
		logging.LogError("Insufficient grade bounds for student processing", fmt.Errorf("need 5 grade bounds, got %d", len(gradeBounds)))
		return students
	}

	lowerBound1 := gradeBounds[0].LowerBound
	lowerBound2 := gradeBounds[1].LowerBound
	lowerBound3 := gradeBounds[2].LowerBound
	lowerBound4 := gradeBounds[3].LowerBound
	lowerBound5 := gradeBounds[4].LowerBound

	for i := range students {
		students[i].Grade = CalculateGrade(students[i].Points,
			lowerBound1, lowerBound2, lowerBound3, lowerBound4, lowerBound5)
	}

	logging.LogInfo("Students processed",
		"student_count", len(students),
		"grade_bounds_used", len(gradeBounds))

	return students
}

// CalculateAverageGrade calculates the average grade from a list of students
func CalculateAverageGrade(students []models.Student) float64 {
	if len(students) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, student := range students {
		sum += float64(student.Grade)
	}

	average := sum / float64(len(students))

	logging.LogDebug("Average grade calculated",
		"student_count", len(students),
		"average_grade", average)

	return math.Round(average*100) / 100 // Round to 2 decimal places
}

// ParseCSVFile parses an uploaded CSV file and returns a list of students
func ParseCSVFile(fileHeader *multipart.FileHeader) ([]models.Student, error) {
	start := time.Now()

	// Validate file
	if err := security.ValidateUpload(fileHeader); err != nil {
		logging.LogSecurityEvent("Invalid file upload attempted", "medium",
			"filename", fileHeader.Filename,
			"size", fileHeader.Size,
			"error", err.Error())
		return nil, err
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		logging.LogError("Failed to open uploaded file", err,
			"filename", fileHeader.Filename,
			"size", fileHeader.Size)
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	// Try to detect delimiter: first try comma, then semicolon
	firstBytes := make([]byte, 1024)
	n, _ := file.Read(firstBytes)
	if _, err := file.Seek(0, 0); err != nil {
		return []models.Student{}, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	delimiter := ','
	if n > 0 {
		content := string(firstBytes[:n])
		commaCount := strings.Count(content, ",")
		semicolonCount := strings.Count(content, ";")

		// Use semicolon if it appears more frequently than comma
		if semicolonCount > commaCount {
			delimiter = ';'
		}
	}

	reader.Comma = delimiter
	logging.LogDebug("CSV delimiter detected", "delimiter", string(delimiter))

	var students []models.Student
	var skippedRows int

	for rowNum := 0; ; rowNum++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logging.LogWarn("CSV parsing error",
				"filename", fileHeader.Filename,
				"row", rowNum,
				"error", err.Error())
			skippedRows++
			continue
		}

		// Skip header row or empty rows
		if rowNum == 0 && (len(record) > 0 && strings.ToLower(record[0]) == "name") {
			continue
		}

		if len(record) < 2 {
			skippedRows++
			continue
		}

		// Extract name and points
		name := strings.TrimSpace(record[0])
		pointsStr := strings.TrimSpace(record[1])

		// Skip empty rows
		if name == "" && pointsStr == "" {
			continue
		}

		if name == "" {
			logging.LogWarn("Empty name in CSV",
				"filename", fileHeader.Filename,
				"row", rowNum,
				"points", pointsStr)
			skippedRows++
			continue
		}

		// Parse points
		pointsStr = strings.ReplaceAll(pointsStr, ",", ".") // Handle German decimal format
		points, err := strconv.ParseFloat(pointsStr, 64)
		if err != nil {
			logging.LogWarn("Invalid points value in CSV",
				"filename", fileHeader.Filename,
				"row", rowNum,
				"name", name,
				"points_str", pointsStr,
				"error", err.Error())
			skippedRows++
			continue
		}

		// Validate points (reasonable range)
		if points < 0 || points > 1000 {
			logging.LogWarn("Points value out of reasonable range",
				"filename", fileHeader.Filename,
				"row", rowNum,
				"name", name,
				"points", points)
			skippedRows++
			continue
		}

		// Sanitize name and add student
		sanitizedName := security.SanitizeName(name)
		students = append(students, models.Student{
			Name:   sanitizedName,
			Points: points,
		})

		// Limit number of students for security
		if len(students) >= models.MaxStudents {
			logging.LogWarn("Maximum student limit reached",
				"filename", fileHeader.Filename,
				"max_students", models.MaxStudents)
			break
		}
	}

	duration := time.Since(start)

	if len(students) == 0 {
		err := fmt.Errorf("no valid student data found in CSV file")
		logging.LogError("CSV parsing resulted in no students", err,
			"filename", fileHeader.Filename,
			"total_rows_processed", skippedRows)
		return nil, err
	}

	logging.LogFileOperation("csv_parse", fileHeader.Filename, fileHeader.Size, duration, true,
		"total_students", len(students),
		"skipped_rows", skippedRows)

	return students, nil
}
