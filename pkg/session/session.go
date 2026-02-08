package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
)

// Store manages user sessions with automatic cleanup
type Store struct {
	sessions map[string]*Data
	mutex    sync.RWMutex
}

// Data holds session information with expiration
type Data struct {
	PageData  models.PageData
	ExpiresAt time.Time
}

// NewStore creates a new session store and starts cleanup routine
func NewStore() *Store {
	store := &Store{
		sessions: make(map[string]*Data),
	}
	store.startCleanup()
	return store
}

// Set stores session data with automatic expiration
func (s *Store) Set(id string, data models.PageData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sessions[id] = &Data{
		PageData:  data,
		ExpiresAt: time.Now().Add(time.Duration(models.SessionTimeout) * time.Second),
	}

	logging.LogDebug("Session data stored",
		"session_id", id,
		"has_students", data.HasStudents,
		"student_count", len(data.Students),
		"expires_at", s.sessions[id].ExpiresAt.Format(time.RFC3339))
}

// Get retrieves session data if it exists and hasn't expired
func (s *Store) Get(id string) (models.PageData, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	sessionData, exists := s.sessions[id]
	if !exists {
		logging.LogDebug("Session not found", "session_id", id)
		return models.PageData{}, false
	}

	if time.Now().After(sessionData.ExpiresAt) {
		logging.LogDebug("Session expired",
			"session_id", id,
			"expired_at", sessionData.ExpiresAt.Format(time.RFC3339))

		delete(s.sessions, id)

		return models.PageData{}, false
	}

	logging.LogDebug("Session retrieved successfully",
		"session_id", id,
		"has_students", sessionData.PageData.HasStudents,
		"student_count", len(sessionData.PageData.Students))

	return sessionData.PageData, true
}

// Delete removes a session
func (s *Store) Delete(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.sessions[id]; exists {
		delete(s.sessions, id)
		logging.LogDebug("Session deleted", "session_id", id)
	}
}

// startCleanup runs a background goroutine to clean up expired sessions
func (s *Store) startCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.cleanupExpired()
		}
	}()
}

// cleanupExpired removes all expired sessions
func (s *Store) cleanupExpired() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, sessionData := range s.sessions {
		if now.After(sessionData.ExpiresAt) {
			delete(s.sessions, id)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		logging.LogInfo("Cleaned up expired sessions",
			"expired_count", expiredCount,
			"remaining_sessions", len(s.sessions))
	}
}

// GenerateSessionID creates a cryptographically secure random session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		logging.LogError("Failed to generate session ID", err)
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetSessionCount returns the current number of active sessions
func (s *Store) GetSessionCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.sessions)
}
