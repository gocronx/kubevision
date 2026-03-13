package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

// ---------------------------------------------------------------------------
// Mock UserRepo for auth middleware tests
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	users map[uint]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uint]*model.User)}
}

func (m *mockUserRepo) addUser(u *model.User) {
	m.users[u.ID] = u
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uint) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) Update(_ context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) GetByOAuthID(_ context.Context, provider, oauthID string) (*model.User, error) {
	for _, u := range m.users {
		if u.AuthProvider == provider && u.OAuthID == oauthID {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) Delete(_ context.Context, id uint) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) List(_ context.Context) ([]model.User, error) {
	var result []model.User
	for _, u := range m.users {
		result = append(result, *u)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret-key-for-middleware-tests", 15*time.Minute, 7*24*time.Hour)
}

// apiResponse mirrors the response.Response struct for JSON decoding in tests.
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// performRequest creates a test gin engine with the AuthMiddleware and a
// downstream handler that records whether it was invoked. It returns the
// response recorder and whether the downstream handler was called.
func performRequest(jwtMgr *auth.JWTManager, repo *mockUserRepo, req *http.Request) (*httptest.ResponseRecorder, bool) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	handlerCalled := false
	var capturedUserID uint
	var capturedUsername string
	var capturedRole string

	router := gin.New()
	router.Use(AuthMiddleware(jwtMgr, repo, nil))
	router.GET("/protected", func(c *gin.Context) {
		handlerCalled = true
		capturedUserID = GetUserID(c)
		capturedUsername = GetUsername(c)
		capturedRole = GetUserRole(c)

		response.Success(c, gin.H{
			"userID":   capturedUserID,
			"username": capturedUsername,
			"role":     capturedRole,
		})
	})

	router.ServeHTTP(w, req)
	_ = capturedUserID
	_ = capturedUsername
	_ = capturedRole
	return w, handlerCalled
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeUnauthorized {
		t.Errorf("expected code %d, got %d", bizerr.CodeUnauthorized, resp.Code)
	}
	if resp.Message != "missing authorization header" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no Bearer prefix", "Token abc123"},
		{"only one part", "Bearertoken"},
		{"Basic auth", "Basic dXNlcjpwYXNz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jwtMgr := newTestJWTManager()
			repo := newMockUserRepo()

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tc.header)
			w, handlerCalled := performRequest(jwtMgr, repo, req)

			if handlerCalled {
				t.Error("downstream handler should NOT have been called")
			}

			var resp apiResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Code != bizerr.CodeUnauthorized {
				t.Errorf("expected code %d, got %d", bizerr.CodeUnauthorized, resp.Code)
			}
		})
	}
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeUnauthorized {
		t.Errorf("expected code %d, got %d", bizerr.CodeUnauthorized, resp.Code)
	}
	if resp.Message != "empty token" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	repo.addUser(&model.User{
		ID:           42,
		Username:     "testuser",
		Role:         "admin",
		IsActive:     true,
		TokenVersion: 1,
	})

	claims := &auth.TokenClaims{
		UserID:       42,
		Username:     "testuser",
		Role:         "admin",
		TokenVersion: 1,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if !handlerCalled {
		t.Error("downstream handler should have been called")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeSuccess {
		t.Errorf("expected code %d, got %d (message: %s)", bizerr.CodeSuccess, resp.Code, resp.Message)
	}

	// Verify the context values were set correctly by decoding the data field.
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}

	if uint(data["userID"].(float64)) != 42 {
		t.Errorf("expected userID 42 in response data, got %v", data["userID"])
	}
	if data["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", data["username"])
	}
	if data["role"] != "admin" {
		t.Errorf("expected role 'admin', got %v", data["role"])
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Create a JWT manager with a very short TTL to force expiry.
	jwtMgr := auth.NewJWTManager("test-secret", 1*time.Millisecond, 7*24*time.Hour)
	repo := newMockUserRepo()

	repo.addUser(&model.User{
		ID:           1,
		Username:     "testuser",
		Role:         "admin",
		IsActive:     true,
		TokenVersion: 0,
	})

	claims := &auth.TokenClaims{
		UserID:       1,
		Username:     "testuser",
		Role:         "admin",
		TokenVersion: 0,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called for expired token")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeTokenExpired {
		t.Errorf("expected code %d, got %d", bizerr.CodeTokenExpired, resp.Code)
	}
}

func TestAuthMiddleware_InvalidJWTSignature(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	// Generate a token with a different secret.
	otherJWT := auth.NewJWTManager("other-secret", 15*time.Minute, 7*24*time.Hour)
	claims := &auth.TokenClaims{
		UserID:       1,
		Username:     "testuser",
		Role:         "admin",
		TokenVersion: 0,
	}
	tokenStr, err := otherJWT.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called for invalid signature")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeTokenExpired {
		t.Errorf("expected code %d, got %d", bizerr.CodeTokenExpired, resp.Code)
	}
}

func TestAuthMiddleware_RevokedTokenVersion(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	// User has token version 2 in the database.
	repo.addUser(&model.User{
		ID:           10,
		Username:     "revokeduser",
		Role:         "dev",
		IsActive:     true,
		TokenVersion: 2,
	})

	// Token was issued with version 1 (old version).
	claims := &auth.TokenClaims{
		UserID:       10,
		Username:     "revokeduser",
		Role:         "dev",
		TokenVersion: 1,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called for revoked token")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeTokenExpired {
		t.Errorf("expected code %d, got %d", bizerr.CodeTokenExpired, resp.Code)
	}
	if resp.Message != "token has been revoked" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestAuthMiddleware_DisabledUser(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	repo.addUser(&model.User{
		ID:           20,
		Username:     "disabled",
		Role:         "dev",
		IsActive:     false,
		TokenVersion: 0,
	})

	claims := &auth.TokenClaims{
		UserID:       20,
		Username:     "disabled",
		Role:         "dev",
		TokenVersion: 0,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called for disabled user")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeForbidden {
		t.Errorf("expected code %d, got %d", bizerr.CodeForbidden, resp.Code)
	}
}

func TestAuthMiddleware_UserNotFoundInDB(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	// Generate a valid token for user 99 but do NOT add the user to the repo.
	claims := &auth.TokenClaims{
		UserID:       99,
		Username:     "ghost",
		Role:         "admin",
		TokenVersion: 0,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if handlerCalled {
		t.Error("downstream handler should NOT have been called when user is not in DB")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != bizerr.CodeUnauthorized {
		t.Errorf("expected code %d, got %d", bizerr.CodeUnauthorized, resp.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Helper functions
// ---------------------------------------------------------------------------

func TestGetUserID_NoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	uid := GetUserID(c)
	if uid != 0 {
		t.Errorf("expected 0 when no userID in context, got %d", uid)
	}
}

func TestGetUsername_NoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	username := GetUsername(c)
	if username != "" {
		t.Errorf("expected empty string when no username in context, got %q", username)
	}
}

func TestGetUserRole_NoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	role := GetUserRole(c)
	if role != "" {
		t.Errorf("expected empty string when no role in context, got %q", role)
	}
}

func TestGetUserID_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "not-a-uint")

	uid := GetUserID(c)
	if uid != 0 {
		t.Errorf("expected 0 when userID is wrong type, got %d", uid)
	}
}

func TestGetUsername_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("username", 12345)

	username := GetUsername(c)
	if username != "" {
		t.Errorf("expected empty string when username is wrong type, got %q", username)
	}
}

func TestGetUserRole_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userRole", 12345)

	role := GetUserRole(c)
	if role != "" {
		t.Errorf("expected empty string when role is wrong type, got %q", role)
	}
}

// TestAuthMiddleware_DBRoleOverridesClaimRole verifies that when a user's role
// has been updated in the database after the JWT was issued, the middleware
// injects the current database role rather than the (stale) claim role.
// This is the core invariant introduced by the change on line 123 of auth.go.
func TestAuthMiddleware_DBRoleOverridesClaimRole(t *testing.T) {
	jwtMgr := newTestJWTManager()
	repo := newMockUserRepo()

	// DB now has the user promoted to "admin".
	repo.addUser(&model.User{
		ID:           55,
		Username:     "promoted",
		Role:         "admin",
		IsActive:     true,
		TokenVersion: 1,
	})

	// JWT was issued when the user was still a "viewer" (stale claim).
	claims := &auth.TokenClaims{
		UserID:       55,
		Username:     "promoted",
		Role:         "viewer",
		TokenVersion: 1,
	}
	tokenStr, err := jwtMgr.GenerateAccessToken(claims)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w, handlerCalled := performRequest(jwtMgr, repo, req)

	if !handlerCalled {
		t.Error("downstream handler should have been called")
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != bizerr.CodeSuccess {
		t.Errorf("expected success code, got %d (message: %s)", resp.Code, resp.Message)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}

	// The injected role must reflect the current DB value ("admin"),
	// NOT the stale JWT claim ("viewer").
	if data["role"] != "admin" {
		t.Errorf("expected DB role 'admin' to override stale JWT claim 'viewer', got %v", data["role"])
	}
}

func TestAuth_NoopMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	handlerCalled := false
	router := gin.New()
	router.Use(Auth())
	router.GET("/open", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/open", nil)
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("noop Auth middleware should pass through to handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
