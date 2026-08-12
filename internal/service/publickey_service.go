package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	wa "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

const (
	ceremonyRegistration = "registration"
	ceremonyLogin        = "login"
)

type PublicKeyService struct {
	repo       *repository.PublicKeyRepo
	users      repository.UserRepo
	auth       *AuthService
	engine     *wa.WebAuthn
	cfg        config.PublicKeyAuthConfig
	appCfg     *config.Config
	logger     *zap.Logger
	configured bool
}

type PublicKeyCredentialInfo struct {
	ID             uint       `json:"id"`
	Label          string     `json:"label"`
	Transports     []string   `json:"transports"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastUsedAt     *time.Time `json:"lastUsedAt"`
	BackupEligible bool       `json:"backupEligible"`
	BackupState    bool       `json:"backupState"`
}

type CeremonyOptions struct {
	CeremonyID string `json:"ceremonyId"`
	Options    any    `json:"options"`
}

type webAuthnUser struct {
	user        *model.User
	credentials []wa.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                   { return u.user.PublicKeyHandle }
func (u *webAuthnUser) WebAuthnName() string                 { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string          { return u.user.Username }
func (u *webAuthnUser) WebAuthnCredentials() []wa.Credential { return u.credentials }

func NewPublicKeyService(repo *repository.PublicKeyRepo, users repository.UserRepo, authService *AuthService, cfg *config.Config, logger *zap.Logger) (*PublicKeyService, error) {
	s := &PublicKeyService{repo: repo, users: users, auth: authService, cfg: cfg.Auth.PublicKey, appCfg: cfg, logger: logger}
	if !s.cfg.Enabled {
		return s, nil
	}
	if s.cfg.RPID == "" || s.cfg.RPDisplayName == "" || len(s.cfg.Origins) == 0 {
		return nil, fmt.Errorf("public key authentication requires rp_id, rp_display_name, and origins")
	}
	for _, rawOrigin := range s.cfg.Origins {
		origin, parseErr := url.Parse(rawOrigin)
		if parseErr != nil || (origin.Scheme != "https" && !(origin.Scheme == "http" && (origin.Hostname() == "localhost" || origin.Hostname() == "127.0.0.1"))) || origin.Hostname() == "" || origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
			return nil, fmt.Errorf("invalid public key origin %q", rawOrigin)
		}
		host := strings.ToLower(origin.Hostname())
		rpID := strings.ToLower(s.cfg.RPID)
		if host != rpID && !strings.HasSuffix(host, "."+rpID) {
			return nil, fmt.Errorf("public key origin host %q is not within RP ID %q", host, s.cfg.RPID)
		}
	}
	uv := protocol.UserVerificationRequirement(s.cfg.UserVerification)
	if uv != protocol.VerificationRequired && uv != protocol.VerificationPreferred && uv != protocol.VerificationDiscouraged {
		return nil, fmt.Errorf("invalid public key user_verification %q", s.cfg.UserVerification)
	}
	if s.cfg.CounterPolicy != "deny" && s.cfg.CounterPolicy != "warn" {
		return nil, fmt.Errorf("invalid public key counter_policy %q", s.cfg.CounterPolicy)
	}
	if s.cfg.ChallengeTTL < time.Minute || s.cfg.ChallengeTTL > 10*time.Minute {
		return nil, fmt.Errorf("public key challenge_ttl must be between 1m and 10m")
	}
	engine, err := wa.New(&wa.Config{
		RPID: s.cfg.RPID, RPDisplayName: s.cfg.RPDisplayName, RPOrigins: s.cfg.Origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{UserVerification: uv},
		AttestationPreference:  protocol.PreferNoAttestation,
		Timeouts: wa.TimeoutsConfig{
			Login:        wa.TimeoutConfig{Enforce: true, Timeout: s.cfg.ChallengeTTL, TimeoutUVD: s.cfg.ChallengeTTL},
			Registration: wa.TimeoutConfig{Enforce: true, Timeout: s.cfg.ChallengeTTL, TimeoutUVD: s.cfg.ChallengeTTL},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize WebAuthn: %w", err)
	}
	s.engine, s.configured = engine, true
	logger.Warn("public key RP configuration is persistent security state; changing RP ID or origins may invalidate credentials", zap.String("rp_id", s.cfg.RPID), zap.Strings("origins", s.cfg.Origins))
	return s, nil
}

func (s *PublicKeyService) Enabled() bool { return s.configured }

func (s *PublicKeyService) BeginRegistration(ctx context.Context, userID uint, label, password, totp string) (*CeremonyOptions, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 128 {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "credential label is required")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || !user.IsActive {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "reauthentication failed")
	}
	if !s.verifyReauthentication(user, password, totp) {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "reauthentication failed")
	}
	if len(user.PublicKeyHandle) == 0 {
		handle := make([]byte, 32)
		if _, err := rand.Read(handle); err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to initialize credential identity")
		}
		if err := s.repo.SetUserHandle(ctx, user.ID, handle); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to initialize credential identity")
		}
		user, err = s.users.GetByID(ctx, userID)
		if err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to load account")
		}
	}
	wu, err := s.loadUser(ctx, user)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to load credentials")
	}
	creation, session, err := s.engine.BeginRegistration(wu,
		wa.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		wa.WithExclusions(wa.Credentials(wu.credentials).CredentialDescriptors()))
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to begin registration")
	}
	return s.storeCeremony(ctx, user.ID, ceremonyRegistration, label, creation, session)
}

func (s *PublicKeyService) FinishRegistration(ctx context.Context, userID uint, ceremonyID string, request *http.Request) (*PublicKeyCredentialInfo, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	ceremony, session, err := s.consume(ctx, ceremonyID, ceremonyRegistration)
	if err != nil || ceremony.UserID != userID {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "registration ceremony is invalid or expired")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || !user.IsActive {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "registration failed")
	}
	wu, err := s.loadUser(ctx, user)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "registration failed")
	}
	credential, err := s.engine.FinishRegistration(wu, session, request)
	if err != nil {
		s.logger.Warn("public key registration rejected", zap.Uint("user_id", userID), zap.Error(err))
		return nil, bizerr.New(bizerr.CodeUnauthorized, "registration failed")
	}
	record := fromWebAuthnCredential(userID, ceremony.Label, credential)
	if err := s.repo.CreateCredential(ctx, record); err != nil {
		if isUniqueViolation(err) {
			return nil, bizerr.New(bizerr.CodeConflict, "credential is already registered")
		}
		return nil, bizerr.New(bizerr.CodeInternal, "failed to store credential")
	}
	return credentialInfo(record), nil
}

func (s *PublicKeyService) BeginLogin(ctx context.Context, username string) (*CeremonyOptions, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	var options any
	var session *wa.SessionData
	var userID uint
	if strings.TrimSpace(username) != "" {
		if user, err := s.users.GetByUsername(ctx, strings.TrimSpace(username)); err == nil && user.IsActive {
			if wu, loadErr := s.loadUser(ctx, user); loadErr == nil && len(wu.credentials) > 0 {
				assertion, sess, beginErr := s.engine.BeginLogin(wu)
				if beginErr == nil {
					options, session, userID = assertion, sess, user.ID
				}
			}
		}
	}
	if session == nil {
		assertion, sess, err := s.engine.BeginDiscoverableLogin()
		if err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "authentication unavailable")
		}
		options, session = assertion, sess
	}
	return s.storeCeremony(ctx, userID, ceremonyLogin, "", options, session)
}

func (s *PublicKeyService) FinishLogin(ctx context.Context, ceremonyID string, request *http.Request) (*LoginResponse, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	ceremony, session, err := s.consume(ctx, ceremonyID, ceremonyLogin)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
	}
	var wu *webAuthnUser
	var validated *wa.Credential
	if ceremony.UserID != 0 {
		user, lookupErr := s.users.GetByID(ctx, ceremony.UserID)
		if lookupErr != nil || !user.IsActive {
			return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
		}
		wu, err = s.loadUser(ctx, user)
		if err == nil {
			validated, err = s.engine.FinishLogin(wu, session, request)
		}
	} else {
		var returned wa.User
		returned, validated, err = s.engine.FinishPasskeyLogin(func(rawID, handle []byte) (wa.User, error) {
			user, lookupErr := s.repo.FindUserByHandle(ctx, handle)
			if lookupErr != nil || !user.IsActive {
				return nil, errors.New("credential not found")
			}
			candidate, loadErr := s.loadUser(ctx, user)
			if loadErr != nil {
				return nil, loadErr
			}
			owned := false
			for _, credential := range candidate.credentials {
				if string(credential.ID) == string(rawID) {
					owned = true
					break
				}
			}
			if !owned {
				return nil, errors.New("credential not found")
			}
			wu = candidate
			return candidate, nil
		}, session, request)
		_ = returned
	}
	if err != nil || wu == nil || validated == nil {
		s.logger.Warn("public key authentication rejected", zap.Error(err))
		return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
	}
	record, err := s.repo.FindActiveByCredentialID(ctx, validated.ID)
	if err != nil || record.UserID != wu.user.ID {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
	}
	if validated.Authenticator.CloneWarning {
		s.logger.Warn("public key signature counter rollback detected", zap.Uint("user_id", wu.user.ID), zap.Uint("credential_id", record.ID), zap.String("policy", s.cfg.CounterPolicy))
		if s.cfg.CounterPolicy == "deny" {
			return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
		}
	}
	now := time.Now()
	if err := s.repo.UpdateAfterLogin(ctx, record.ID, record.SignCount, validated.Authenticator.SignCount, validated.Flags.BackupState, now); err != nil {
		if errors.Is(err, repository.ErrCounterRace) {
			s.logger.Warn("concurrent public key counter update rejected", zap.Uint("credential_id", record.ID))
			return nil, bizerr.New(bizerr.CodeUnauthorized, "authentication failed")
		}
		return nil, bizerr.New(bizerr.CodeInternal, "authentication failed")
	}
	return s.auth.IssueLoginForUser(ctx, wu.user.ID)
}

func (s *PublicKeyService) List(ctx context.Context, userID uint) ([]PublicKeyCredentialInfo, error) {
	records, err := s.repo.ListActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]PublicKeyCredentialInfo, 0, len(records))
	for i := range records {
		items = append(items, *credentialInfo(&records[i]))
	}
	return items, nil
}
func (s *PublicKeyService) Rename(ctx context.Context, userID, id uint, label string) error {
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 128 {
		return bizerr.New(bizerr.CodeParamInvalid, "credential label is required")
	}
	if err := s.repo.Rename(ctx, userID, id, label); err != nil {
		return bizerr.New(bizerr.CodeNotFound, "credential not found")
	}
	return nil
}
func (s *PublicKeyService) Revoke(ctx context.Context, userID, id uint) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "user not found")
	}
	other := user.PasswordHash != "" || user.TOTPEnabled || user.RecoveryCodesEnc != "" || (user.AuthProvider != "" && user.AuthProvider != "local")
	if err := s.repo.Revoke(ctx, userID, id, other, time.Now()); errors.Is(err, repository.ErrLastAuthenticationMethod) {
		return bizerr.New(bizerr.CodeConflict, "add another authentication or recovery method before revoking this credential")
	} else if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "credential not found")
	}
	return nil
}

func (s *PublicKeyService) AdminRevoke(ctx context.Context, targetUserID, id uint) error {
	return s.Revoke(ctx, targetUserID, id)
}

func (s *PublicKeyService) requireEnabled() error {
	if !s.configured {
		return bizerr.New(bizerr.CodeNotFound, "public key authentication is not enabled")
	}
	return nil
}
func (s *PublicKeyService) verifyReauthentication(user *model.User, password, totp string) bool {
	if password != "" && auth.CheckPassword(password, user.PasswordHash) {
		return true
	}
	if !user.TOTPEnabled || totp == "" || user.TOTPSecretEnc == "" {
		return false
	}
	secret, err := auth.DecryptSecret(user.TOTPSecretEnc, s.appCfg.EncryptKey)
	return err == nil && auth.ValidateCodeWithOptions(secret, totp)
}
func (s *PublicKeyService) loadUser(ctx context.Context, user *model.User) (*webAuthnUser, error) {
	records, err := s.repo.ListActive(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	credentials := make([]wa.Credential, 0, len(records))
	for i := range records {
		credential, convErr := toWebAuthnCredential(&records[i])
		if convErr != nil {
			return nil, convErr
		}
		credentials = append(credentials, credential)
	}
	return &webAuthnUser{user: user, credentials: credentials}, nil
}
func (s *PublicKeyService) storeCeremony(ctx context.Context, userID uint, kind, label string, options any, session *wa.SessionData) (*CeremonyOptions, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	expires := time.Now().Add(s.cfg.ChallengeTTL)
	if !session.Expires.IsZero() && session.Expires.Before(expires) {
		expires = session.Expires
	}
	if err := s.repo.CreateCeremony(ctx, &model.PublicKeyCeremony{ID: id, UserID: userID, Kind: kind, SessionJSON: string(data), Label: label, ExpiresAt: expires}); err != nil {
		return nil, err
	}
	return &CeremonyOptions{CeremonyID: id, Options: options}, nil
}
func (s *PublicKeyService) consume(ctx context.Context, id, kind string) (*model.PublicKeyCeremony, wa.SessionData, error) {
	var session wa.SessionData
	ceremony, err := s.repo.ConsumeCeremony(ctx, id, kind, time.Now())
	if err != nil {
		return nil, session, err
	}
	if err := json.Unmarshal([]byte(ceremony.SessionJSON), &session); err != nil {
		return nil, session, err
	}
	return ceremony, session, nil
}
func fromWebAuthnCredential(userID uint, label string, c *wa.Credential) *model.PublicKeyCredential {
	transports, _ := json.Marshal(c.Transport)
	return &model.PublicKeyCredential{UserID: userID, CredentialID: c.ID, PublicKey: c.PublicKey, Attestation: c.AttestationType, TransportsJSON: string(transports), AAGUID: c.Authenticator.AAGUID, Attachment: string(c.Authenticator.Attachment), SignCount: c.Authenticator.SignCount, BackupEligible: c.Flags.BackupEligible, BackupState: c.Flags.BackupState, Label: label}
}
func toWebAuthnCredential(c *model.PublicKeyCredential) (wa.Credential, error) {
	var transports []protocol.AuthenticatorTransport
	if c.TransportsJSON != "" {
		if err := json.Unmarshal([]byte(c.TransportsJSON), &transports); err != nil {
			return wa.Credential{}, err
		}
	}
	flags := protocol.FlagUserPresent
	if c.BackupEligible {
		flags |= protocol.FlagBackupEligible
	}
	if c.BackupState {
		flags |= protocol.FlagBackupState
	}
	return wa.Credential{ID: c.CredentialID, PublicKey: c.PublicKey, AttestationType: c.Attestation, Transport: transports, Flags: wa.NewCredentialFlags(flags), Authenticator: wa.Authenticator{AAGUID: c.AAGUID, SignCount: c.SignCount, Attachment: protocol.AuthenticatorAttachment(c.Attachment)}}, nil
}
func credentialInfo(c *model.PublicKeyCredential) *PublicKeyCredentialInfo {
	var transports []string
	_ = json.Unmarshal([]byte(c.TransportsJSON), &transports)
	return &PublicKeyCredentialInfo{ID: c.ID, Label: c.Label, Transports: transports, CreatedAt: c.CreatedAt, LastUsedAt: c.LastUsedAt, BackupEligible: c.BackupEligible, BackupState: c.BackupState}
}
func isUniqueViolation(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unique") || strings.Contains(text, "duplicate")
}
