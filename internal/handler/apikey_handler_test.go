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
// Mock repos for APIKeyHandler tests
// ---------------------------------------------------------------------------

type stubAPIKeyRepo struct {
	keys   map[uint]*model.APIKey
	byHash map[string]*model.APIKey
	nextID uint
}

func newStubAPIKeyRepo() *stubAPIKeyRepo {
	return &stubAPIKeyRepo{
		keys:   make(map[uint]*model.APIKey),
		byHash: make(map[string]*model.APIKey),
		nextID: 1,
	}
}

func (r *stubAPIKeyRepo) Create(_ context.Context, key *model.APIKey) error {
	key.ID = r.nextID
	r.nextID++
	cp := *key
	r.keys[key.ID] = &cp
	r.byHash[key.KeyHash] = &cp
	return nil
}

func (r *stubAPIKeyRepo) GetByKeyHash(_ context.Context, hash string) (*model.APIKey, error) {
	k, ok := r.byHash[hash]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *k
	return &cp, nil
}

func (r *stubAPIKeyRepo) ListByUser(_ context.Context, userID uint) ([]model.APIKey, error) {
	var result []model.APIKey
	for _, k := range r.keys {
		if k.UserID == userID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (r *stubAPIKeyRepo) Delete(_ context.Context, id uint) error {
	k, ok := r.keys[id]
	if !ok {
		return errors.New("not found")
	}
	delete(r.byHash, k.KeyHash)
	delete(r.keys, id)
	return nil
}

var _ repository.APIKeyRepo = (*stubAPIKeyRepo)(nil)

type stubUserRepoForKey struct {
	users map[uint]*model.User
}

func newStubUserRepoForKey(users ...*model.User) *stubUserRepoForKey {
	r := &stubUserRepoForKey{users: make(map[uint]*model.User)}
	for _, u := range users {
		r.users[u.ID] = u
	}
	return r
}

func (r *stubUserRepoForKey) Create(_ context.Context, _ *model.User) error { return nil }
func (r *stubUserRepoForKey) GetByID(_ context.Context, id uint) (*model.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}
func (r *stubUserRepoForKey) GetByUsername(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.New("not found")
}
func (r *stubUserRepoForKey) Update(_ context.Context, _ *model.User) error { return nil }
func (r *stubUserRepoForKey) Delete(_ context.Context, _ uint) error        { return nil }
func (r *stubUserRepoForKey) List(_ context.Context) ([]model.User, error)  { return nil, nil }
func (r *stubUserRepoForKey) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.New("not found")
}
func (r *stubUserRepoForKey) GetByOAuthID(_ context.Context, _, _ string) (*model.User, error) {
	return nil, errors.New("not found")
}

var _ repository.UserRepo = (*stubUserRepoForKey)(nil)

// fakeAuth injects userID and userRole into the gin context for testing.
func fakeAuth(userID uint, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Next()
	}
}

func setupAPIKeyHandler(t *testing.T) (*gin.Engine, *APIKeyHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	user := &model.User{ID: 1, Username: "testuser", Role: "viewer"}
	user.ID = 1
	apiKeyRepo := newStubAPIKeyRepo()
	userRepo := newStubUserRepoForKey(user)
	svc := service.NewAPIKeyService(apiKeyRepo, userRepo)
	handler := NewAPIKeyHandler(svc)

	router := gin.New()
	router.Use(fakeAuth(1, "viewer"))
	router.POST("/api/v1/api-keys", handler.Generate)
	router.GET("/api/v1/api-keys", handler.List)
	router.DELETE("/api/v1/api-keys/:id", handler.Revoke)
	return router, handler
}

func TestAPIKeyHandler_Generate_Success(t *testing.T) {
	router, _ := setupAPIKeyHandler(t)

	body := map[string]interface{}{
		"name": "my-key",
	}
	w := performRequest(router, "POST", "/api/v1/api-keys", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var data service.GenerateKeyResponse
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "my-key", data.Name)
	assert.NotEmpty(t, data.PlainKey)
}

func TestAPIKeyHandler_Generate_WithExpiry(t *testing.T) {
	router, _ := setupAPIKeyHandler(t)

	expiry := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := map[string]interface{}{
		"name":      "expiring-key",
		"expiresAt": expiry,
	}
	w := performRequest(router, "POST", "/api/v1/api-keys", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestAPIKeyHandler_Generate_MissingName(t *testing.T) {
	router, _ := setupAPIKeyHandler(t)

	w := performRequest(router, "POST", "/api/v1/api-keys", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestAPIKeyHandler_Generate_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKeyRepo := newStubAPIKeyRepo()
	userRepo := newStubUserRepoForKey()
	svc := service.NewAPIKeyService(apiKeyRepo, userRepo)
	handler := NewAPIKeyHandler(svc)

	router := gin.New()
	// No fakeAuth — userID=0
	router.POST("/api/v1/api-keys", handler.Generate)

	w := performRequest(router, "POST", "/api/v1/api-keys", map[string]string{"name": "key"})

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40100, resp.Code)
}

func TestAPIKeyHandler_List_Success(t *testing.T) {
	router, _ := setupAPIKeyHandler(t)

	// Generate a key first.
	performRequest(router, "POST", "/api/v1/api-keys", map[string]string{"name": "key1"})

	w := performRequest(router, "GET", "/api/v1/api-keys", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestAPIKeyHandler_Revoke_InvalidID(t *testing.T) {
	router, _ := setupAPIKeyHandler(t)

	w := performRequest(router, "DELETE", "/api/v1/api-keys/abc", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}
