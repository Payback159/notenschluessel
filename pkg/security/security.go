package security

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
	"golang.org/x/time/rate"
)

// ipLimiter wraps a rate limiter with a last-seen timestamp for cleanup
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter manages rate limiting per IP address with automatic cleanup
type RateLimiter struct {
	limiters map[string]*ipLimiter
	mutex    sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with background cleanup
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
	}
	rl.startCleanup()
	return rl
}

// GetLimiter returns a rate limiter for the given IP address
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(models.RateLimit)/60, models.RateBurst)
		rl.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}

		logging.LogDebug("Created new rate limiter for IP",
			"ip", ip,
			"rate_per_minute", models.RateLimit,
			"burst", models.RateBurst)

		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// startCleanup runs a background goroutine to remove stale rate limiters
func (rl *RateLimiter) startCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanupStale()
		}
	}()
}

// cleanupStale removes rate limiters not seen in the last 10 minutes
func (rl *RateLimiter) cleanupStale() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	threshold := time.Now().Add(-10 * time.Minute)
	removed := 0
	for ip, entry := range rl.limiters {
		if entry.lastSeen.Before(threshold) {
			delete(rl.limiters, ip)
			removed++
		}
	}

	if removed > 0 {
		logging.LogInfo("Cleaned up stale rate limiters",
			"removed", removed,
			"remaining", len(rl.limiters))
	}
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

// GetClientIP extracts the real client IP from request headers
func GetClientIP(r *http.Request) string {
	// Cloudflare sets this header with the verified client IP
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}

	// Fallback: Check for forwarded IP in common headers
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

// SanitizeName removes potentially dangerous characters from names.
// Note: html/template auto-escapes output, so no manual HTML entity encoding needed.
func SanitizeName(name string) string {
	// Remove HTML tag characters and control characters
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
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
