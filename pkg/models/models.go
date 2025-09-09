package models

// Student represents a student with their name, points and calculated grade
type Student struct {
	Name   string
	Points float64
	Grade  int
}

// MessageType defines the type of message (success, error, warning)
type MessageType string

const (
	MessageSuccess MessageType = "success"
	MessageError   MessageType = "error"
	MessageWarning MessageType = "warning"
)

// Message represents a user feedback message
type Message struct {
	Type MessageType
	Text string
}

// PageData holds all data needed to render the HTML template
type PageData struct {
	MaxPoints          int
	MinPoints          float64
	BreakPointPercent  float64
	GradeBounds        []GradeBound
	Students           []Student
	AverageGrade       float64
	HasResults         bool
	HasStudents        bool
	CalculationSuccess bool
	Message            *Message
	SessionID          string
	// CSRFField removed - using Go 1.25+ native cross-origin protection
}

// GradeBound represents a grade boundary with upper and lower point limits
type GradeBound struct {
	Grade      int
	LowerBound float64
	UpperBound float64
}

// Constants for security limits
const (
	MaxFileSize    = 10 << 20 // 10MB
	MaxStudents    = 10000
	MaxNameLength  = 200
	SessionTimeout = 24 * 60 * 60 // 24 hours in seconds
	RateLimit      = 10           // requests per minute
	RateBurst      = 20           // burst capacity
)
