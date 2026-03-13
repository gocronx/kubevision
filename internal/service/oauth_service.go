package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// OAuthProviderInfo is the public information about an available OAuth provider.
type OAuthProviderInfo struct {
	Name    string `json:"name"`
	AuthURL string `json:"authUrl"`
}

// OAuthService handles OAuth2/OIDC authentication flows.
type OAuthService struct {
	userRepo   repository.UserRepo
	jwtManager *auth.JWTManager
	cfg        *config.Config
	logger     *zap.Logger
	httpClient *http.Client

	// In-memory state store for CSRF protection.
	mu     sync.Mutex
	states map[string]stateEntry
}

type stateEntry struct {
	provider  string
	expiresAt time.Time
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	userRepo repository.UserRepo,
	jwtManager *auth.JWTManager,
	cfg *config.Config,
	logger *zap.Logger,
) *OAuthService {
	svc := &OAuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		states:     make(map[string]stateEntry),
	}
	return svc
}

// ListProviders returns the enabled OAuth providers.
func (s *OAuthService) ListProviders() []OAuthProviderInfo {
	if !s.cfg.OAuth.Enabled {
		return nil
	}
	var providers []OAuthProviderInfo
	for _, p := range s.cfg.OAuth.Providers {
		oauthCfg := s.oauthConfig(p)
		if oauthCfg != nil {
			providers = append(providers, OAuthProviderInfo{
				Name:    p.Name,
				AuthURL: oauthCfg.Endpoint.AuthURL,
			})
		}
	}
	return providers
}

// GetAuthorizationURL generates an OAuth authorization URL with a state parameter.
func (s *OAuthService) GetAuthorizationURL(providerName string) (string, error) {
	p, oauthCfg := s.findProvider(providerName)
	if p == nil {
		return "", bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("OAuth provider %q not found", providerName))
	}

	state, err := randomState()
	if err != nil {
		return "", bizerr.New(bizerr.CodeInternal, "failed to generate state")
	}

	s.mu.Lock()
	s.states[state] = stateEntry{provider: providerName, expiresAt: time.Now().Add(5 * time.Minute)}
	s.mu.Unlock()

	return oauthCfg.AuthCodeURL(state), nil
}

// HandleCallback processes the OAuth callback: exchanges code for token,
// fetches user info, creates/links user, and returns JWT tokens.
func (s *OAuthService) HandleCallback(ctx context.Context, providerName, code, state string) (*LoginResult, error) {
	// Validate state
	s.mu.Lock()
	entry, ok := s.states[state]
	if ok {
		delete(s.states, state)
	}
	s.mu.Unlock()

	if !ok || entry.expiresAt.Before(time.Now()) || entry.provider != providerName {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid or expired OAuth state")
	}

	p, oauthCfg := s.findProvider(providerName)
	if p == nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("provider %q not found", providerName))
	}

	// Exchange code for token
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, fmt.Sprintf("token exchange failed: %v", err))
	}

	// Fetch user info
	userInfo, err := s.fetchUserInfo(ctx, providerName, p, token)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to fetch user info: %v", err))
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, providerName, userInfo)
	if err != nil {
		return nil, err
	}

	// Generate JWT tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(&auth.TokenClaims{
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
	})
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate access token")
	}
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, user.TokenVersion)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to generate refresh token")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.Update(ctx, user)

	return &LoginResult{
		FullTokens: &LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			User: UserInfo{
				ID:          user.ID,
				Username:    user.Username,
				Role:        user.Role,
				TOTPEnabled: user.TOTPEnabled,
			},
		},
	}, nil
}

// CleanupExpiredStates removes expired state entries.
func (s *OAuthService) CleanupExpiredStates() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.states {
		if v.expiresAt.Before(now) {
			delete(s.states, k)
		}
	}
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

type oauthUserInfo struct {
	ID       string
	Username string
	Email    string
}

func (s *OAuthService) findProvider(name string) (*config.OAuthProvider, *oauth2.Config) {
	for _, p := range s.cfg.OAuth.Providers {
		if p.Name == name {
			oauthCfg := s.oauthConfig(p)
			return &p, oauthCfg
		}
	}
	return nil, nil
}

func (s *OAuthService) oauthConfig(p config.OAuthProvider) *oauth2.Config {
	if p.ClientID == "" || p.ClientSecret == "" {
		return nil
	}

	scopes := p.Scopes
	if len(scopes) == 0 {
		switch p.Name {
		case "github":
			scopes = []string{"read:user", "user:email"}
		case "google":
			scopes = []string{"openid", "email", "profile"}
		default:
			scopes = []string{"openid", "email", "profile"}
		}
	}

	endpoint := oauth2.Endpoint{
		AuthURL:  p.AuthURL,
		TokenURL: p.TokenURL,
	}

	// Well-known endpoints for built-in providers.
	switch p.Name {
	case "github":
		if endpoint.AuthURL == "" {
			endpoint.AuthURL = "https://github.com/login/oauth/authorize"
		}
		if endpoint.TokenURL == "" {
			endpoint.TokenURL = "https://github.com/login/oauth/access_token"
		}
	case "google":
		if endpoint.AuthURL == "" {
			endpoint.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
		}
		if endpoint.TokenURL == "" {
			endpoint.TokenURL = "https://oauth2.googleapis.com/token"
		}
	}

	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       scopes,
		RedirectURL:  p.RedirectURL,
	}
}

func (s *OAuthService) fetchUserInfo(ctx context.Context, providerName string, p *config.OAuthProvider, token *oauth2.Token) (*oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	var url string
	switch providerName {
	case "github":
		url = "https://api.github.com/user"
	case "google":
		url = "https://www.googleapis.com/oauth2/v2/userinfo"
	default:
		url = p.UserInfoURL
	}

	if url == "" {
		return nil, fmt.Errorf("userinfo URL not configured for provider %q", providerName)
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	info := &oauthUserInfo{}
	switch providerName {
	case "github":
		if v, ok := raw["id"]; ok {
			info.ID = fmt.Sprintf("%v", v)
		}
		if v, ok := raw["login"].(string); ok {
			info.Username = v
		}
		if v, ok := raw["email"].(string); ok {
			info.Email = v
		}
	case "google":
		if v, ok := raw["id"].(string); ok {
			info.ID = v
		}
		if v, ok := raw["name"].(string); ok {
			info.Username = v
		}
		if v, ok := raw["email"].(string); ok {
			info.Email = v
		}
	default:
		// Generic OIDC: try common claims.
		if v, ok := raw["sub"].(string); ok {
			info.ID = v
		}
		if v, ok := raw["preferred_username"].(string); ok {
			info.Username = v
		} else if v, ok := raw["name"].(string); ok {
			info.Username = v
		}
		if v, ok := raw["email"].(string); ok {
			info.Email = v
		}
	}

	if info.ID == "" {
		return nil, fmt.Errorf("could not extract user ID from %s response", providerName)
	}
	return info, nil
}

func (s *OAuthService) findOrCreateUser(ctx context.Context, providerName string, info *oauthUserInfo) (*model.User, error) {
	// Try to find by OAuth ID first.
	user, err := s.userRepo.GetByOAuthID(ctx, providerName, info.ID)
	if err == nil {
		return user, nil
	}

	// Try to find by email and link.
	if info.Email != "" {
		user, err = s.userRepo.GetByEmail(ctx, info.Email)
		if err == nil {
			user.OAuthID = info.ID
			user.AuthProvider = providerName
			if err := s.userRepo.Update(ctx, user); err != nil {
				return nil, bizerr.New(bizerr.CodeInternal, "failed to link OAuth account")
			}
			return user, nil
		}
	}

	// Create new user.
	username := info.Username
	if username == "" {
		username = fmt.Sprintf("%s_%s", providerName, info.ID)
	}
	// Ensure unique username.
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		username = fmt.Sprintf("%s_%s_%s", providerName, info.ID, info.Username)
	}

	newUser := &model.User{
		Username:     username,
		Email:        info.Email,
		OAuthID:      info.ID,
		PasswordHash: "-", // OAuth users don't have passwords.
		Role:         "viewer",
		AuthProvider: providerName,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to create OAuth user")
	}
	return newUser, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
