package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/session"
)

func init() {
	logging.InitLogger()
}

func newTestHandler() *Handler {
	tmpl := template.Must(template.New("index.html").Parse(
		`{{if .Message}}<div>{{.Message.Text}}</div>{{end}}` +
			`{{if .HasResults}}<div>results</div>{{end}}`))
	store := session.NewStore()
	return NewHandler(tmpl, store)
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
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	h.HandleHome(w, req)

	if !strings.Contains(w.Body.String(), "Ungültige Punkteschrittweite") {
		t.Error("response should contain error for invalid min points")
	}
}
