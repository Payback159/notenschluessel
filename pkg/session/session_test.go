package session

import (
	"sync"
	"testing"
	"time"

	"github.com/payback159/notenschluessel/pkg/logging"
	"github.com/payback159/notenschluessel/pkg/models"
)

func init() {
	logging.InitLogger()
}

// --- GenerateSessionID ---

func TestGenerateSessionID_UniqueAndLength(t *testing.T) {
	id1, err1 := GenerateSessionID()
	id2, err2 := GenerateSessionID()

	if err1 != nil || err2 != nil {
		t.Fatalf("GenerateSessionID returned error: %v / %v", err1, err2)
	}
	if len(id1) != 32 {
		t.Errorf("expected 32 hex chars, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("two generated IDs should not be equal")
	}
}

func TestGenerateSessionID_HexEncoded(t *testing.T) {
	id, _ := GenerateSessionID()
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex char %c in session ID %s", c, id)
		}
	}
}

// --- Store Set/Get ---

func TestStore_SetAndGet(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	data := models.PageData{
		MaxPoints:  100,
		HasResults: true,
	}
	store.Set("test-id", data)

	got, ok := store.Get("test-id")
	if !ok {
		t.Fatal("expected session to exist")
	}
	if got.MaxPoints != 100 {
		t.Errorf("MaxPoints: want 100, got %d", got.MaxPoints)
	}
	if !got.HasResults {
		t.Error("HasResults should be true")
	}
}

func TestStore_GetNonExistent(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	_, ok := store.Get("does-not-exist")
	if ok {
		t.Error("expected false for non-existent session")
	}
}

func TestStore_GetExpired(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	// Manually insert an expired session
	store.sessions["expired"] = &Data{
		PageData:  models.PageData{MaxPoints: 50},
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	_, ok := store.Get("expired")
	if ok {
		t.Error("expected false for expired session")
	}

	// Verify it was cleaned up
	store.mutex.Lock()
	_, stillExists := store.sessions["expired"]
	store.mutex.Unlock()
	if stillExists {
		t.Error("expired session should have been deleted from map")
	}
}

func TestStore_OverwriteSession(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	store.Set("id1", models.PageData{MaxPoints: 10})
	store.Set("id1", models.PageData{MaxPoints: 99})

	got, ok := store.Get("id1")
	if !ok {
		t.Fatal("session should exist")
	}
	if got.MaxPoints != 99 {
		t.Errorf("want 99, got %d", got.MaxPoints)
	}
}

// --- Delete ---

func TestStore_Delete(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}
	store.Set("del-me", models.PageData{MaxPoints: 5})

	store.Delete("del-me")

	_, ok := store.Get("del-me")
	if ok {
		t.Error("session should have been deleted")
	}
}

func TestStore_DeleteNonExistent(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}
	// Should not panic
	store.Delete("nope")
}

// --- GetSessionCount ---

func TestStore_GetSessionCount(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	if store.GetSessionCount() != 0 {
		t.Error("expected 0 sessions initially")
	}

	store.Set("a", models.PageData{})
	store.Set("b", models.PageData{})

	if store.GetSessionCount() != 2 {
		t.Errorf("expected 2 sessions, got %d", store.GetSessionCount())
	}
}

// --- cleanupExpired ---

func TestStore_CleanupExpired(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	// Add one valid and one expired session
	store.sessions["valid"] = &Data{
		PageData:  models.PageData{MaxPoints: 100},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	store.sessions["expired1"] = &Data{
		PageData:  models.PageData{MaxPoints: 50},
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.sessions["expired2"] = &Data{
		PageData:  models.PageData{MaxPoints: 30},
		ExpiresAt: time.Now().Add(-2 * time.Hour),
	}

	store.cleanupExpired()

	if store.GetSessionCount() != 1 {
		t.Errorf("expected 1 session after cleanup, got %d", store.GetSessionCount())
	}
	if _, ok := store.Get("valid"); !ok {
		t.Error("valid session should still exist")
	}
}

// --- Concurrency ---

func TestStore_ConcurrentAccess(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	var wg sync.WaitGroup
	// Run many concurrent Set/Get/Delete operations
	for i := 0; i < 100; i++ {
		wg.Add(3)
		id := "id-" + string(rune('A'+i%26))

		go func() {
			defer wg.Done()
			store.Set(id, models.PageData{MaxPoints: 42})
		}()
		go func() {
			defer wg.Done()
			store.Get(id)
		}()
		go func() {
			defer wg.Done()
			store.Delete(id)
		}()
	}
	wg.Wait()
	// Test passes if no race condition / panic occurs
}

func TestStore_SessionDataWithStudents(t *testing.T) {
	store := &Store{sessions: make(map[string]*Data)}

	data := models.PageData{
		MaxPoints:   100,
		HasResults:  true,
		HasStudents: true,
		Students: []models.Student{
			{Name: "Alice", Points: 85, Grade: 1},
			{Name: "Bob", Points: 60, Grade: 3},
		},
		AverageGrade: 2.0,
		GradeBounds: []models.GradeBound{
			{Grade: 1, LowerBound: 85, UpperBound: 100},
			{Grade: 2, LowerBound: 70, UpperBound: 84.5},
		},
	}

	store.Set("full-data", data)
	got, ok := store.Get("full-data")
	if !ok {
		t.Fatal("session should exist")
	}
	if len(got.Students) != 2 {
		t.Errorf("want 2 students, got %d", len(got.Students))
	}
	if got.Students[0].Name != "Alice" {
		t.Errorf("want Alice, got %s", got.Students[0].Name)
	}
	if got.AverageGrade != 2.0 {
		t.Errorf("want average 2.0, got %.2f", got.AverageGrade)
	}
}
