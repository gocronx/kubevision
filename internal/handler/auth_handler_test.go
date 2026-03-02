package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/config"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	testJWTSecret     = "test-secret-for-auth-handler-tests"
	testAccessTokenTTL  = 15 * time.Minute
	testRefreshTokenTTL = 24 * time.Hour
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupAuthTestDB creates an in-memory SQLite DB with migrations and seed data.
// Each call creates an isolated database using a unique DSN.
func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
	}
	logger, _ := zap.NewDevelopment()
	db, err := repository.NewDB(cfg, logger)
	require.NoError(t, err)
	return db
}

// setupAuthHandler creates a real AuthHandler backed by a test SQLite DB.
func setupAuthHandler(t *testing.T) (*AuthHandler, *auth.JWTManager) {
	t.Helper()
	db := setupAuthTestDB(t)
	userRepo := repository.NewUserRepo(db)
	jwtManager := auth.NewJWTManager(testJWTSecret, testAccessTokenTTL, testRefreshTokenTTL)
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		Database:   config.DatabaseConfig{Driver: "sqlite"},
		EncryptKey: "test-encrypt-key-32-bytes-padding",
	}
	authService := service.NewAuthService(userRepo, jwtManager, cfg, logger)
	handler := NewAuthHandler(authService)
	return handler, jwtManager
}

// authAPIResponse is the JSON structure returned by the API.
type authAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// loginResponseData is the data field when login succeeds.
type loginResponseData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

func performRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthHandler_Login_ValidCredentials(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	w := performRequest(router, "POST", "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "business code should be 0 (success)")
	assert.Equal(t, "success", resp.Message)

	// Parse the data field.
	var data loginResponseData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.NotEmpty(t, data.AccessToken, "access token should not be empty")
	assert.NotEmpty(t, data.RefreshToken, "refresh token should not be empty")
	assert.Equal(t, "admin", data.User.Username)
	assert.Equal(t, "super-admin", data.User.Role)
	assert.NotZero(t, data.User.ID)
}

func TestAuthHandler_Login_MissingUsername(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	// Missing username.
	body := map[string]string{
		"password": "admin123",
	}
	w := performRequest(router, "POST", "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid")
	assert.Contains(t, resp.Message, "required")
}

func TestAuthHandler_Login_MissingPassword(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	// Missing password.
	body := map[string]string{
		"username": "admin",
	}
	w := performRequest(router, "POST", "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid")
	assert.Contains(t, resp.Message, "required")
}

func TestAuthHandler_Login_EmptyBody(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	// Empty JSON body.
	w := performRequest(router, "POST", "/api/v1/auth/login", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for empty body")
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body := map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	}
	w := performRequest(router, "POST", "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code, "should return CodeUnauthorized")
	assert.Contains(t, resp.Message, "invalid")
}

func TestAuthHandler_Login_NonExistentUser(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	body := map[string]string{
		"username": "nonexistent",
		"password": "anypassword",
	}
	w := performRequest(router, "POST", "/api/v1/auth/login", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code, "should return CodeUnauthorized for non-existent user")
}

func TestAuthHandler_Refresh_ValidToken(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)
	router.POST("/api/v1/auth/refresh", handler.Refresh)

	// First, login to get a valid refresh token.
	loginBody := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	loginW := performRequest(router, "POST", "/api/v1/auth/login", loginBody)

	var loginResp authAPIResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	require.Equal(t, 0, loginResp.Code)

	var loginData loginResponseData
	err = json.Unmarshal(loginResp.Data, &loginData)
	require.NoError(t, err)
	require.NotEmpty(t, loginData.RefreshToken)

	// Now use the refresh token.
	refreshBody := map[string]string{
		"refreshToken": loginData.RefreshToken,
	}
	refreshW := performRequest(router, "POST", "/api/v1/auth/refresh", refreshBody)

	assert.Equal(t, http.StatusOK, refreshW.Code)

	var refreshResp authAPIResponse
	err = json.Unmarshal(refreshW.Body.Bytes(), &refreshResp)
	require.NoError(t, err)
	assert.Equal(t, 0, refreshResp.Code, "refresh should succeed")

	var refreshData loginResponseData
	err = json.Unmarshal(refreshResp.Data, &refreshData)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshData.AccessToken, "new access token should not be empty")
	assert.NotEmpty(t, refreshData.RefreshToken, "new refresh token should not be empty")
	assert.Equal(t, "admin", refreshData.User.Username)
	assert.Equal(t, "super-admin", refreshData.User.Role)
}

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/refresh", handler.Refresh)

	// Missing refreshToken field.
	body := map[string]string{}
	w := performRequest(router, "POST", "/api/v1/auth/refresh", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid")
	assert.Contains(t, resp.Message, "refreshToken")
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	handler, _ := setupAuthHandler(t)

	router := gin.New()
	router.POST("/api/v1/auth/refresh", handler.Refresh)

	body := map[string]string{
		"refreshToken": "invalid.jwt.token",
	}
	w := performRequest(router, "POST", "/api/v1/auth/refresh", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40101, resp.Code, "should return CodeTokenExpired for invalid token")
}

// Verify the response package is properly imported (compile-time check).
var _ = response.Response{}
