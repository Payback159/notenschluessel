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

// --- Arbitrary points / step sizes (sweep over the valid input domain) ---

// isMultipleOf reports whether v is an integer multiple of step (within epsilon).
func isMultipleOf(v, step float64) bool {
	r := math.Mod(v, step)
	return r < 1e-9 || math.Abs(r-step) < 1e-9
}

// assertBoundsInvariants checks the properties that must hold for every valid
// combination of maxPoints, minPoints and breakPointPercent.
func assertBoundsInvariants(t *testing.T, maxPoints int, minPoints, breakPointPercent float64) {
	t.Helper()

	bounds := CalculateGradeBounds(maxPoints, minPoints, breakPointPercent)

	if len(bounds) != 5 {
		t.Fatalf("max=%d step=%g bp=%g: expected 5 bounds, got %d",
			maxPoints, minPoints, breakPointPercent, len(bounds))
	}

	// Grades labelled 1..5 in order
	for i, b := range bounds {
		if b.Grade != i+1 {
			t.Errorf("max=%d step=%g bp=%g: bound[%d].Grade want %d, got %d",
				maxPoints, minPoints, breakPointPercent, i, i+1, b.Grade)
		}
	}

	// Grade 1 always tops out at maxPoints, grade 5 always bottoms out at 0
	if bounds[0].UpperBound != float64(maxPoints) {
		t.Errorf("max=%d step=%g bp=%g: grade 1 upper want %d, got %.2f",
			maxPoints, minPoints, breakPointPercent, maxPoints, bounds[0].UpperBound)
	}
	if bounds[4].LowerBound != 0 {
		t.Errorf("max=%d step=%g bp=%g: grade 5 lower want 0, got %.2f",
			maxPoints, minPoints, breakPointPercent, bounds[4].LowerBound)
	}

	// The breakpoint is the lower bound of grade 4 (rounded to the step).
	// Mirror the production grouping exactly to avoid float rounding drift.
	breakAbs := float64(maxPoints) * (breakPointPercent / 100.0)
	wantBreak := math.Max(0, math.Round(breakAbs/minPoints)*minPoints)
	if math.Abs(bounds[3].LowerBound-wantBreak) > 1e-9 {
		t.Errorf("max=%d step=%g bp=%g: grade 4 lower (breakpoint) want %.2f, got %.2f",
			maxPoints, minPoints, breakPointPercent, wantBreak, bounds[3].LowerBound)
	}

	for _, b := range bounds {
		// No negative lower bounds
		if b.LowerBound < 0 {
			t.Errorf("max=%d step=%g bp=%g: grade %d has negative lower bound %.2f",
				maxPoints, minPoints, breakPointPercent, b.Grade, b.LowerBound)
		}
		// Lower bounds must align to the step size
		if !isMultipleOf(b.LowerBound, minPoints) {
			t.Errorf("max=%d step=%g bp=%g: grade %d lower bound %.4f not a multiple of step",
				maxPoints, minPoints, breakPointPercent, b.Grade, b.LowerBound)
		}
	}

	// Lower bounds must be monotonically non-increasing from grade 1 to 5
	for i := 1; i < len(bounds); i++ {
		if bounds[i].LowerBound > bounds[i-1].LowerBound {
			t.Errorf("max=%d step=%g bp=%g: grade %d lower (%.2f) > grade %d lower (%.2f)",
				maxPoints, minPoints, breakPointPercent,
				bounds[i].Grade, bounds[i].LowerBound,
				bounds[i-1].Grade, bounds[i-1].LowerBound)
		}
	}
}

// TestCalculateGradeBounds_ArbitraryInputs sweeps a wide range of point counts,
// step sizes and breakpoints to ensure the core invariants always hold.
func TestCalculateGradeBounds_ArbitraryInputs(t *testing.T) {
	maxPointsCases := []int{7, 10, 13, 20, 37, 45, 50, 73, 100, 137, 250, 1000}
	stepCases := []float64{0.25, 0.5, 1, 2, 2.5, 5}
	breakCases := []float64{1, 25, 40, 50, 60, 66.6, 75, 90, 99}

	for _, mp := range maxPointsCases {
		for _, step := range stepCases {
			if step > float64(mp) { // mirrors handler validation: step <= maxPoints
				continue
			}
			for _, bp := range breakCases {
				assertBoundsInvariants(t, mp, step, bp)
			}
		}
	}
}

// TestCalculateGradeBounds_NonInvertedRanges verifies that when the scale has
// enough resolution (enough step increments to fit five distinct grades) every
// grade range is well-formed: upper >= lower and ranges do not overlap.
func TestCalculateGradeBounds_NonInvertedRanges(t *testing.T) {
	maxPointsCases := []int{20, 37, 45, 50, 73, 100, 250}
	stepCases := []float64{0.25, 0.5, 1, 2.5}
	breakCases := []float64{40, 50, 60, 66.6}

	for _, mp := range maxPointsCases {
		for _, step := range stepCases {
			// Require enough increments so five grades can be represented.
			if float64(mp)/step < 16 {
				continue
			}
			for _, bp := range breakCases {
				bounds := CalculateGradeBounds(mp, step, bp)
				for _, b := range bounds {
					if b.UpperBound < b.LowerBound {
						t.Errorf("max=%d step=%g bp=%g: grade %d inverted range %.2f-%.2f",
							mp, step, bp, b.Grade, b.LowerBound, b.UpperBound)
					}
				}
				// Adjacent grades must not overlap: grade i upper < grade i-1 lower.
				for i := 1; i < len(bounds); i++ {
					if bounds[i].UpperBound >= bounds[i-1].LowerBound {
						t.Errorf("max=%d step=%g bp=%g: grade %d upper (%.2f) overlaps grade %d lower (%.2f)",
							mp, step, bp, bounds[i].Grade, bounds[i].UpperBound,
							bounds[i-1].Grade, bounds[i-1].LowerBound)
					}
				}
			}
		}
	}
}

// TestCalculateGradeBounds_BreakpointRegression locks in the reported bug fix:
// max 45, step 0.5, breakpoint 50% must put the passing line (grade 4) at 22.5
// and let grade 5 cover 0-22, not collapse to ~7 points.
func TestCalculateGradeBounds_BreakpointRegression(t *testing.T) {
	bounds := CalculateGradeBounds(45, 0.5, 50)

	want := []struct {
		grade        int
		lower, upper float64
	}{
		{1, 39.5, 45.0},
		{2, 34.0, 39.0},
		{3, 28.0, 33.5},
		{4, 22.5, 27.5},
		{5, 0.0, 22.0},
	}

	for i, w := range want {
		b := bounds[i]
		if b.Grade != w.grade || math.Abs(b.LowerBound-w.lower) > 1e-9 || math.Abs(b.UpperBound-w.upper) > 1e-9 {
			t.Errorf("grade %d: want %.1f-%.1f, got %.1f-%.1f",
				w.grade, w.lower, w.upper, b.LowerBound, b.UpperBound)
		}
	}
}

// TestCalculateGradeBounds_EqualSegmentsAboveBreakpoint verifies that the four
// grades above the breakpoint span equal segments before rounding, for an
// arbitrary point count and step size.
func TestCalculateGradeBounds_EqualSegmentsAboveBreakpoint(t *testing.T) {
	// max 80, step 1, breakpoint 50% -> breakpoint at 40, segment = 10.
	bounds := CalculateGradeBounds(80, 1, 50)

	expected := map[int]float64{1: 70, 2: 60, 3: 50, 4: 40}
	for _, b := range bounds {
		if want, ok := expected[b.Grade]; ok {
			if math.Abs(b.LowerBound-want) > 1e-9 {
				t.Errorf("grade %d lower bound: want %.1f, got %.1f", b.Grade, want, b.LowerBound)
			}
		}
	}
}

// TestCalculateGradeBounds_RoundingAlignsToStep checks step alignment for an
// unusual step size (0.25) and a non-round point count.
func TestCalculateGradeBounds_RoundingAlignsToStep(t *testing.T) {
	bounds := CalculateGradeBounds(37, 0.25, 60)

	for _, b := range bounds {
		if !isMultipleOf(b.LowerBound, 0.25) {
			t.Errorf("grade %d lower bound %.4f is not a multiple of 0.25", b.Grade, b.LowerBound)
		}
	}
}

func TestValidateGradeBounds_Valid(t *testing.T) {
	bounds := CalculateGradeBounds(45, 0.5, 50)

	valid, reason := ValidateGradeBounds(bounds)
	if !valid {
		t.Fatalf("expected valid bounds, got invalid: %s", reason)
	}
}

func TestValidateGradeBounds_InvalidDegenerateRanges(t *testing.T) {
	// Coarse step size with very low breakpoint collapses/inverts ranges.
	bounds := CalculateGradeBounds(10, 5, 1)

	valid, reason := ValidateGradeBounds(bounds)
	if valid {
		t.Fatal("expected invalid bounds, got valid")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason for invalid bounds")
	}
}
