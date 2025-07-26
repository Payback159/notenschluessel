package logging

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"
)

var (
	logger    *slog.Logger
	startTime time.Time
)

// InitLogger initializes the structured logging system
func InitLogger() {
	startTime = time.Now()

	// Configure JSON logging for production
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	LogInfo("Logger initialized successfully",
		"handler", "json",
		"level", "info",
		"source_enabled", true)
}

// LogInfo logs an informational message
func LogInfo(msg string, args ...any) {
	logger.Info(msg, args...)
}

// LogError logs an error message with error details
func LogError(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err}, args...)
	logger.Error(msg, allArgs...)
}

// LogWarn logs a warning message
func LogWarn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// LogDebug logs a debug message
func LogDebug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// LogCritical logs a critical error and also writes to stderr
func LogCritical(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err, "severity", "critical"}, args...)
	logger.Error(msg, allArgs...)
	log.Printf("CRITICAL: %s: %v", msg, err)
}

// LogPerformance logs performance metrics
func LogPerformance(operation string, duration time.Duration, args ...any) {
	allArgs := append([]any{
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"duration_str", duration.String(),
	}, args...)
	logger.Info("Performance metric", allArgs...)
}

// LogSecurityEvent logs security-related events
func LogSecurityEvent(event string, severity string, args ...any) {
	allArgs := append([]any{
		"event_type", "security",
		"security_event", event,
		"severity", severity,
	}, args...)
	logger.Warn("Security event", allArgs...)
}

// LogSystemStats logs system statistics and resource usage
func LogSystemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)

	LogInfo("System statistics",
		"uptime_seconds", int(uptime.Seconds()),
		"uptime_str", uptime.String(),
		"goroutines", runtime.NumGoroutine(),
		"memory_alloc_mb", bToMb(m.Alloc),
		"memory_total_alloc_mb", bToMb(m.TotalAlloc),
		"memory_sys_mb", bToMb(m.Sys),
		"gc_runs", m.NumGC,
		"next_gc_mb", bToMb(m.NextGC))
}

// LogHTTPRequest logs HTTP request details
func LogHTTPRequest(method, path, userAgent, ip string, statusCode int, duration time.Duration) {
	LogInfo("HTTP request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"user_agent", userAgent,
		"client_ip", ip)
}

// LogFileOperation logs file upload/download operations
func LogFileOperation(operation, filename string, size int64, duration time.Duration, success bool, args ...any) {
	allArgs := append([]any{
		"operation", operation,
		"filename", filename,
		"size_bytes", size,
		"size_mb", float64(size) / (1024 * 1024),
		"duration_ms", duration.Milliseconds(),
		"success", success,
	}, args...)

	if success {
		LogInfo("File operation completed", allArgs...)
	} else {
		LogError("File operation failed", fmt.Errorf("operation failed"), allArgs...)
	}
}

// LogCalculation logs grade calculation operations
func LogCalculation(maxPoints int, minPoints float64, breakPoint float64, studentCount int, duration time.Duration, success bool) {
	LogInfo("Grade calculation",
		"max_points", maxPoints,
		"min_points", minPoints,
		"break_point_percent", breakPoint,
		"student_count", studentCount,
		"duration_ms", duration.Milliseconds(),
		"success", success)
}

// Helper function to convert bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
