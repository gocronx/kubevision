package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

// ---------------------------------------------------------------------------
// Mock implementation of repository.UserRepo
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	users       map[string]*model.User // keyed by username
	usersById   map[uint]*model.User   // keyed by ID
	updateCalls int
	updateErr   error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:     make(map[string]*model.User),
		usersById: make(map[uint]*model.User),
	}
}

func (m *mockUserRepo) addUser(u *model.User) {
	m.users[u.Username] = u
	m.usersById[u.ID] = u
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.users[user.Username] = user
	m.usersById[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uint) (*model.User, error) {
	u, ok := m.usersById[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) Update(_ context.Context, user *model.User) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.users[user.Username] = user
	m.usersById[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range m.usersById {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) GetByOAuthID(_ context.Context, provider, oauthID string) (*model.User, error) {
	for _, u := range m.usersById {
		if u.AuthProvider == provider && u.OAuthID == oauthID {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) Delete(_ context.Context, id uint) error {
	for username, u := range m.users {
		if u.ID == id {
			delete(m.users, username)
			break
		}
	}
	delete(m.usersById, id)
	return nil
}

func (m *mockUserRepo) List(_ context.Context) ([]model.User, error) {
	var result []model.User
	for _, u := range m.usersById {
		result = append(result, *u)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret-key-for-unit-tests", 15*time.Minute, 7*24*time.Hour)
}

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestConfig() *config.Config {
	cfg := config.Default()
	cfg.EncryptKey = "test-encrypt-key-32-bytes-padding"
	return cfg
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return hash
}

// ---------------------------------------------------------------------------
// Tests: Login
// ---------------------------------------------------------------------------

func TestAuthService_Login(t *testing.T) {
	t.Run("correct credentials returns tokens and user info", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           1,
			Username:     "admin",
			PasswordHash: mustHashPassword(t, "correctpassword"),
			Role:         "admin",
			IsActive:     true,
			TokenVersion: 0,
		})

		result, err := svc.Login(context.Background(), "admin", "correctpassword")
		if err != nil {
			t.Fatalf("Login returned unexpected error: %v", err)
		}
		if result.FullTokens == nil {
			t.Fatal("expected FullTokens to be set")
		}
		resp := result.FullTokens

		if resp.AccessToken == "" {
			t.Error("expected non-empty access token")
		}
		if resp.RefreshToken == "" {
			t.Error("expected non-empty refresh token")
		}
		if resp.User.ID != 1 {
			t.Errorf("expected user ID 1, got %d", resp.User.ID)
		}
		if resp.User.Username != "admin" {
			t.Errorf("expected username 'admin', got %q", resp.User.Username)
		}
		if resp.User.Role != "admin" {
			t.Errorf("expected role 'admin', got %q", resp.User.Role)
		}

		// Verify that the access token is valid and can be parsed.
		claims, err := jwtMgr.ParseToken(resp.AccessToken)
		if err != nil {
			t.Fatalf("failed to parse generated access token: %v", err)
		}
		if claims.UserID != 1 {
			t.Errorf("token claims UserID = %d, want 1", claims.UserID)
		}
		if claims.Username != "admin" {
			t.Errorf("token claims Username = %q, want 'admin'", claims.Username)
		}

		// Verify last login time was updated.
		if repo.updateCalls == 0 {
			t.Error("expected Update to be called to set last login time")
		}
	})

	t.Run("wrong password returns unauthorized error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           1,
			Username:     "admin",
			PasswordHash: mustHashPassword(t, "correctpassword"),
			Role:         "admin",
			IsActive:     true,
		})

		result, err := svc.Login(context.Background(), "admin", "wrongpassword")
		if result != nil {
			t.Error("expected nil result for wrong password")
		}
		if err == nil {
			t.Fatal("expected error for wrong password, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeUnauthorized {
			t.Errorf("expected error code %d, got %d", bizerr.CodeUnauthorized, bizErr.Code)
		}
	})

	t.Run("non-existent user returns unauthorized error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		result, err := svc.Login(context.Background(), "nonexistent", "anypassword")
		if result != nil {
			t.Error("expected nil result for non-existent user")
		}
		if err == nil {
			t.Fatal("expected error for non-existent user, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeUnauthorized {
			t.Errorf("expected error code %d, got %d", bizerr.CodeUnauthorized, bizErr.Code)
		}
	})

	t.Run("disabled user returns forbidden error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           2,
			Username:     "disabled_user",
			PasswordHash: mustHashPassword(t, "password123"),
			Role:         "dev",
			IsActive:     false,
		})

		result, err := svc.Login(context.Background(), "disabled_user", "password123")
		if result != nil {
			t.Error("expected nil result for disabled user")
		}
		if err == nil {
			t.Fatal("expected error for disabled user, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeForbidden {
			t.Errorf("expected error code %d, got %d", bizerr.CodeForbidden, bizErr.Code)
		}
	})

	t.Run("successful login updates last login time even if update fails", func(t *testing.T) {
		repo := newMockUserRepo()
		repo.updateErr = errors.New("db connection error")
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           3,
			Username:     "user3",
			PasswordHash: mustHashPassword(t, "pass"),
			Role:         "dev",
			IsActive:     true,
		})

		// Login should still succeed even if Update fails (it just logs the error).
		result, err := svc.Login(context.Background(), "user3", "pass")
		if err != nil {
			t.Fatalf("Login should succeed even if Update fails: %v", err)
		}
		if result == nil || result.FullTokens == nil {
			t.Fatal("expected non-nil FullTokens")
		}
		if result.FullTokens.User.Username != "user3" {
			t.Errorf("expected username 'user3', got %q", result.FullTokens.User.Username)
		}
	})

	t.Run("user with 2FA enabled returns temp token", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:            4,
			Username:      "mfa_user",
			PasswordHash:  mustHashPassword(t, "pass"),
			Role:          "dev",
			IsActive:      true,
			TOTPEnabled:   true,
			TOTPSecretEnc: "dummyencryptedsecret",
		})

		result, err := svc.Login(context.Background(), "mfa_user", "pass")
		if err != nil {
			t.Fatalf("Login returned unexpected error: %v", err)
		}
		if result.FullTokens != nil {
			t.Error("expected FullTokens to be nil when 2FA is required")
		}
		if result.TwoFARequired == nil {
			t.Fatal("expected TwoFARequired to be set")
		}
		if result.TwoFARequired.TempToken == "" {
			t.Error("expected non-empty temp token")
		}
		// Verify the temp token is parseable.
		claims, err := jwtMgr.ParseTempToken(result.TwoFARequired.TempToken)
		if err != nil {
			t.Fatalf("failed to parse temp token: %v", err)
		}
		if claims.UserID != 4 {
			t.Errorf("temp token UserID = %d, want 4", claims.UserID)
		}
		if !claims.Pending2FA {
			t.Error("temp token should have Pending2FA=true")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: RefreshToken
// ---------------------------------------------------------------------------

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("valid refresh token returns new token pair", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           1,
			Username:     "admin",
			PasswordHash: mustHashPassword(t, "password"),
			Role:         "admin",
			IsActive:     true,
			TokenVersion: 0,
		})

		// Generate a valid refresh token.
		refreshToken, err := jwtMgr.GenerateRefreshToken(1, 0)
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}

		resp, err := svc.RefreshToken(context.Background(), refreshToken)
		if err != nil {
			t.Fatalf("RefreshToken returned unexpected error: %v", err)
		}

		if resp.AccessToken == "" {
			t.Error("expected non-empty access token")
		}
		if resp.RefreshToken == "" {
			t.Error("expected non-empty refresh token")
		}
		if resp.User.ID != 1 {
			t.Errorf("expected user ID 1, got %d", resp.User.ID)
		}
		if resp.User.Username != "admin" {
			t.Errorf("expected username 'admin', got %q", resp.User.Username)
		}
		if resp.User.Role != "admin" {
			t.Errorf("expected role 'admin', got %q", resp.User.Role)
		}

		// The new access token should be parseable.
		claims, err := jwtMgr.ParseToken(resp.AccessToken)
		if err != nil {
			t.Fatalf("failed to parse new access token: %v", err)
		}
		if claims.UserID != 1 {
			t.Errorf("new token UserID = %d, want 1", claims.UserID)
		}
	})

	t.Run("invalid refresh token returns token expired error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		resp, err := svc.RefreshToken(context.Background(), "invalid-token-string")
		if resp != nil {
			t.Error("expected nil response for invalid token")
		}
		if err == nil {
			t.Fatal("expected error for invalid token, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeTokenExpired {
			t.Errorf("expected error code %d, got %d", bizerr.CodeTokenExpired, bizErr.Code)
		}
	})

	t.Run("refresh token for non-existent user returns unauthorized error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		// Generate a token for user ID 999 which does not exist in the repo.
		refreshToken, err := jwtMgr.GenerateRefreshToken(999, 0)
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}

		resp, err := svc.RefreshToken(context.Background(), refreshToken)
		if resp != nil {
			t.Error("expected nil response for non-existent user")
		}
		if err == nil {
			t.Fatal("expected error for non-existent user, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeUnauthorized {
			t.Errorf("expected error code %d, got %d", bizerr.CodeUnauthorized, bizErr.Code)
		}
	})

	t.Run("refresh token for disabled user returns forbidden error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		repo.addUser(&model.User{
			ID:           5,
			Username:     "disabled",
			PasswordHash: mustHashPassword(t, "pw"),
			Role:         "dev",
			IsActive:     false,
		})

		refreshToken, err := jwtMgr.GenerateRefreshToken(5, 0)
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}

		resp, err := svc.RefreshToken(context.Background(), refreshToken)
		if resp != nil {
			t.Error("expected nil response for disabled user")
		}
		if err == nil {
			t.Fatal("expected error for disabled user, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeForbidden {
			t.Errorf("expected error code %d, got %d", bizerr.CodeForbidden, bizErr.Code)
		}
	})

	t.Run("refresh token signed with different secret returns error", func(t *testing.T) {
		repo := newMockUserRepo()
		jwtMgr := newTestJWTManager()
		svc := NewAuthService(repo, jwtMgr, newTestConfig(), newTestLogger())

		// Generate token with a different secret.
		otherJWT := auth.NewJWTManager("other-secret", 15*time.Minute, 7*24*time.Hour)
		refreshToken, err := otherJWT.GenerateRefreshToken(1, 0)
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}

		resp, err := svc.RefreshToken(context.Background(), refreshToken)
		if resp != nil {
			t.Error("expected nil response for token signed with wrong secret")
		}
		if err == nil {
			t.Fatal("expected error for token signed with wrong secret, got nil")
		}

		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeTokenExpired {
			t.Errorf("expected error code %d, got %d", bizerr.CodeTokenExpired, bizErr.Code)
		}
	})
}
