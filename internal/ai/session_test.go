package ai

import (
	"testing"
	"time"
)

func TestSessionSaveAndTake(t *testing.T) {
	s := newSessionStore()
	id := s.save(&pendingSession{clusterName: "prod", userID: 7, toolCall: ToolCall{ID: "c1"}})
	if id == "" {
		t.Fatal("expected a session id")
	}

	got, result := s.takeOwned(id, 7)
	if result != sessionTaken {
		t.Fatal("expected to find the session")
	}
	if got.clusterName != "prod" || got.toolCall.ID != "c1" {
		t.Fatalf("unexpected session: %+v", got)
	}

	// Sessions are single-use.
	if _, result := s.takeOwned(id, 7); result != sessionMissing {
		t.Fatal("session should be consumed after take")
	}
}

func TestSessionExpiry(t *testing.T) {
	s := newSessionStore()
	id := s.save(&pendingSession{userID: 7})
	// Force expiry.
	s.mu.Lock()
	s.sessions[id].expiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()

	if _, result := s.takeOwned(id, 7); result != sessionExpired {
		t.Fatal("expired session should not be returned")
	}
}

func TestSessionTakeMissing(t *testing.T) {
	s := newSessionStore()
	if _, result := s.takeOwned("does-not-exist", 7); result != sessionMissing {
		t.Fatal("missing session should report not found")
	}
}

func TestSessionOwnerMismatchDoesNotConsume(t *testing.T) {
	s := newSessionStore()
	id := s.save(&pendingSession{userID: 7})

	if _, result := s.takeOwned(id, 8); result != sessionForbidden {
		t.Fatalf("result = %v, want forbidden", result)
	}
	if _, result := s.takeOwned(id, 7); result != sessionTaken {
		t.Fatalf("owner could not consume session after mismatch: %v", result)
	}
}
