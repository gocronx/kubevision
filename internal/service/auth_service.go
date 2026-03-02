package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// LoginResponse holds the data returned after a successful login or token refresh.
type LoginResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	User         UserInfo `json:"user"`
}

// UserInfo is a safe subset of user data included in login responses.
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// AuthService encapsulates business logic for authentication and authorization.
type AuthService struct {
	userRepo   repository.UserRepo
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

// NewAuthService creates a new AuthService with the given dependencies.
func NewAuthService(userRepo repository.UserRepo, jwtManager *auth.JWTManager, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Login authenticates a user by username and password, and returns tokens.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	// Look up user by username.
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid username or password")
	}

	// Verify account is active.
	if !user.IsActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
	}

	// Verify password.
	if !auth.CheckPassword(password, user.PasswordHash) {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid username or password")
	}

	// Generate access token.
	claims := &auth.TokenClaims{
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
	}
	accessToken, err := s.jwtManager.GenerateAccessToken(claims)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate access token")
	}

	// Generate refresh token.
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate refresh token")
	}

	// Update last login time.
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update last login time", zap.String("username", user.Username), zap.Error(err))
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}

// RefreshToken validates a refresh token and returns a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Parse the refresh token to extract user ID.
	userID, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeTokenExpired, "invalid or expired refresh token")
	}

	// Look up the user to ensure they still exist and are active.
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "user not found")
	}

	if !user.IsActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
	}

	// Generate new access token.
	claims := &auth.TokenClaims{
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
	}
	newAccessToken, err := s.jwtManager.GenerateAccessToken(claims)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate access token")
	}

	// Generate new refresh token.
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate refresh token")
	}

	return &LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}
