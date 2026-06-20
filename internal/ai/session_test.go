package ai

import (
	"testing"
	"time"
)

func TestSessionSaveAndTake(t *testing.T) {
	s := newSessionStore()
	id := s.save(&pendingSession{clusterName: "prod", toolCall: ToolCall{ID: "c1"}})
	if id == "" {
		t.Fatal("expected a session id")
	}

	got, ok := s.take(id)
	if !ok {
		t.Fatal("expected to find the session")
	}
	if got.clusterName != "prod" || got.toolCall.ID != "c1" {
		t.Fatalf("unexpected session: %+v", got)
	}

	// Sessions are single-use.
	if _, ok := s.take(id); ok {
		t.Fatal("session should be consumed after take")
	}
}

func TestSessionExpiry(t *testing.T) {
	s := newSessionStore()
	id := s.save(&pendingSession{})
	// Force expiry.
	s.mu.Lock()
	s.sessions[id].expiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()

	if _, ok := s.take(id); ok {
		t.Fatal("expired session should not be returned")
	}
}

func TestSessionTakeMissing(t *testing.T) {
	s := newSessionStore()
	if _, ok := s.take("does-not-exist"); ok {
		t.Fatal("missing session should report not found")
	}
}
