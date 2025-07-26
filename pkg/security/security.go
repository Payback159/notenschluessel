package security

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting per IP address
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mutex    sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

// GetLimiter returns a rate limiter for the given IP address
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(models.RateLimit)/60, models.RateBurst) // per second rate from per minute
		rl.limiters[ip] = limiter

		logging.LogDebug("Created new rate limiter for IP",
			"ip", ip,
			"rate_per_minute", models.RateLimit,
			"burst", models.RateBurst)
	}

	return limiter
}

// RateLimitMiddleware provides rate limiting functionality
func (rl *RateLimiter) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			logging.LogSecurityEvent("Rate limit exceeded", "high",
				"ip", ip,
				"user_agent", r.UserAgent(),
				"path", r.URL.Path,
				"method", r.Method)

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// SecurityHeaders adds security headers to HTTP responses
func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")

		// Only set HSTS in production
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next(w, r)
	}
}

// GetClientIP extracts the real client IP from request headers
func GetClientIP(r *http.Request) string {
	// Check for forwarded IP in common headers
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to remote address
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}

	// Remove brackets for IPv6
	ip = strings.Trim(ip, "[]")

	return ip
}

// ValidateUpload validates file uploads for security
func ValidateUpload(fileHeader *multipart.FileHeader) error {
	if fileHeader.Size > models.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", fileHeader.Size, models.MaxFileSize)
	}

	filename := fileHeader.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return fmt.Errorf("only CSV files are allowed")
	}

	// Additional filename validation
	if len(filename) > models.MaxNameLength {
		return fmt.Errorf("filename too long (max: %d characters)", models.MaxNameLength)
	}

	// Check for potentially dangerous characters
	dangerousChars := []string{"../", "..\\", "<", ">", "|", "&", ";", "$", "`"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains invalid characters")
		}
	}

	return nil
}

// SanitizeName removes or replaces potentially dangerous characters from names
func SanitizeName(name string) string {
	// Remove HTML tags and dangerous characters
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "&", "&amp;")
	name = strings.ReplaceAll(name, "\"", "&quot;")
	name = strings.ReplaceAll(name, "'", "&#39;")
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "\r", " ")
	name = strings.ReplaceAll(name, "\t", " ")

	// Trim whitespace and limit length
	name = strings.TrimSpace(name)
	if len(name) > models.MaxNameLength {
		name = name[:models.MaxNameLength]
	}

	return name
}
