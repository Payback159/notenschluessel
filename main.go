package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/csrf"
	"github.com/payback159/notenschluessel/pkg/downloads"
	"github.com/payback159/notenschluessel/pkg/handlers"
	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/security"
	"github.com/payback159/notenschluessel/pkg/session"
)

func main() {
	// Initialize structured logging
	logging.InitLogger()

	logging.LogInfo("Starting Notenschluessel service",
		"version", "v1.0.0",
		"environment", os.Getenv("ENV"))

	// Check for health check flag
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		// Simple health check - just exit with 0 if we can start
		fmt.Println("OK")
		os.Exit(0)
	}

	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))
	logging.LogInfo("Templates loaded successfully")

	// Generate CSRF key
	csrfKey := make([]byte, 32)
	if _, err := rand.Read(csrfKey); err != nil {
		logging.LogCritical("Failed to generate CSRF key", err)
		log.Fatal("Failed to generate CSRF key:", err)
	}
	logging.LogDebug("CSRF key generated successfully")

	// Configure CSRF protection
	var csrfMiddleware func(http.Handler) http.Handler
	if os.Getenv("ENV") == "production" {
		csrfMiddleware = csrf.Protect(csrfKey, csrf.Secure(true))
		logging.LogInfo("CSRF protection enabled for production")
	} else {
		// Development mode - disable CSRF protection for easier development
		csrfMiddleware = func(h http.Handler) http.Handler {
			return h // No CSRF protection in development
		}
		logging.LogInfo("CSRF protection disabled for development")
	}

	// Initialize session store
	sessionStore := session.NewStore()
	logging.LogInfo("Session store initialized")

	// Initialize rate limiter
	rateLimiter := security.NewRateLimiter()
	logging.LogInfo("Rate limiter initialized")

	// Initialize handlers
	handler := handlers.NewHandler(templates, sessionStore)
	logging.LogInfo("HTTP handlers initialized")

	// Create multiplexer
	mux := http.NewServeMux()

	// Register main handler with middleware
	mux.HandleFunc("/", security.SecurityHeaders(rateLimiter.RateLimitMiddleware(handler.HandleHome)))

	// Add CSV download handlers (no rate limiting for downloads)
	mux.HandleFunc("/download/grade-scale", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleGradeScaleCSV(w, r, sessionStore)
	}))
	mux.HandleFunc("/download/student-results", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleStudentResultsCSV(w, r, sessionStore)
	}))
	mux.HandleFunc("/download/combined", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleCombinedCSV(w, r, sessionStore)
	}))

	// Add Excel download handlers
	mux.HandleFunc("/download/grade-scale-excel", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleGradeScaleExcel(w, r, sessionStore)
	}))
	mux.HandleFunc("/download/student-results-excel", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleStudentResultsExcel(w, r, sessionStore)
	}))
	mux.HandleFunc("/download/combined-excel", security.SecurityHeaders(func(w http.ResponseWriter, r *http.Request) {
		downloads.HandleCombinedExcel(w, r, sessionStore)
	}))

	// Apply CSRF middleware to the entire mux
	protectedHandler := csrfMiddleware(mux)

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
	fmt.Println("Server läuft auf http://localhost:8080")

	server := &http.Server{
		Addr:           ":8080",
		Handler:        protectedHandler,
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

	log.Fatal(server.ListenAndServe())
}
