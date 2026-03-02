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
	UserID uint `json:"uid"`
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
func (m *JWTManager) GenerateRefreshToken(userID uint) (string, error) {
	now := time.Now()
	claims := &refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "kubevision",
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
		},
		UserID: userID,
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

// ParseRefreshToken parses and validates a refresh token and returns the user ID.
func (m *JWTManager) ParseRefreshToken(tokenStr string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &refreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return 0, fmt.Errorf("parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*refreshClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid refresh token claims")
	}

	return claims.UserID, nil
}
