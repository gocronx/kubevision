package auth

import (
	"testing"
	"time"
)

const (
	testSecret     = "test-secret-key-for-jwt-unit-tests"
	testAccessTTL  = 5 * time.Minute
	testRefreshTTL = 24 * time.Hour
)

func newTestJWTManager() *JWTManager {
	return NewJWTManager(testSecret, testAccessTTL, testRefreshTTL)
}

// ---------------------------------------------------------------------------
// Access token tests
// ---------------------------------------------------------------------------

func TestGenerateAndParseAccessToken(t *testing.T) {
	t.Run("roundtrip: generate then parse returns original claims", func(t *testing.T) {
		mgr := newTestJWTManager()
		input := &TokenClaims{
			UserID:       42,
			Username:     "alice",
			Role:         "admin",
			TokenVersion: 3,
			ClusterRoles: map[string]string{"cluster-a": "viewer"},
		}

		tokenStr, err := mgr.GenerateAccessToken(input)
		if err != nil {
			t.Fatalf("GenerateAccessToken error: %v", err)
		}
		if tokenStr == "" {
			t.Fatal("GenerateAccessToken returned empty string")
		}

		parsed, err := mgr.ParseToken(tokenStr)
		if err != nil {
			t.Fatalf("ParseToken error: %v", err)
		}

		if parsed.UserID != 42 {
			t.Errorf("UserID = %d, want 42", parsed.UserID)
		}
		if parsed.Username != "alice" {
			t.Errorf("Username = %q, want %q", parsed.Username, "alice")
		}
		if parsed.Role != "admin" {
			t.Errorf("Role = %q, want %q", parsed.Role, "admin")
		}
		if parsed.TokenVersion != 3 {
			t.Errorf("TokenVersion = %d, want 3", parsed.TokenVersion)
		}
		if got, ok := parsed.ClusterRoles["cluster-a"]; !ok || got != "viewer" {
			t.Errorf("ClusterRoles[\"cluster-a\"] = %q (ok=%v), want %q", got, ok, "viewer")
		}
		if parsed.Issuer != "kubevision" {
			t.Errorf("Issuer = %q, want %q", parsed.Issuer, "kubevision")
		}
		if parsed.Subject != "42" {
			t.Errorf("Subject = %q, want %q", parsed.Subject, "42")
		}
	})

	t.Run("claims contain correct UserID, Username, Role, TokenVersion", func(t *testing.T) {
		tests := []struct {
			name         string
			userID       uint
			username     string
			role         string
			tokenVersion int
		}{
			{"basic user", 1, "bob", "viewer", 0},
			{"admin user", 999, "root", "superadmin", 7},
			{"zero id", 0, "ghost", "none", 1},
		}

		mgr := newTestJWTManager()

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				input := &TokenClaims{
					UserID:       tc.userID,
					Username:     tc.username,
					Role:         tc.role,
					TokenVersion: tc.tokenVersion,
				}
				tokenStr, err := mgr.GenerateAccessToken(input)
				if err != nil {
					t.Fatalf("GenerateAccessToken error: %v", err)
				}

				parsed, err := mgr.ParseToken(tokenStr)
				if err != nil {
					t.Fatalf("ParseToken error: %v", err)
				}

				if parsed.UserID != tc.userID {
					t.Errorf("UserID = %d, want %d", parsed.UserID, tc.userID)
				}
				if parsed.Username != tc.username {
					t.Errorf("Username = %q, want %q", parsed.Username, tc.username)
				}
				if parsed.Role != tc.role {
					t.Errorf("Role = %q, want %q", parsed.Role, tc.role)
				}
				if parsed.TokenVersion != tc.tokenVersion {
					t.Errorf("TokenVersion = %d, want %d", parsed.TokenVersion, tc.tokenVersion)
				}
			})
		}
	})
}

func TestParseTokenFailures(t *testing.T) {
	mgr := newTestJWTManager()

	t.Run("invalid token string", func(t *testing.T) {
		invalidTokens := []struct {
			name  string
			token string
		}{
			{"empty string", ""},
			{"random garbage", "not.a.jwt"},
			{"truncated", "eyJhbGciOiJIUzI1NiJ9.eyJ1aWQiOjF9"},
			{"completely wrong", "xxxxx.yyyyy.zzzzz"},
		}

		for _, tc := range invalidTokens {
			t.Run(tc.name, func(t *testing.T) {
				_, err := mgr.ParseToken(tc.token)
				if err == nil {
					t.Errorf("ParseToken(%q) expected error, got nil", tc.token)
				}
			})
		}
	})

	t.Run("expired token", func(t *testing.T) {
		shortMgr := NewJWTManager(testSecret, 1*time.Millisecond, testRefreshTTL)
		input := &TokenClaims{
			UserID:   1,
			Username: "expiring",
			Role:     "user",
		}
		tokenStr, err := shortMgr.GenerateAccessToken(input)
		if err != nil {
			t.Fatalf("GenerateAccessToken error: %v", err)
		}

		// Wait for the token to expire.
		time.Sleep(50 * time.Millisecond)

		_, err = shortMgr.ParseToken(tokenStr)
		if err == nil {
			t.Error("ParseToken on expired token expected error, got nil")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		otherMgr := NewJWTManager("different-secret", testAccessTTL, testRefreshTTL)
		input := &TokenClaims{
			UserID:   1,
			Username: "alice",
			Role:     "admin",
		}
		tokenStr, err := mgr.GenerateAccessToken(input)
		if err != nil {
			t.Fatalf("GenerateAccessToken error: %v", err)
		}

		_, err = otherMgr.ParseToken(tokenStr)
		if err == nil {
			t.Error("ParseToken with wrong secret expected error, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Refresh token tests
// ---------------------------------------------------------------------------

func TestGenerateAndParseRefreshToken(t *testing.T) {
	t.Run("roundtrip: generate then parse returns original userID", func(t *testing.T) {
		mgr := newTestJWTManager()

		tokenStr, err := mgr.GenerateRefreshToken(42, 3)
		if err != nil {
			t.Fatalf("GenerateRefreshToken error: %v", err)
		}
		if tokenStr == "" {
			t.Fatal("GenerateRefreshToken returned empty string")
		}

		claims, err := mgr.ParseRefreshToken(tokenStr)
		if err != nil {
			t.Fatalf("ParseRefreshToken error: %v", err)
		}
		if claims.UserID != 42 {
			t.Errorf("ParseRefreshToken returned userID = %d, want 42", claims.UserID)
		}
		if claims.TokenVersion != 3 {
			t.Errorf("ParseRefreshToken returned tokenVersion = %d, want 3", claims.TokenVersion)
		}
	})

	t.Run("various user IDs", func(t *testing.T) {
		mgr := newTestJWTManager()
		userIDs := []uint{0, 1, 100, 999999}

		for _, id := range userIDs {
			tokenStr, err := mgr.GenerateRefreshToken(id, 0)
			if err != nil {
				t.Fatalf("GenerateRefreshToken(%d) error: %v", id, err)
			}
			got, err := mgr.ParseRefreshToken(tokenStr)
			if err != nil {
				t.Fatalf("ParseRefreshToken for userID %d error: %v", id, err)
			}
			if got.UserID != id {
				t.Errorf("ParseRefreshToken returned %d, want %d", got.UserID, id)
			}
		}
	})
}

func TestParseRefreshTokenFailures(t *testing.T) {
	mgr := newTestJWTManager()

	t.Run("invalid token string", func(t *testing.T) {
		invalidTokens := []struct {
			name  string
			token string
		}{
			{"empty string", ""},
			{"random garbage", "not.a.jwt"},
			{"completely wrong", "xxxxx.yyyyy.zzzzz"},
		}

		for _, tc := range invalidTokens {
			t.Run(tc.name, func(t *testing.T) {
				_, err := mgr.ParseRefreshToken(tc.token)
				if err == nil {
					t.Errorf("ParseRefreshToken(%q) expected error, got nil", tc.token)
				}
			})
		}
	})

	t.Run("expired refresh token", func(t *testing.T) {
		shortMgr := NewJWTManager(testSecret, testAccessTTL, 1*time.Millisecond)
		tokenStr, err := shortMgr.GenerateRefreshToken(1, 0)
		if err != nil {
			t.Fatalf("GenerateRefreshToken error: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		_, err = shortMgr.ParseRefreshToken(tokenStr)
		if err == nil {
			t.Error("ParseRefreshToken on expired token expected error, got nil")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		otherMgr := NewJWTManager("different-secret", testAccessTTL, testRefreshTTL)
		tokenStr, err := mgr.GenerateRefreshToken(1, 0)
		if err != nil {
			t.Fatalf("GenerateRefreshToken error: %v", err)
		}

		_, err = otherMgr.ParseRefreshToken(tokenStr)
		if err == nil {
			t.Error("ParseRefreshToken with wrong secret expected error, got nil")
		}
	})
}
