# Copilot Instructions for Notenschlüssel

## Project Overview

**Notenschlüssel** is an Austrian grading scale calculator built in **Go 1.26** with a focus on security, structured logging, and educational use (DSGVO compliant). It calculates grade boundaries using a Austrian 1-5 grading scale with configurable breakpoints, supports CSV and manual student input, and includes CSV/Excel export capabilities.

### Architecture

- **Monolithic HTTP service** (Go 1.26 stdlib only)
- **Core layers**: handlers → calculator → models, with security/logging/session support
- **Middleware stack**: Security headers → CSRF protection (native Go 1.25) → Rate limiting → Handler
- **Data flow**: Form input → Calculation → Session storage → Template rendering or export

## Critical Patterns & Conventions

### Go 1.25+ Native CSRF Protection

⚠️ **Key architectural decision**: Uses `http.NewCrossOriginProtection()` instead of external libraries.

- CSRF middleware wraps all handlers in `main.go` (line 106-120)
- Configure trusted origins at startup: `csrf.AddTrustedOrigin("https://hostname")`
- Env-aware configuration: production uses HTTPS only, dev allows localhost variants
- **Never add CSRF tokens to forms** - browser headers (`Sec-Fetch-Site`, `Origin`) are validated instead
- Wrapping pattern: `csrf.Handler(middleware(handlerFunc))`

### Structured Logging (slog, JSON output)

All logging must use the `logging` package, never `fmt.Print` for application events:

```go
// ✅ DO: Structured logging with key-value pairs
logging.LogInfo("User submitted form",
    "session_id", sessionID,
    "max_points", maxPoints,
    "students_count", len(students))

// ❌ DON'T: Printf debugging
fmt.Printf("maxPoints: %d\n", maxPoints)
```

Key functions:

- `logging.LogInfo(msg string, args ...any)` - informational events
- `logging.LogError(msg string, err error, args ...any)` - errors with stack context
- `logging.LogDebug(msg string, args ...any)` - low-level details (only when debugging)
- `logging.LogWarn(msg string, args ...any)` - warnings
- `logging.LogCritical(msg string, err error, args ...any)` - critical failures
- `logging.LogPerformance(operation string, duration time.Duration, args ...any)` - performance metrics
- `logging.LogSecurityEvent(msg string, severity string, args ...any)` - security-relevant (rate limits, auth failures)
- `logging.LogSystemStats()` - called every 10 minutes for resource usage

### Session Management

Sessions store grade calculations and file uploads per user (see `pkg/session/`):

```go
// Store data
sessionStore.Set(sessionID, pageData)

// Retrieve and use
data, ok := sessionStore.Get(sessionID)
```

- SessionID passed to all templates and handlers for audit trails
- Session timeout: 24 hours (configurable in models)
- No external session backend - in-memory only (stateless for horizontal scaling in future)

### Security Middleware Stack (main.go:65-103)

Applied in order: Security headers → CSRF → Rate limiting → Handler

**Security headers set**:

- `X-Frame-Options: DENY` - prevent clickjacking
- `Content-Security-Policy: default-src 'self'` - strict CSP with unsafe-inline for styles/scripts only (educational tool)
- `Strict-Transport-Security: max-age=31536000` (production only)
- `X-Content-Type-Options: nosniff`, `X-XSS-Protection: 1; mode=block`

**Rate limiting** (per IP, 10 requests/minute, burst 20):

```go
rateLimiter := security.NewRateLimiter() // Create once in main
// Wrapped as middleware: rateLimiter.RateLimitMiddleware(handler)
```

### Form Handling & Validation

- All POST forms lack CSRF tokens (native Go 1.25 protection)
- Input validation in `handlers.HandleHome()`:
  - `maxPoints`: required, positive integer
  - `minPoints`: required, positive float
  - `breakPointPercent`: 1-99 range
  - `inputMode`: `csv` or `manual`
  - Student input: CSV upload or manual table entries (mutually exclusive)
  - CSV file (optional in CSV mode): max 10MB, validated CSV structure
  - Empty student list is allowed (grade scale only)
  - Degenerate scale combinations are rejected via `calculator.ValidateGradeBounds()`
- Validation errors returned as `Message{Type: "error", Text: "..."}` in template data
- ✅ Safe template rendering via `h.executeTemplateSafe()` to catch panics

### File Upload Processing

Located in `pkg/calculator/ParseCSVFile()`:

- Accepts `.csv` files up to 10MB
- Expects header row: `Name,Punkte` (German locale)
- Numeric validation: points must be floats in range 0-1000
- Returns `[]models.Student` or error with logging
- Used only for grade calculation, NOT stored permanently (session-only)

### Grade Calculation Algorithm

`pkg/calculator/CalculateGradeBounds()` implements the Austrian 1-5 scale.

The **breakpoint (Knickpunkt)** is the passing threshold: it marks the lower bound of grade 4 (Genügend). Everything below the breakpoint is grade 5 (Nicht Genügend). The range from the breakpoint up to the maximum points is split into four equal segments for grades 4, 3, 2 and 1.

With `breakAbs = maxPoints * breakPointPercent/100` and `segment = (maxPoints - breakAbs) / 4`:

- Grade 1: `breakAbs + 3*segment` to maxPoints
- Grade 2: `breakAbs + 2*segment` to grade 1 lower bound
- Grade 3: `breakAbs + 1*segment` to grade 2 lower bound
- Grade 4: `breakAbs` (breakpoint) to grade 3 lower bound
- Grade 5: 0 to below the breakpoint

Example (max 45, breakpoint 50% → 22.5): Grade 4 starts at 22.5, grade 5 covers 0–22.

**Rounding**: All boundaries rounded to nearest `minPoints` increment to avoid ambiguity.

### Export Handlers (CSV & Excel)

Six export endpoints (all CSRF-protected):

1. `/download/grade-scale` - Boundaries as CSV
2. `/download/student-results` - Student names + grades as CSV
3. `/download/combined` - Full data as CSV
4. `/download/*-excel` variants - Same data as `.xlsx`

Handlers in `pkg/downloads/`:

- Retrieve data from session store
- Set `Content-Disposition: attachment; filename=...` header
- Set proper MIME types (`text/csv`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`)
- Log export events for audit

### Environment Configuration

```bash
# .env file
ENV=production              # 'production' or 'development'
HOSTNAME=notenschluessel.example.com  # Used for CSRF trusted origins
```

- **Production**: HTTPS only, strict CSP, enforced HSTS
- **Development**: HTTP/HTTPS localhost allowed, relaxed CSP inline scripts
- Running with `--health-check` flag returns "OK" and exits (for container probes)

## Project Structure

```
notenschluessel/
├── main.go                    # Server setup, middleware wiring, route definitions
├── pkg/
│   ├── calculator/            # Grade boundary & student grade logic
│   ├── downloads/             # CSV/Excel export generation
│   ├── handlers/              # HTTP request handlers (main business logic)
│   ├── logging/               # Structured slog wrapper (JSON output)
│   ├── models/                # Data types, constants (security limits, rates)
│   ├── security/              # Rate limiting, IP extraction, CSV validation
│   └── session/               # In-memory session store
├── templates/                 # HTML templates (single index.html)
├── dockerfile                 # Multi-stage build, scratch base, non-root user
└── compose.yml                # Docker Compose for local dev
```

## Development Workflows

### Building & Running Locally

```bash
# Run directly
go run main.go

# With Docker
docker compose up

# Health check
./main --health-check
```

### Testing

Existing unit tests cover calculator, downloads, handlers, security and session packages:

```bash
go test ./pkg/calculator -v
go test ./... -cover
```

### Adding New Features

1. **New calculation variant**: Add to `pkg/calculator/`, call from `HandleHome()`
2. **New export format**: Add handler in `pkg/downloads/`, register route in `main.go`, wrap with CSRF/security
3. **New endpoint**: Always wrap with full middleware stack: `securityHeaders(csrf.Handler(rateLimiter.RateLimitMiddleware(...)))`
4. **New session data**: Use `sessionStore.Set(sessionID, pageData)` pattern
5. **New error case**: Log with `logging.LogError()` and return `Message{Type: "error", ...}` to template

### Debugging

- Enable detailed logs: Change `logging.go` line 20 `slog.LevelInfo` → `slog.LevelDebug`
- Check rate limit state: `RateLimiter.limiters` map (exported for debugging)
- Session data inspection: `SessionStore.sessions` map
- Template rendering: `h.executeTemplateSafe()` catches and logs panics

## Key Dependencies

- **excelize** (`github.com/xuri/excelize`) - Excel file generation
- **golang.org/x/time/rate** - Token bucket rate limiting
- **stdlib only**: No external HTTP framework, use `net/http` and `html/template` directly

## Common Mistakes to Avoid

1. ❌ Adding CSRF tokens to form fields - Go 1.25 validates headers instead
2. ❌ Using `fmt.Print*` for app events - Always use `logging` package
3. ❌ Forgetting to wrap handlers with `securityHeaders(csrf.Handler(...))`
4. ❌ Combining CSV and manual student input in one request
5. ❌ Logging student names in warning/error paths
6. ❌ Treating session ID as secret - It's logged and should be unpredictable but not encryption-strength

## Important Files for Reference

- **Security architecture**: `main.go:65-103` (middleware setup)
- **CSRF configuration**: `main.go:41-53` (trusted origins logic)
- **Form processing**: `pkg/handlers/handlers.go:HandleHome()`
- **Grading algorithm**: `pkg/calculator/calculator.go:CalculateGradeBounds()`
- **Exports**: `pkg/downloads/` (all three formats)
- **Logging events**: `pkg/logging/logging.go` (all LogX functions)
