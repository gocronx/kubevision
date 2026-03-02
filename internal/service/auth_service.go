package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/config"
	"github.com/kubevision/kubevision/internal/model"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// LoginResponse holds the data returned after a successful login or token refresh.
type LoginResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	User         UserInfo `json:"user"`
}

// Login2FARequiredResponse is returned when a user has 2FA enabled.
// The frontend must exchange TempToken for real JWT tokens via POST /auth/2fa/verify.
type Login2FARequiredResponse struct {
	TempToken string `json:"tempToken"`
}

// Setup2FAResponse holds the TOTP setup information shown to the user.
type Setup2FAResponse struct {
	Secret        string   `json:"secret"`
	OtpauthURL    string   `json:"otpauthUrl"`
	RecoveryCodes []string `json:"recoveryCodes"`
}

// UserInfo is a safe subset of user data included in login responses.
type UserInfo struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	TOTPEnabled bool   `json:"totpEnabled"`
}

// LoginResult is a discriminated union returned from Login.
// Exactly one of FullTokens or TwoFARequired will be non-nil.
type LoginResult struct {
	// FullTokens is set when authentication is complete.
	FullTokens *LoginResponse
	// TwoFARequired is set when a second factor is needed.
	TwoFARequired *Login2FARequiredResponse
}

// AuthService encapsulates business logic for authentication and authorization.
type AuthService struct {
	userRepo   repository.UserRepo
	jwtManager *auth.JWTManager
	cfg        *config.Config
	logger     *zap.Logger
}

// NewAuthService creates a new AuthService with the given dependencies.
func NewAuthService(userRepo repository.UserRepo, jwtManager *auth.JWTManager, cfg *config.Config, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		cfg:        cfg,
		logger:     logger,
	}
}

// Login authenticates a user by username and password.
// If the user has 2FA enabled, LoginResult.TwoFARequired is populated and
// FullTokens is nil — the handler must respond with code 40102.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
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

	// Update last login time.
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update last login time", zap.String("username", user.Username), zap.Error(err))
	}

	// If 2FA is enabled, return a short-lived temp token for the verification step.
	if user.TOTPEnabled {
		tempToken, err := s.jwtManager.GenerateTempToken(user.ID)
		if err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to generate temp token")
		}
		return &LoginResult{
			TwoFARequired: &Login2FARequiredResponse{TempToken: tempToken},
		}, nil
	}

	tokens, err := s.buildLoginResponse(user)
	if err != nil {
		return nil, err
	}
	return &LoginResult{FullTokens: tokens}, nil
}

// RefreshToken validates a refresh token and returns a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Parse the refresh token to extract user ID and token version.
	rtClaims, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeTokenExpired, "invalid or expired refresh token")
	}

	// Look up the user to ensure they still exist and are active.
	user, err := s.userRepo.GetByID(ctx, rtClaims.UserID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "user not found")
	}

	if !user.IsActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
	}

	// Verify the token version matches the stored version.
	// A mismatch means the token was revoked (e.g. via logout-all or password change).
	if rtClaims.TokenVersion != user.TokenVersion {
		return nil, bizerr.New(bizerr.CodeTokenExpired, "token has been revoked")
	}

	return s.buildLoginResponse(user)
}

// Setup2FA generates a new TOTP secret for the user and returns the QR code URL
// and a fresh set of recovery codes. The secret is NOT activated yet; call
// Enable2FA after the user verifies the code to activate it.
func (s *AuthService) Setup2FA(ctx context.Context, userID uint) (*Setup2FAResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	if user.TOTPEnabled {
		return nil, bizerr.New(bizerr.CodeConflict, "2FA is already enabled; disable it first")
	}

	// Generate a fresh TOTP secret.
	secret, otpauthURL, err := auth.GenerateSecret(user.Username)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate TOTP secret")
	}

	// Generate recovery codes.
	recoveryCodes, err := auth.GenerateRecoveryCodes(8)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate recovery codes")
	}

	// Encrypt and persist the pending secret (TOTPEnabled stays false).
	encSecret, err := auth.EncryptSecret(secret, s.cfg.EncryptKey)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to encrypt TOTP secret")
	}

	// Encode recovery codes as JSON and encrypt.
	codesJSON, _ := json.Marshal(recoveryCodes)
	encCodes, err := auth.EncryptSecret(string(codesJSON), s.cfg.EncryptKey)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to encrypt recovery codes")
	}

	user.TOTPSecretEnc = encSecret
	user.RecoveryCodesEnc = encCodes
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to save TOTP setup")
	}

	return &Setup2FAResponse{
		Secret:        secret,
		OtpauthURL:    otpauthURL,
		RecoveryCodes: recoveryCodes,
	}, nil
}

// Enable2FA activates 2FA for the user after verifying the provided TOTP code.
func (s *AuthService) Enable2FA(ctx context.Context, userID uint, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	if user.TOTPEnabled {
		return bizerr.New(bizerr.CodeConflict, "2FA is already enabled")
	}

	if user.TOTPSecretEnc == "" {
		return bizerr.New(bizerr.CodeParamInvalid, "run setup first before enabling 2FA")
	}

	// Decrypt the pending secret.
	secret, err := auth.DecryptSecret(user.TOTPSecretEnc, s.cfg.EncryptKey)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to decrypt TOTP secret")
	}

	// Validate the code.
	if !auth.ValidateCodeWithOptions(secret, code) {
		return bizerr.New(bizerr.Code2FAFailed, "invalid TOTP code")
	}

	// Activate 2FA.
	user.TOTPEnabled = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to enable 2FA")
	}

	return nil
}

// Disable2FA deactivates 2FA for the user after verifying the provided TOTP code.
func (s *AuthService) Disable2FA(ctx context.Context, userID uint, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}

	if !user.TOTPEnabled {
		return bizerr.New(bizerr.CodeConflict, "2FA is not enabled")
	}

	// Decrypt stored secret.
	secret, err := auth.DecryptSecret(user.TOTPSecretEnc, s.cfg.EncryptKey)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to decrypt TOTP secret")
	}

	// Validate the code.
	if !auth.ValidateCodeWithOptions(secret, code) {
		return bizerr.New(bizerr.Code2FAFailed, "invalid TOTP code")
	}

	// Clear all TOTP fields.
	user.TOTPEnabled = false
	user.TOTPSecretEnc = ""
	user.RecoveryCodesEnc = ""
	if err := s.userRepo.Update(ctx, user); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to disable 2FA")
	}

	return nil
}

// Verify2FA validates a TOTP code presented with the pending temp token.
// On success it returns full JWT tokens.
func (s *AuthService) Verify2FA(ctx context.Context, tempToken, code string) (*LoginResponse, error) {
	claims, err := s.jwtManager.ParseTempToken(tempToken)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid or expired temp token")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "user not found")
	}

	if !user.IsActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
	}

	if !user.TOTPEnabled || user.TOTPSecretEnc == "" {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "2FA is not enabled for this user")
	}

	secret, err := auth.DecryptSecret(user.TOTPSecretEnc, s.cfg.EncryptKey)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to decrypt TOTP secret")
	}

	if !auth.ValidateCodeWithOptions(secret, code) {
		return nil, bizerr.New(bizerr.Code2FAFailed, "invalid TOTP code")
	}

	return s.buildLoginResponse(user)
}

// UseRecoveryCode validates a recovery code and exchanges it for JWT tokens.
// The used code is removed from the stored list to prevent reuse.
func (s *AuthService) UseRecoveryCode(ctx context.Context, tempToken, recoveryCode string) (*LoginResponse, error) {
	claims, err := s.jwtManager.ParseTempToken(tempToken)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid or expired temp token")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "user not found")
	}

	if !user.IsActive {
		return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
	}

	if user.RecoveryCodesEnc == "" {
		return nil, bizerr.New(bizerr.Code2FAFailed, "no recovery codes available")
	}

	// Decrypt stored codes.
	codesJSON, err := auth.DecryptSecret(user.RecoveryCodesEnc, s.cfg.EncryptKey)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to decrypt recovery codes")
	}

	var codes []string
	if err := json.Unmarshal([]byte(codesJSON), &codes); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to parse recovery codes")
	}

	// Normalise the provided code (upper-case, trim whitespace).
	provided := strings.ToUpper(strings.TrimSpace(recoveryCode))

	found := false
	remaining := make([]string, 0, len(codes))
	for _, c := range codes {
		if subtle.ConstantTimeCompare([]byte(c), []byte(provided)) == 1 {
			found = true
			// Consume this code — do not add to remaining.
		} else {
			remaining = append(remaining, c)
		}
	}

	if !found {
		return nil, bizerr.New(bizerr.Code2FAFailed, "invalid recovery code")
	}

	// Persist the updated (reduced) recovery code list.
	if len(remaining) == 0 {
		user.RecoveryCodesEnc = ""
	} else {
		newCodesJSON, _ := json.Marshal(remaining)
		encCodes, err := auth.EncryptSecret(string(newCodesJSON), s.cfg.EncryptKey)
		if err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to encrypt recovery codes")
		}
		user.RecoveryCodesEnc = encCodes
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to update recovery codes")
	}

	return s.buildLoginResponse(user)
}

// buildLoginResponse generates a full access+refresh token pair for the given user.
func (s *AuthService) buildLoginResponse(user *model.User) (*LoginResponse, error) {
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

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, user.TokenVersion)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate refresh token")
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Role:        user.Role,
			TOTPEnabled: user.TOTPEnabled,
		},
	}, nil
}
