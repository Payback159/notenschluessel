package handlers

import (
	"bytes"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"github.com/payback159/notenschluessel/pkg/session"
)

func init() {
	logging.InitLogger()
}

func newTestHandler() *Handler {
	tmpl := template.Must(template.New("index.html").Parse(
		`{{if .Message}}<div>{{.Message.Text}}</div>{{end}}` +
			`{{if .HasResults}}<div>results</div>{{end}}`))
	template.Must(tmpl.New("privacy.html").Parse(`privacy-page`))
	store := session.NewStore()
	return NewHandler(tmpl, store)
}

func buildMultipartRequest(fields map[string][]string, includeFile bool) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for name, values := range fields {
		for _, value := range values {
			if err := writer.WriteField(name, value); err != nil {
				return nil, err
			}
		}
	}

	if includeFile {
		part, err := writer.CreateFormFile("csvFile", "students.csv")
		if err != nil {
			return nil, err
		}
		_, err = part.Write([]byte("Name,Punkte\nAlice,10\n"))
		if err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// --- HandleHome GET ---

func TestHandleHome_GET(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /: want 200, got %d", w.Code)
	}
}

func TestHandleHome_NotFound(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GET /nonexistent: want 404, got %d", w.Code)
	}
}

func TestHandleHome_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT /: want 405, got %d", w.Code)
	}
}

// --- HandleCalculation POST ---

func buildMultipartBody(fields map[string]string) string {
	var sb strings.Builder
	for name, value := range fields {
		sb.WriteString("--boundary\r\n")
		sb.WriteString("Content-Disposition: form-data; name=\"" + name + "\"\r\n\r\n")
		sb.WriteString(value + "\r\n")
	}
	sb.WriteString("--boundary--\r\n")
	return sb.String()
}

func TestHandleHome_POST_Valid(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "100",
		"minPoints":         "0.5",
		"breakPointPercent": "50",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST /: want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "results") {
		t.Error("response should contain results")
	}
}

func TestHandleHome_POST_InvalidMaxPoints(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "-5",
		"minPoints":         "0.5",
		"breakPointPercent": "50",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 (with error message), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Ungültige maximale Punktzahl") {
		t.Error("response should contain error message for invalid max points")
	}
}

func TestHandleHome_POST_InvalidBreakPoint(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "100",
		"minPoints":         "0.5",
		"breakPointPercent": "150",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if !strings.Contains(w.Body.String(), "Ungültiger Knickpunkt") {
		t.Error("response should contain error for invalid breakpoint")
	}
}

func TestHandleHome_POST_MaxPointsTooHigh(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "9999",
		"minPoints":         "0.5",
		"breakPointPercent": "50",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if !strings.Contains(w.Body.String(), "Ungültige maximale Punktzahl") {
		t.Error("maxPoints > 1000 should be rejected")
	}
}

// --- Session cookie ---

func TestHandleHome_POST_SetsCookie(t *testing.T) {
	store := session.NewStore()
	tmpl := template.Must(template.New("index.html").Parse(`{{if .HasResults}}ok{{end}}`))
	h := NewHandler(tmpl, store)

	body := buildMultipartBody(map[string]string{
		"maxPoints":         "100",
		"minPoints":         "0.5",
		"breakPointPercent": "50",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session_id" {
			found = true
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Error("session cookie should be SameSite=Strict")
			}
			if len(c.Value) != 32 {
				t.Errorf("session ID should be 32 hex chars, got %d", len(c.Value))
			}
		}
	}
	if !found {
		t.Error("session_id cookie not set after successful calculation")
	}
}

func TestHandleHome_POST_InvalidMinPoints(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "100",
		"minPoints":         "-1",
		"breakPointPercent": "50",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if !strings.Contains(w.Body.String(), "Ungültige Punkteschrittweite") {
		t.Error("response should contain error for invalid min points")
	}
}

func TestHandleHome_POST_InvalidScaleCombination(t *testing.T) {
	h := newTestHandler()
	body := buildMultipartBody(map[string]string{
		"maxPoints":         "10",
		"minPoints":         "5",
		"breakPointPercent": "1",
		"inputMode":         "csv",
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 (with error message), got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "keine gültige Notenskala") {
		t.Error("response should contain error for invalid scale combination")
	}
	if strings.Contains(w.Body.String(), "results") {
		t.Error("invalid scale combination should not render results")
	}
}

func TestHandleHome_POST_ManualOnlyValid(t *testing.T) {
	h := newTestHandler()
	req, err := buildMultipartRequest(map[string][]string{
		"maxPoints":         []string{"100"},
		"minPoints":         []string{"0.5"},
		"breakPointPercent": []string{"50"},
		"inputMode":         []string{"manual"},
		"manualName":        []string{"Alice", "Bob"},
		"manualPoints":      []string{"80", "45.5"},
	}, false)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "results") {
		t.Error("response should contain results for manual-only input")
	}
}

func TestHandleHome_POST_BothInputModesRejected(t *testing.T) {
	h := newTestHandler()
	req, err := buildMultipartRequest(map[string][]string{
		"maxPoints":         []string{"100"},
		"minPoints":         []string{"0.5"},
		"breakPointPercent": []string{"50"},
		"inputMode":         []string{"manual"},
		"manualName":        []string{"Alice"},
		"manualPoints":      []string{"80"},
	}, true)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if !strings.Contains(w.Body.String(), "nicht erlaubt") {
		t.Error("response should contain exclusivity error when both inputs are provided")
	}
}

func TestHandlePrivacy_GET(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	w := httptest.NewRecorder()

	h.HandlePrivacy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /privacy: want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "privacy-page") {
		t.Error("privacy template should be rendered")
	}
}

func TestHandleDeleteSession_POST(t *testing.T) {
	store := session.NewStore()
	tmpl := template.Must(template.New("index.html").Parse(`{{if .Message}}<div>{{.Message.Text}}</div>{{end}}`))
	template.Must(tmpl.New("privacy.html").Parse(`privacy-page`))
	h := NewHandler(tmpl, store)

	store.Set("abc123", models.PageData{MaxPoints: 100})

	req := httptest.NewRequest(http.MethodPost, "/session/delete", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
	w := httptest.NewRecorder()

	h.HandleDeleteSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST /session/delete: want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "gelöscht") {
		t.Error("response should contain deletion success message")
	}
	if _, exists := store.Get("abc123"); exists {
		t.Error("session should be removed from store")
	}
}
