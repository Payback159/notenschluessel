package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/payback159/notenschluessel/pkg/downloads"
	"github.com/payback159/notenschluessel/pkg/handlers"
	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/security"
	"github.com/payback159/notenschluessel/pkg/session"
)

func main() {
	// Health check mode: make a real HTTP request to the running server
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://localhost:8080/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		fmt.Println("OK")
		os.Exit(0)
	}

	// Initialize structured logging
	logging.InitLogger()

	logging.LogInfo("Starting Notenschluessel service",
		"version", "v1.0.0",
		"environment", os.Getenv("ENV"))

	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))
	logging.LogInfo("Templates loaded successfully")

	csrf := http.NewCrossOriginProtection()

	// Add trusted origins for our application
	if os.Getenv("ENV") == "production" {
		// In production, only trust HTTPS origins
		if host := os.Getenv("HOSTNAME"); host != "" {
			csrf.AddTrustedOrigin("https://" + host)
			logging.LogInfo("CSRF protection configured for production", "host", host)
		} else {
			logging.LogInfo("CSRF protection enabled with default same-origin policy")
		}
	} else {
		// In development, allow both HTTP and HTTPS localhost
		csrf.AddTrustedOrigin("http://localhost:8080")
		csrf.AddTrustedOrigin("https://localhost:8080")
		csrf.AddTrustedOrigin("http://127.0.0.1:8080")
		csrf.AddTrustedOrigin("https://127.0.0.1:8080")
		logging.LogInfo("CSRF protection enabled for development with localhost origins")
	}

	// Initialize session store
	sessionStore := session.NewStore()
	logging.LogInfo("Session store initialized")

	// Initialize rate limiter
	rateLimiter := security.NewRateLimiter()
	logging.LogInfo("Rate limiter initialized")

	securityHeaders := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent embedding in frames
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Strict transport security (HTTPS only)
			if os.Getenv("ENV") == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Content Security Policy - very strict for our simple tool
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"form-action 'self'; "+
					"frame-ancestors 'none'")

			// Restrict browser features
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

			next.ServeHTTP(w, r)
		})
	}
	logging.LogInfo("Security headers middleware initialized")

	// Initialize handlers
	handler := handlers.NewHandler(templates, sessionStore)
	logging.LogInfo("HTTP handlers initialized")

	// Application routes (all protected by the middleware stack below)
	appMux := http.NewServeMux()
	appMux.HandleFunc("/", handler.HandleHome)
	appMux.HandleFunc("/download/grade-scale", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleGradeScaleCSV(w, r, sessionStore)
	})
	appMux.HandleFunc("/download/student-results", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleStudentResultsCSV(w, r, sessionStore)
	})
	appMux.HandleFunc("/download/combined", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleCombinedCSV(w, r, sessionStore)
	})
	appMux.HandleFunc("/download/grade-scale-excel", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleGradeScaleExcel(w, r, sessionStore)
	})
	appMux.HandleFunc("/download/student-results-excel", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleStudentResultsExcel(w, r, sessionStore)
	})
	appMux.HandleFunc("/download/combined-excel", func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleCombinedExcel(w, r, sessionStore)
	})

	// Apply middleware stack once to all app routes
	protectedApp := securityHeaders(csrf.Handler(rateLimiter.RateLimitMiddleware(appMux)))

	// Top-level mux: healthz bypasses middleware, everything else goes through
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	mux.Handle("/", protectedApp)

	// Start periodic system statistics logging
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			logging.LogSystemStats()
		}
	}()

	// Log initial system statistics
	logging.LogSystemStats()

	// Start server
	logging.LogInfo("Server starting on port 8080")

	server := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	logging.LogInfo("Server configured with security timeouts",
		"read_timeout", "15s",
		"write_timeout", "15s",
		"idle_timeout", "60s",
		"max_header_bytes", "1MB")

	if err := server.ListenAndServe(); err != nil {
		logging.LogCritical("Server failed to start", err)
		os.Exit(1)
	}
}
