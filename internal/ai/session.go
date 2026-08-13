package ai

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// sessionTTL bounds how long a pending mutation may wait for user approval.
const sessionTTL = 15 * time.Minute

const (
	maxPendingSessions        = 100
	maxPendingSessionsPerUser = 5
)

// pendingSession captures the conversation state frozen at the moment a
// mutation tool requested user approval, so the agent loop can resume exactly
// where it paused once the user confirms.
type pendingSession struct {
	id            string
	messages      []Message // full history up to and including the assistant tool-call turn
	toolCall      ToolCall  // the mutation awaiting approval
	clusterID     uint
	clusterName   string
	userID        uint
	username      string
	clientIP      string
	correlationID string
	expiresAt     time.Time
}

type sessionTakeResult int

const (
	sessionTaken sessionTakeResult = iota
	sessionMissing
	sessionExpired
	sessionForbidden
)

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
func (s *sessionStore) save(sess *pendingSession) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked()
	if len(s.sessions) >= maxPendingSessions {
		return "", false
	}
	userSessions := 0
	for _, existing := range s.sessions {
		if existing.userID == sess.userID {
			userSessions++
		}
	}
	if userSessions >= maxPendingSessionsPerUser {
		return "", false
	}
	id := uuid.NewString()
	sess.id = id
	if sess.correlationID == "" {
		sess.correlationID = uuid.NewString()
	}
	sess.expiresAt = time.Now().Add(sessionTTL)
	s.sessions[id] = sess
	return id, true
}

// take removes and returns the session for id, or (nil, false) if it is absent
// or expired.
func (s *sessionStore) takeOwned(id string, userID uint) (*pendingSession, sessionTakeResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, sessionMissing
	}
	if time.Now().After(sess.expiresAt) {
		delete(s.sessions, id)
		return sess, sessionExpired
	}
	if sess.userID != userID {
		return sess, sessionForbidden
	}
	delete(s.sessions, id)
	return sess, sessionTaken
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
