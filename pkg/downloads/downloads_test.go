package downloads

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"github.com/payback159/notenschluessel/pkg/session"
)

func init() {
	logging.InitLogger()
}

// --- sanitizeCSVField ---

func TestSanitizeCSVField_Normal(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"Alice", "Alice"},
		{"Bob Smith", "Bob Smith"},
		{"", ""},
	}
	for _, tc := range cases {
		got := sanitizeCSVField(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeCSVField(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeCSVField_FormulaInjection(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"=CMD()", "'=CMD()"},
		{"+1+1", "'+1+1"},
		{"-1-1", "'-1-1"},
		{"@SUM(A1)", "'@SUM(A1)"},
		{"\tmalicious", "'\tmalicious"},
		{"\rmalicious", "'\rmalicious"},
	}
	for _, tc := range cases {
		got := sanitizeCSVField(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeCSVField(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeCSVField_QuotesAndCommas(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{`He said "hi"`, `"He said ""hi"""`},
		{"a,b,c", `"a,b,c"`},
		{"line1\nline2", `"line1` + "\n" + `line2"`},
	}
	for _, tc := range cases {
		got := sanitizeCSVField(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeCSVField(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeCSVField_InjectionAndQuotes(t *testing.T) {
	// Formula prefix AND quotes: should get both protections
	got := sanitizeCSVField(`=CMD("evil")`)
	want := `"'=CMD(""evil"")"`
	if got != want {
		t.Errorf("sanitizeCSVField with injection+quotes: got %q, want %q", got, want)
	}
}

// --- getSessionIDFromCookie ---

func TestGetSessionIDFromCookie_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})

	got := getSessionIDFromCookie(req)
	if got != "abc123" {
		t.Errorf("want abc123, got %s", got)
	}
}

func TestGetSessionIDFromCookie_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	got := getSessionIDFromCookie(req)
	if got != "" {
		t.Errorf("want empty, got %s", got)
	}
}

// --- setDownloadHeaders ---

func TestSetDownloadHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	setDownloadHeaders(w, "text/csv", "test.csv")

	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("Content-Type: got %s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Content-Disposition") != "attachment; filename=test.csv" {
		t.Errorf("Content-Disposition: got %s", w.Header().Get("Content-Disposition"))
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control: got %s", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Pragma") != "no-cache" {
		t.Errorf("Pragma: got %s", w.Header().Get("Pragma"))
	}
}

// --- CSV Download Handlers ---

func newTestStore(sessionID string, data models.PageData) *session.Store {
	store := session.NewStore()
	store.Set(sessionID, data)
	return store
}

func TestHandleGradeScaleCSV_NoSession(t *testing.T) {
	store := session.NewStore()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/grade-scale", nil)

	HandleGradeScaleCSV(w, req, store)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleGradeScaleCSV_WithData(t *testing.T) {
	sid := "test-session-csv"
	data := models.PageData{
		HasResults: true,
		GradeBounds: []models.GradeBound{
			{Grade: 1, LowerBound: 85, UpperBound: 100},
			{Grade: 2, LowerBound: 70, UpperBound: 84.5},
			{Grade: 3, LowerBound: 55, UpperBound: 69.5},
			{Grade: 4, LowerBound: 40, UpperBound: 54.5},
			{Grade: 5, LowerBound: 0, UpperBound: 39.5},
		},
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/grade-scale", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleGradeScaleCSV(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("Content-Type: got %s", w.Header().Get("Content-Type"))
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("response body should not be empty")
	}
	// Check header row
	if body[:len("Note,Punktebereich von,Punktebereich bis")] != "Note,Punktebereich von,Punktebereich bis" {
		t.Error("CSV header row missing")
	}
}

func TestHandleStudentResultsCSV_NoStudents(t *testing.T) {
	sid := "sid-no-students"
	data := models.PageData{HasResults: true, HasStudents: false}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/student-results", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleStudentResultsCSV(w, req, store)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleStudentResultsCSV_WithStudents(t *testing.T) {
	sid := "sid-with-students"
	data := models.PageData{
		HasResults:  true,
		HasStudents: true,
		Students: []models.Student{
			{Name: "Alice", Points: 90, Grade: 1},
			{Name: "Bob", Points: 60, Grade: 3},
		},
		AverageGrade: 2.0,
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/student-results", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleStudentResultsCSV(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("response body should not be empty")
	}
}

func TestHandleCombinedCSV_WithData(t *testing.T) {
	sid := "sid-combined"
	data := models.PageData{
		HasResults:  true,
		HasStudents: true,
		GradeBounds: []models.GradeBound{
			{Grade: 1, LowerBound: 85, UpperBound: 100},
		},
		Students: []models.Student{
			{Name: "Alice", Points: 90, Grade: 1},
		},
		AverageGrade: 1.0,
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/combined", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleCombinedCSV(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("response body should not be empty")
	}
}

// --- Excel handlers (basic smoke tests) ---

func TestHandleGradeScaleExcel_NoSession(t *testing.T) {
	store := session.NewStore()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/grade-scale-excel", nil)

	HandleGradeScaleExcel(w, req, store)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHandleGradeScaleExcel_WithData(t *testing.T) {
	sid := "sid-excel-grades"
	data := models.PageData{
		HasResults: true,
		GradeBounds: []models.GradeBound{
			{Grade: 1, LowerBound: 85, UpperBound: 100},
			{Grade: 2, LowerBound: 70, UpperBound: 84.5},
			{Grade: 3, LowerBound: 55, UpperBound: 69.5},
			{Grade: 4, LowerBound: 40, UpperBound: 54.5},
			{Grade: 5, LowerBound: 0, UpperBound: 39.5},
		},
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/grade-scale-excel", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleGradeScaleExcel(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("Content-Type: got %s", ct)
	}
}

func TestHandleStudentResultsExcel_WithStudents(t *testing.T) {
	sid := "sid-excel-students"
	data := models.PageData{
		HasResults:  true,
		HasStudents: true,
		Students: []models.Student{
			{Name: "Alice", Points: 90, Grade: 1},
			{Name: "Bob", Points: 60, Grade: 3},
		},
		AverageGrade: 2.0,
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/student-results-excel", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleStudentResultsExcel(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestHandleCombinedExcel_WithData(t *testing.T) {
	sid := "sid-excel-combined"
	data := models.PageData{
		HasResults:  true,
		HasStudents: true,
		GradeBounds: []models.GradeBound{
			{Grade: 1, LowerBound: 85, UpperBound: 100},
			{Grade: 2, LowerBound: 70, UpperBound: 84.5},
		},
		Students: []models.Student{
			{Name: "Alice", Points: 90, Grade: 1},
		},
		AverageGrade: 1.0,
	}
	store := newTestStore(sid, data)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/combined-excel", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sid})

	HandleCombinedExcel(w, req, store)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}
