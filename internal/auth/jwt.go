package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims holds the JWT claims for access tokens.
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID       uint              `json:"uid"`
	Username     string            `json:"username"`
	Role         string            `json:"role"`
	ClusterRoles map[string]string `json:"clusterRoles,omitempty"`
	TokenVersion int               `json:"tv"`
}

// refreshClaims holds the JWT claims for refresh tokens.
type refreshClaims struct {
	jwt.RegisteredClaims
	UserID       uint `json:"uid"`
	TokenVersion int  `json:"tv"`
}

// TempTokenClaims holds the JWT claims for short-lived 2FA pending tokens.
// These are issued after successful password auth when 2FA is required,
// and are used exclusively to authorize the 2FA verification step.
type TempTokenClaims struct {
	jwt.RegisteredClaims
	UserID     uint `json:"uid"`
	Pending2FA bool `json:"pending2fa"`
}

// JWTManager handles JWT token generation and validation.
type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTManager creates a new JWTManager with the given secret and TTL values.
func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateAccessToken creates a signed JWT access token with the given claims.
func (m *JWTManager) GenerateAccessToken(claims *TokenClaims) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "kubevision",
		Subject:   fmt.Sprintf("%d", claims.UserID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// GenerateRefreshToken creates a signed JWT refresh token for the given user.
// tokenVersion must match the user's current TokenVersion so that refresh
// tokens are invalidated whenever the version is bumped (e.g. on logout-all).
func (m *JWTManager) GenerateRefreshToken(userID uint, tokenVersion int) (string, error) {
	now := time.Now()
	claims := &refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "kubevision",
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
		},
		UserID:       userID,
		TokenVersion: tokenVersion,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken parses and validates an access token string and returns its claims.
func (m *JWTManager) ParseToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GenerateTempToken creates a short-lived JWT used during the 2FA verification step.
// The token is valid for 5 minutes and carries the pending2fa flag.
func (m *JWTManager) GenerateTempToken(userID uint) (string, error) {
	now := time.Now()
	claims := &TempTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "kubevision",
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		UserID:     userID,
		Pending2FA: true,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseTempToken parses and validates a 2FA pending token.
// It returns an error if the token is not a valid pending2fa token.
func (m *JWTManager) ParseTempToken(tokenStr string) (*TempTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TempTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse temp token: %w", err)
	}

	claims, ok := token.Claims.(*TempTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid temp token claims")
	}
	if !claims.Pending2FA {
		return nil, fmt.Errorf("token is not a 2FA pending token")
	}
	return claims, nil
}

// RefreshTokenClaims is the parsed payload of a refresh token.
type RefreshTokenClaims struct {
	UserID       uint
	TokenVersion int
}

// ParseRefreshToken parses and validates a refresh token and returns its claims.
func (m *JWTManager) ParseRefreshToken(tokenStr string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &refreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*refreshClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	return &RefreshTokenClaims{
		UserID:       claims.UserID,
		TokenVersion: claims.TokenVersion,
	}, nil
}
