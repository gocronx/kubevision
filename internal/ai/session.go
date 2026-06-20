package ai

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// sessionTTL bounds how long a pending mutation may wait for user approval.
const sessionTTL = 15 * time.Minute

// pendingSession captures the conversation state frozen at the moment a
// mutation tool requested user approval, so the agent loop can resume exactly
// where it paused once the user confirms.
type pendingSession struct {
	id          string
	messages    []Message // full history up to and including the assistant tool-call turn
	toolCall    ToolCall  // the mutation awaiting approval
	clusterID   uint
	clusterName string
	userRole    string
	expiresAt   time.Time
}

// sessionStore is a TTL-bounded in-memory store of pending mutation sessions.
// Sessions are single-use: resuming consumes them.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*pendingSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*pendingSession)}
}

// save stores a session and returns its generated ID.
func (s *sessionStore) save(sess *pendingSession) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked()
	id := uuid.NewString()
	sess.id = id
	sess.expiresAt = time.Now().Add(sessionTTL)
	s.sessions[id] = sess
	return id
}

// take removes and returns the session for id, or (nil, false) if it is absent
// or expired.
func (s *sessionStore) take(id string) (*pendingSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	delete(s.sessions, id)
	if time.Now().After(sess.expiresAt) {
		return nil, false
	}
	return sess, true
}

// evictExpiredLocked drops expired sessions. Callers must hold s.mu.
func (s *sessionStore) evictExpiredLocked() {
	now := time.Now()
	for id, sess := range s.sessions {
		if now.After(sess.expiresAt) {
			delete(s.sessions, id)
		}
	}
}
