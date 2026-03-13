package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub TerminalSessionRepo for handler tests
// ---------------------------------------------------------------------------

type stubTerminalSessionRepo struct {
	sessions map[uint]*model.TerminalSession
	nextID   uint
}

func newStubTerminalSessionRepo() *stubTerminalSessionRepo {
	return &stubTerminalSessionRepo{
		sessions: make(map[uint]*model.TerminalSession),
		nextID:   1,
	}
}

func (r *stubTerminalSessionRepo) Create(_ context.Context, s *model.TerminalSession) error {
	s.ID = r.nextID
	s.CreatedAt = time.Now()
	r.nextID++
	cp := *s
	r.sessions[s.ID] = &cp
	return nil
}

func (r *stubTerminalSessionRepo) GetByID(_ context.Context, id uint) (*model.TerminalSession, error) {
	s, ok := r.sessions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *s
	return &cp, nil
}

func (r *stubTerminalSessionRepo) ListByUser(_ context.Context, userID uint) ([]model.TerminalSession, error) {
	var result []model.TerminalSession
	for _, s := range r.sessions {
		if userID == 0 || s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (r *stubTerminalSessionRepo) PurgeExpired(_ context.Context) (int64, error) {
	return 0, nil
}

var _ repository.TerminalSessionRepo = (*stubTerminalSessionRepo)(nil)

func setupTerminalSessionHandler(t *testing.T, userID uint, role string) (*gin.Engine, *TerminalSessionHandler, *stubTerminalSessionRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newStubTerminalSessionRepo()
	svc := service.NewTerminalSessionService(repo)
	handler := NewTerminalSessionHandler(svc)

	router := gin.New()
	router.Use(fakeAuth(userID, role))
	router.GET("/api/v1/terminal-sessions", handler.List)
	router.GET("/api/v1/terminal-sessions/:id", handler.Get)
	router.GET("/api/v1/terminal-sessions/:id/play", handler.Play)
	return router, handler, repo
}

// seedSession adds a session directly to the repo for testing.
func seedSession(repo *stubTerminalSessionRepo, userID uint, pod string) *model.TerminalSession {
	s := &model.TerminalSession{
		UserID:     userID,
		Cluster:    "prod",
		Namespace:  "default",
		Pod:        pod,
		Container:  "main",
		Recording:  `{"version":2}`,
		DurationMs: 5000,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}
	s.ID = repo.nextID
	s.CreatedAt = time.Now()
	repo.nextID++
	cp := *s
	repo.sessions[s.ID] = &cp
	return s
}

func TestTerminalSessionHandler_List_Admin(t *testing.T) {
	router, _, repo := setupTerminalSessionHandler(t, 1, "admin")

	seedSession(repo, 1, "pod-a")
	seedSession(repo, 2, "pod-b")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestTerminalSessionHandler_List_RegularUser(t *testing.T) {
	router, _, repo := setupTerminalSessionHandler(t, 1, "viewer")

	seedSession(repo, 1, "my-pod")
	seedSession(repo, 2, "other-pod")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestTerminalSessionHandler_List_Pagination(t *testing.T) {
	router, _, repo := setupTerminalSessionHandler(t, 1, "admin")

	for i := 0; i < 5; i++ {
		seedSession(repo, 1, "pod")
	}

	w := performRequest(router, "GET", "/api/v1/terminal-sessions?limit=2&offset=0", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestTerminalSessionHandler_Get_Success(t *testing.T) {
	router, _, repo := setupTerminalSessionHandler(t, 1, "viewer")

	s := seedSession(repo, 1, "test-pod")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions/"+uintToStr(s.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var meta service.TerminalSessionMeta
	require.NoError(t, json.Unmarshal(resp.Data, &meta))
	assert.Equal(t, "test-pod", meta.Pod)
}

func TestTerminalSessionHandler_Get_InvalidID(t *testing.T) {
	router, _, _ := setupTerminalSessionHandler(t, 1, "viewer")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions/abc", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestTerminalSessionHandler_Get_NotFound(t *testing.T) {
	router, _, _ := setupTerminalSessionHandler(t, 1, "viewer")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions/999", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestTerminalSessionHandler_Play_Success(t *testing.T) {
	router, _, repo := setupTerminalSessionHandler(t, 1, "viewer")

	s := seedSession(repo, 1, "play-pod")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions/"+uintToStr(s.ID)+"/play", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.NotEmpty(t, data["recording"])
}

func TestTerminalSessionHandler_Play_InvalidID(t *testing.T) {
	router, _, _ := setupTerminalSessionHandler(t, 1, "viewer")

	w := performRequest(router, "GET", "/api/v1/terminal-sessions/xyz/play", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}
