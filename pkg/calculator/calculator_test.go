package calculator

import (
	"math"
	"testing"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
)

func TestMain(m *testing.M) {
	logging.InitLogger()
	m.Run()
}

// --- CalculateGradeBounds ---

func TestCalculateGradeBounds_StandardCase(t *testing.T) {
	bounds := CalculateGradeBounds(100, 0.5, 50)

	if len(bounds) != 5 {
		t.Fatalf("expected 5 grade bounds, got %d", len(bounds))
	}

	// Grade 1: upper bound must be maxPoints
	if bounds[0].UpperBound != 100 {
		t.Errorf("grade 1 upper bound: want 100, got %.1f", bounds[0].UpperBound)
	}

	// All grades must be 1-5
	for i, b := range bounds {
		if b.Grade != i+1 {
			t.Errorf("bound[%d].Grade: want %d, got %d", i, i+1, b.Grade)
		}
	}

	// Lower bounds must be non-negative
	for _, b := range bounds {
		if b.LowerBound < 0 {
			t.Errorf("grade %d has negative lower bound: %.1f", b.Grade, b.LowerBound)
		}
	}

	// Lower bounds must be monotonically decreasing
	for i := 1; i < len(bounds); i++ {
		if bounds[i].LowerBound > bounds[i-1].LowerBound {
			t.Errorf("grade %d lower bound (%.1f) > grade %d lower bound (%.1f)",
				bounds[i].Grade, bounds[i].LowerBound,
				bounds[i-1].Grade, bounds[i-1].LowerBound)
		}
	}

	// Each grade's upper bound must be less than previous grade's lower bound
	for i := 1; i < len(bounds); i++ {
		if bounds[i].UpperBound >= bounds[i-1].LowerBound {
			t.Errorf("grade %d upper bound (%.1f) >= grade %d lower bound (%.1f), ranges overlap",
				bounds[i].Grade, bounds[i].UpperBound,
				bounds[i-1].Grade, bounds[i-1].LowerBound)
		}
	}
}

func TestCalculateGradeBounds_RoundingToMinPoints(t *testing.T) {
	bounds := CalculateGradeBounds(100, 0.5, 50)

	for _, b := range bounds {
		// Check that lower bounds are multiples of minPoints (0.5)
		remainder := math.Mod(b.LowerBound, 0.5)
		if remainder > 1e-9 && math.Abs(remainder-0.5) > 1e-9 {
			t.Errorf("grade %d lower bound %.2f is not a multiple of 0.5", b.Grade, b.LowerBound)
		}
	}
}

func TestCalculateGradeBounds_LowBreakpoint(t *testing.T) {
	// Very low breakpoint: all bounds must stay non-negative
	bounds := CalculateGradeBounds(100, 1, 5)

	for _, b := range bounds {
		if b.LowerBound < 0 {
			t.Errorf("grade %d has negative lower bound: %.1f", b.Grade, b.LowerBound)
		}
	}
}

func TestCalculateGradeBounds_HighBreakpoint(t *testing.T) {
	bounds := CalculateGradeBounds(100, 1, 90)

	if bounds[0].UpperBound != 100 {
		t.Errorf("grade 1 upper bound: want 100, got %.1f", bounds[0].UpperBound)
	}

	// Breakpoint at 90%: grade 2 lower bound should be around 90
	if bounds[1].LowerBound < 80 || bounds[1].LowerBound > 100 {
		t.Errorf("grade 2 lower bound %.1f seems unexpected for 90%% breakpoint", bounds[1].LowerBound)
	}
}

func TestCalculateGradeBounds_MinPointsWholeNumber(t *testing.T) {
	bounds := CalculateGradeBounds(50, 1, 50)

	for _, b := range bounds {
		remainder := math.Mod(b.LowerBound, 1)
		if remainder > 1e-9 && math.Abs(remainder-1) > 1e-9 {
			t.Errorf("grade %d lower bound %.2f is not a whole number", b.Grade, b.LowerBound)
		}
	}
}

func TestCalculateGradeBounds_SmallMaxPoints(t *testing.T) {
	bounds := CalculateGradeBounds(10, 0.5, 50)

	if len(bounds) != 5 {
		t.Fatalf("expected 5 bounds, got %d", len(bounds))
	}
	for _, b := range bounds {
		if b.LowerBound < 0 {
			t.Errorf("grade %d lower bound negative: %.1f", b.Grade, b.LowerBound)
		}
		if b.UpperBound > 10 {
			t.Errorf("grade %d upper bound %.1f exceeds maxPoints 10", b.Grade, b.UpperBound)
		}
	}
}

// --- CalculateGrade ---

func TestCalculateGrade(t *testing.T) {
	tests := []struct {
		name   string
		points float64
		want   int
	}{
		{"max points", 100, 1},
		{"grade 1 boundary", 85, 1},
		{"grade 2", 60, 2},
		{"grade 2 boundary", 50, 2},
		{"grade 3", 35, 3},
		{"grade 3 boundary", 30, 3},
		{"grade 4", 20, 4},
		{"grade 4 boundary", 16.5, 4},
		{"grade 5", 10, 5},
		{"zero points", 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateGrade(tt.points, 85, 50, 30, 16.5, 0)
			if got != tt.want {
				t.Errorf("CalculateGrade(%.1f): want %d, got %d", tt.points, tt.want, got)
			}
		})
	}
}

// --- ProcessStudents ---

func TestProcessStudents(t *testing.T) {
	bounds := CalculateGradeBounds(100, 0.5, 50)

	students := []models.Student{
		{Name: "Alice", Points: 95},
		{Name: "Bob", Points: 60},
		{Name: "Charlie", Points: 40},
		{Name: "Diana", Points: 20},
		{Name: "Eve", Points: 5},
	}

	result := ProcessStudents(students, bounds)

	if result[0].Grade != 1 {
		t.Errorf("Alice (95 pts) should be grade 1, got %d", result[0].Grade)
	}
	if result[4].Grade != 5 {
		t.Errorf("Eve (5 pts) should be grade 5, got %d", result[4].Grade)
	}

	// Grades should be monotonically increasing (worse) as points decrease
	for i := 1; i < len(result); i++ {
		if result[i].Grade < result[i-1].Grade {
			t.Errorf("grade for %s (%d) is better than %s (%d) despite fewer points",
				result[i].Name, result[i].Grade, result[i-1].Name, result[i-1].Grade)
		}
	}
}

func TestProcessStudents_InsufficientBounds(t *testing.T) {
	students := []models.Student{{Name: "Test", Points: 50}}
	bounds := []models.GradeBound{{Grade: 1, LowerBound: 85, UpperBound: 100}}

	result := ProcessStudents(students, bounds)

	// Should return students unmodified (grade 0 = unprocessed)
	if result[0].Grade != 0 {
		t.Errorf("expected unmodified grade 0, got %d", result[0].Grade)
	}
}

func TestProcessStudents_EmptyList(t *testing.T) {
	bounds := CalculateGradeBounds(100, 0.5, 50)
	result := ProcessStudents([]models.Student{}, bounds)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d students", len(result))
	}
}

// --- CalculateAverageGrade ---

func TestCalculateAverageGrade(t *testing.T) {
	students := []models.Student{
		{Name: "A", Grade: 1},
		{Name: "B", Grade: 2},
		{Name: "C", Grade: 3},
	}

	avg := CalculateAverageGrade(students)

	if avg != 2.0 {
		t.Errorf("average: want 2.0, got %.2f", avg)
	}
}

func TestCalculateAverageGrade_Empty(t *testing.T) {
	avg := CalculateAverageGrade([]models.Student{})

	if avg != 0.0 {
		t.Errorf("average of empty: want 0.0, got %.2f", avg)
	}
}

func TestCalculateAverageGrade_SingleStudent(t *testing.T) {
	students := []models.Student{{Name: "X", Grade: 4}}
	avg := CalculateAverageGrade(students)

	if avg != 4.0 {
		t.Errorf("average: want 4.0, got %.2f", avg)
	}
}

func TestCalculateAverageGrade_RoundsToTwoDecimals(t *testing.T) {
	students := []models.Student{
		{Name: "A", Grade: 1},
		{Name: "B", Grade: 1},
		{Name: "C", Grade: 2},
	}

	avg := CalculateAverageGrade(students)

	if avg != 1.33 {
		t.Errorf("average: want 1.33, got %.2f", avg)
	}
}
