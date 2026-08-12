package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPublicKeyRepoTestDB(t *testing.T) (*gorm.DB, *PublicKeyRepo) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "public-key.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PublicKeyCredential{}, &model.PublicKeyCeremony{}))
	return db, NewPublicKeyRepo(db)
}

func TestPublicKeyCeremonyConsumedOnceConcurrently(t *testing.T) {
	_, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, repo.CreateCeremony(ctx, &model.PublicKeyCeremony{ID: "single-use", Kind: "login", SessionJSON: `{}`, ExpiresAt: time.Now().Add(time.Minute)}))

	const contenders = 12
	start := make(chan struct{})
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := repo.ConsumeCeremony(ctx, "single-use", "login", time.Now()); err == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	require.Equal(t, int32(1), successes.Load())
	_, err := repo.ConsumeCeremony(ctx, "single-use", "login", time.Now())
	require.ErrorIs(t, err, ErrCeremonyUnavailable)
}

func TestPublicKeyCeremonyRejectsExpiredAndWrongKind(t *testing.T) {
	_, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, repo.CreateCeremony(ctx, &model.PublicKeyCeremony{ID: "expired", Kind: "login", SessionJSON: `{}`, ExpiresAt: time.Now().Add(-time.Second)}))
	_, err := repo.ConsumeCeremony(ctx, "expired", "login", time.Now())
	require.ErrorIs(t, err, ErrCeremonyUnavailable)
	require.NoError(t, repo.CreateCeremony(ctx, &model.PublicKeyCeremony{ID: "registration", Kind: "registration", SessionJSON: `{}`, ExpiresAt: time.Now().Add(time.Minute)}))
	_, err = repo.ConsumeCeremony(ctx, "registration", "login", time.Now())
	require.ErrorIs(t, err, ErrCeremonyUnavailable)
}

func TestPublicKeyCredentialIDIsGloballyUnique(t *testing.T) {
	_, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, repo.CreateCredential(ctx, &model.PublicKeyCredential{UserID: 1, CredentialID: []byte("credential"), PublicKey: []byte("key"), Label: "first"}))
	require.Error(t, repo.CreateCredential(ctx, &model.PublicKeyCredential{UserID: 2, CredentialID: []byte("credential"), PublicKey: []byte("key"), Label: "second"}))
}

func TestPublicKeyRevokeProtectsLastMethod(t *testing.T) {
	db, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&model.User{ID: 7, Username: "owner", PasswordHash: "unused", IsActive: true}).Error)
	credential := &model.PublicKeyCredential{UserID: 7, CredentialID: []byte("only"), PublicKey: []byte("key"), Label: "only"}
	require.NoError(t, repo.CreateCredential(ctx, credential))
	err := repo.Revoke(ctx, 7, credential.ID, false, time.Now())
	require.True(t, errors.Is(err, ErrLastAuthenticationMethod))
	active, listErr := repo.ListActive(ctx, 7)
	require.NoError(t, listErr)
	require.Len(t, active, 1)
	require.NoError(t, repo.Revoke(ctx, 7, credential.ID, true, time.Now()))
	active, listErr = repo.ListActive(ctx, 7)
	require.NoError(t, listErr)
	require.Empty(t, active)
}

func TestPublicKeyCredentialOwnershipRequiredForMutations(t *testing.T) {
	db, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&model.User{ID: 3, Username: "owner", PasswordHash: "unused", IsActive: true}).Error)
	require.NoError(t, db.Create(&model.User{ID: 4, Username: "other", PasswordHash: "unused", IsActive: true}).Error)
	credential := &model.PublicKeyCredential{UserID: 3, CredentialID: []byte("owned"), PublicKey: []byte("key"), Label: "owned"}
	require.NoError(t, repo.CreateCredential(ctx, credential))
	require.ErrorIs(t, repo.Rename(ctx, 4, credential.ID, "stolen"), gorm.ErrRecordNotFound)
	require.ErrorIs(t, repo.Revoke(ctx, 4, credential.ID, true, time.Now()), gorm.ErrRecordNotFound)
}

func TestPublicKeyConcurrentRevokeKeepsOneMethod(t *testing.T) {
	db, repo := newPublicKeyRepoTestDB(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&model.User{ID: 9, Username: "concurrent", PasswordHash: "", IsActive: true}).Error)
	first := &model.PublicKeyCredential{UserID: 9, CredentialID: []byte("first"), PublicKey: []byte("key"), Label: "first"}
	second := &model.PublicKeyCredential{UserID: 9, CredentialID: []byte("second"), PublicKey: []byte("key"), Label: "second"}
	require.NoError(t, repo.CreateCredential(ctx, first))
	require.NoError(t, repo.CreateCredential(ctx, second))
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, id := range []uint{first.ID, second.ID} {
		go func() { <-start; errs <- repo.Revoke(ctx, 9, id, false, time.Now()) }()
	}
	close(start)
	<-errs
	<-errs
	active, err := repo.ListActive(ctx, 9)
	require.NoError(t, err)
	require.NotEmpty(t, active)
}
