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
	TokenType    string            `json:"typ"`
}

// refreshClaims holds the JWT claims for refresh tokens.
type refreshClaims struct {
	jwt.RegisteredClaims
	UserID       uint   `json:"uid"`
	TokenVersion int    `json:"tv"`
	TokenType    string `json:"typ"`
}

// TempTokenClaims holds the JWT claims for short-lived 2FA pending tokens.
// These are issued after successful password auth when 2FA is required,
// and are used exclusively to authorize the 2FA verification step.
type TempTokenClaims struct {
	jwt.RegisteredClaims
	UserID     uint   `json:"uid"`
	Pending2FA bool   `json:"pending2fa"`
	TokenType  string `json:"typ"`
}

// WebSocketTicketClaims holds the identity carried by a short-lived ticket.
// Tickets are separate from access tokens so URL exposure cannot disclose a
// reusable application session.
type WebSocketTicketClaims struct {
	jwt.RegisteredClaims
	UserID    uint   `json:"uid"`
	TokenType string `json:"typ"`
}

const webSocketTicketTTL = 30 * time.Second

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
	claims.TokenType = "access"
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
		TokenType:    "refresh",
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
	if !ok || !token.Valid || claims.TokenType != "" && claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GenerateWebSocketTicket creates a narrowly scoped ticket for WebSocket
// upgrades. The caller must already be authenticated.
func (m *JWTManager) GenerateWebSocketTicket(userID uint) (string, error) {
	now := time.Now()
	claims := &WebSocketTicketClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "kubevision",
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  jwt.ClaimStrings{"kubevision-websocket"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(webSocketTicketTTL)),
		},
		UserID:    userID,
		TokenType: "websocket",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ParseWebSocketTicket validates a short-lived WebSocket ticket. Access and
// refresh tokens are rejected because they lack the required audience.
func (m *JWTManager) ParseWebSocketTicket(ticket string) (*WebSocketTicketClaims, error) {
	token, err := jwt.ParseWithClaims(
		ticket,
		&WebSocketTicketClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithIssuer("kubevision"),
		jwt.WithAudience("kubevision-websocket"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse websocket ticket: %w", err)
	}
	claims, ok := token.Claims.(*WebSocketTicketClaims)
	if !ok || !token.Valid || claims.UserID == 0 || claims.TokenType != "websocket" {
		return nil, fmt.Errorf("invalid websocket ticket claims")
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
		TokenType:  "2fa",
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
	if !claims.Pending2FA || (claims.TokenType != "" && claims.TokenType != "2fa") {
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
	if !ok || !token.Valid || claims.TokenType != "" && claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	return &RefreshTokenClaims{
		UserID:       claims.UserID,
		TokenVersion: claims.TokenVersion,
	}, nil
}
