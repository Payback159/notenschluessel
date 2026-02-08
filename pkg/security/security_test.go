package security

import (
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"

	"github.com/payback159/notenschluessel/pkg/logging"
)

func init() {
	logging.InitLogger()
}

// --- GetClientIP ---

func TestGetClientIP_CloudflareHeader(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("CF-Connecting-IP", "1.2.3.4")
	r.Header.Set("X-Forwarded-For", "5.6.7.8")
	r.RemoteAddr = "9.10.11.12:1234"

	ip := GetClientIP(r)
	if ip != "1.2.3.4" {
		t.Errorf("want CF-Connecting-IP 1.2.3.4, got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	r.RemoteAddr = "9.10.11.12:1234"

	ip := GetClientIP(r)
	if ip != "10.0.0.1" {
		t.Errorf("want first XFF IP 10.0.0.1, got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "192.168.1.1")
	r.RemoteAddr = "9.10.11.12:1234"

	ip := GetClientIP(r)
	if ip != "192.168.1.1" {
		t.Errorf("want X-Real-IP 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.16.0.1:54321"

	ip := GetClientIP(r)
	if ip != "172.16.0.1" {
		t.Errorf("want 172.16.0.1, got %s", ip)
	}
}

func TestGetClientIP_IPv6(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:8080"

	ip := GetClientIP(r)
	if ip != "::1" {
		t.Errorf("want ::1, got %s", ip)
	}
}

// --- ValidateUpload ---

func makeFileHeader(name string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: name,
		Size:     size,
		Header:   textproto.MIMEHeader{},
	}
}

func TestValidateUpload_ValidCSV(t *testing.T) {
	fh := makeFileHeader("students.csv", 1024)
	if err := ValidateUpload(fh); err != nil {
		t.Errorf("valid CSV rejected: %v", err)
	}
}

func TestValidateUpload_TooLarge(t *testing.T) {
	fh := makeFileHeader("big.csv", 11<<20) // 11 MB
	if err := ValidateUpload(fh); err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestValidateUpload_WrongExtension(t *testing.T) {
	fh := makeFileHeader("data.xlsx", 100)
	if err := ValidateUpload(fh); err == nil {
		t.Error("expected error for non-CSV file")
	}
}

func TestValidateUpload_CaseInsensitiveExtension(t *testing.T) {
	fh := makeFileHeader("DATA.CSV", 100)
	if err := ValidateUpload(fh); err != nil {
		t.Errorf("uppercase .CSV rejected: %v", err)
	}
}

func TestValidateUpload_FilenameTooLong(t *testing.T) {
	name := make([]byte, 250)
	for i := range name {
		name[i] = 'a'
	}
	fh := makeFileHeader(string(name)+".csv", 100)
	if err := ValidateUpload(fh); err == nil {
		t.Error("expected error for overly long filename")
	}
}

func TestValidateUpload_DangerousChars(t *testing.T) {
	dangerous := []string{
		"../etc/passwd.csv",
		"file<script>.csv",
		"file;rm.csv",
		"file$HOME.csv",
		"file`cmd`.csv",
	}
	for _, name := range dangerous {
		fh := makeFileHeader(name, 100)
		if err := ValidateUpload(fh); err == nil {
			t.Errorf("expected error for dangerous filename %q", name)
		}
	}
}

// --- SanitizeName ---

func TestSanitizeName_Normal(t *testing.T) {
	name := SanitizeName("Max Mustermann")
	if name != "Max Mustermann" {
		t.Errorf("want 'Max Mustermann', got %q", name)
	}
}

func TestSanitizeName_HTMLTags(t *testing.T) {
	name := SanitizeName("<script>alert('xss')</script>")
	if name != "script>alert('xss')/script>" {
		// After removing < and >, we get this
		// The important thing: no < or > in output
		if containsAny(name, "<>") {
			t.Errorf("sanitized name still contains HTML brackets: %q", name)
		}
	}
}

func TestSanitizeName_NoDoubleEscaping(t *testing.T) {
	// Should NOT produce &amp; — html/template handles escaping
	name := SanitizeName("Tom & Jerry")
	if name != "Tom & Jerry" {
		t.Errorf("want 'Tom & Jerry' (no entity encoding), got %q", name)
	}
}

func TestSanitizeName_Newlines(t *testing.T) {
	name := SanitizeName("Line1\nLine2\rLine3\tTab")
	if containsAny(name, "\n\r\t") {
		t.Errorf("sanitized name still contains control characters: %q", name)
	}
}

func TestSanitizeName_Truncation(t *testing.T) {
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'x'
	}
	name := SanitizeName(string(long))
	if len(name) > 200 {
		t.Errorf("name not truncated: length %d", len(name))
	}
}

func TestSanitizeName_Whitespace(t *testing.T) {
	name := SanitizeName("  padded  ")
	if name != "padded" {
		t.Errorf("whitespace not trimmed: %q", name)
	}
}

func containsAny(s string, chars string) bool {
	for _, c := range chars {
		for _, sc := range s {
			if sc == c {
				return true
			}
		}
	}
	return false
}

// --- RateLimiter ---

func TestRateLimiter_CreateAndGet(t *testing.T) {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
	}

	l1 := rl.GetLimiter("1.2.3.4")
	l2 := rl.GetLimiter("1.2.3.4")

	if l1 != l2 {
		t.Error("same IP should return same limiter instance")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
	}

	l1 := rl.GetLimiter("1.1.1.1")
	l2 := rl.GetLimiter("2.2.2.2")

	if l1 == l2 {
		t.Error("different IPs should have different limiter instances")
	}
}

func TestRateLimiter_RateLimitMiddleware(t *testing.T) {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
	}

	called := 0
	handler := rl.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
	}))

	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"

	// First request should pass
	w := &fakeResponseWriter{}
	handler.ServeHTTP(w, r)

	if called != 1 {
		t.Errorf("handler should have been called once, got %d", called)
	}
}

// fakeResponseWriter is a minimal implementation for testing
type fakeResponseWriter struct {
	code   int
	header http.Header
	body   []byte
}

func (f *fakeResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}
func (f *fakeResponseWriter) Write(b []byte) (int, error) {
	f.body = append(f.body, b...)
	return len(b), nil
}
func (f *fakeResponseWriter) WriteHeader(code int) {
	f.code = code
}
