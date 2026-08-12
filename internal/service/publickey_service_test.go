package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPublicKeyServiceForTest(t *testing.T) (*PublicKeyService, *gorm.DB, *model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PublicKeyCredential{}, &model.PublicKeyCeremony{}))
	hash, err := auth.HashPassword("strong-password")
	require.NoError(t, err)
	user := &model.User{Username: "passkey-user", PasswordHash: hash, Role: "viewer", IsActive: true}
	require.NoError(t, db.Create(user).Error)
	cfg := config.Default()
	cfg.EncryptKey = strings.Repeat("a", 64)
	cfg.Auth.JWTSecret = strings.Repeat("b", 64)
	cfg.Auth.PublicKey = config.PublicKeyAuthConfig{Enabled: true, RPID: "example.test", RPDisplayName: "KubeVision Test", Origins: []string{"https://example.test"}, UserVerification: "required", CounterPolicy: "deny", ChallengeTTL: 2 * time.Minute}
	users := repository.NewUserRepo(db)
	jwt := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL)
	authService := NewAuthService(users, jwt, cfg, zap.NewNop())
	service, err := NewPublicKeyService(repository.NewPublicKeyRepo(db), users, authService, cfg, zap.NewNop())
	require.NoError(t, err)
	return service, db, user
}

func TestPublicKeyBeginLoginBindsRPIDUVAndExpiration(t *testing.T) {
	service, db, _ := newPublicKeyServiceForTest(t)
	result, err := service.BeginLogin(context.Background(), "")
	require.NoError(t, err)
	data, err := json.Marshal(result.Options)
	require.NoError(t, err)
	require.Contains(t, string(data), `"rpId":"example.test"`)
	require.Contains(t, string(data), `"userVerification":"required"`)

	var ceremony model.PublicKeyCeremony
	require.NoError(t, db.First(&ceremony, "id = ?", result.CeremonyID).Error)
	require.WithinDuration(t, time.Now().Add(2*time.Minute), ceremony.ExpiresAt, 2*time.Second)
	var session map[string]any
	require.NoError(t, json.Unmarshal([]byte(ceremony.SessionJSON), &session))
	require.Equal(t, "example.test", session["rpId"])
	require.Equal(t, "required", session["userVerification"])
}

func TestPublicKeyMalformedFinishConsumesChallenge(t *testing.T) {
	service, _, _ := newPublicKeyServiceForTest(t)
	ctx := context.Background()
	begin, err := service.BeginLogin(ctx, "")
	require.NoError(t, err)
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"not":"a credential"}`))
	request.Header.Set("Content-Type", "application/json")
	_, err = service.FinishLogin(ctx, begin.CeremonyID, request)
	require.Error(t, err)
	first := err.(*bizerr.BizError)
	require.Equal(t, bizerr.CodeUnauthorized, first.Code)

	replay := httptest.NewRequest("POST", "/", strings.NewReader(`{"not":"a credential"}`))
	replay.Header.Set("Content-Type", "application/json")
	_, err = service.FinishLogin(ctx, begin.CeremonyID, replay)
	require.Error(t, err)
	require.Equal(t, bizerr.CodeUnauthorized, err.(*bizerr.BizError).Code)
}

func TestPublicKeySessionIssuanceUsesExistingAuthRules(t *testing.T) {
	service, db, user := newPublicKeyServiceForTest(t)
	result, err := service.auth.IssueLoginForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, user.ID, result.User.ID)
	require.Equal(t, user.Role, result.User.Role)

	require.NoError(t, db.Model(user).Update("is_active", false).Error)
	_, err = service.auth.IssueLoginForUser(context.Background(), user.ID)
	require.Error(t, err)
	require.Equal(t, bizerr.CodeForbidden, err.(*bizerr.BizError).Code)
}

func TestPublicKeyConfigRejectsUnsafePolicyValues(t *testing.T) {
	cfg := config.Default()
	cfg.Auth.PublicKey = config.PublicKeyAuthConfig{Enabled: true, RPID: "example.test", RPDisplayName: "Test", Origins: []string{"https://example.test"}, UserVerification: "never", CounterPolicy: "ignore", ChallengeTTL: 2 * time.Minute}
	_, err := NewPublicKeyService(nil, nil, nil, cfg, zap.NewNop())
	require.Error(t, err)
}

func TestPublicKeyConfigRejectsUnrelatedOrigin(t *testing.T) {
	cfg := config.Default()
	cfg.Auth.PublicKey = config.PublicKeyAuthConfig{Enabled: true, RPID: "example.test", RPDisplayName: "Test", Origins: []string{"https://attacker.test"}, UserVerification: "required", CounterPolicy: "deny", ChallengeTTL: 2 * time.Minute}
	_, err := NewPublicKeyService(nil, nil, nil, cfg, zap.NewNop())
	require.Error(t, err)
}
