package repository

import (
	"context"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAPIKeyRepo(t *testing.T) APIKeyRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewAPIKeyRepo(db)
}

func TestAPIKeyRepo_CreateAndGetByKeyHash(t *testing.T) {
	repo := newTestAPIKeyRepo(t)
	ctx := context.Background()

	key := &model.APIKey{
		UserID:    1,
		Name:      "test-key",
		KeyHash:   "abc123hash",
		KeyPrefix: "kv_abc123",
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)
	assert.NotZero(t, key.ID)

	got, err := repo.GetByKeyHash(ctx, "abc123hash")
	require.NoError(t, err)
	assert.Equal(t, "test-key", got.Name)
	assert.Equal(t, uint(1), got.UserID)
}

func TestAPIKeyRepo_GetByKeyHash_NotFound(t *testing.T) {
	repo := newTestAPIKeyRepo(t)
	ctx := context.Background()

	_, err := repo.GetByKeyHash(ctx, "nonexistent")
	require.Error(t, err)
}

func TestAPIKeyRepo_ListByUser(t *testing.T) {
	repo := newTestAPIKeyRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &model.APIKey{
			UserID:    1,
			Name:      "key-" + string(rune('a'+i)),
			KeyHash:   "hash" + string(rune('a'+i)),
			KeyPrefix: "kv_" + string(rune('a'+i)),
		}))
	}
	// Another user's key.
	require.NoError(t, repo.Create(ctx, &model.APIKey{
		UserID:    2,
		Name:      "other-key",
		KeyHash:   "otherhash",
		KeyPrefix: "kv_other",
	}))

	keys, err := repo.ListByUser(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, keys, 3)

	keys2, err := repo.ListByUser(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, keys2, 1)
}

func TestAPIKeyRepo_Delete(t *testing.T) {
	repo := newTestAPIKeyRepo(t)
	ctx := context.Background()

	key := &model.APIKey{
		UserID:    1,
		Name:      "delete-key",
		KeyHash:   "deletehash",
		KeyPrefix: "kv_del",
	}
	require.NoError(t, repo.Create(ctx, key))

	require.NoError(t, repo.Delete(ctx, key.ID))

	_, err := repo.GetByKeyHash(ctx, "deletehash")
	require.Error(t, err)
}

func TestAPIKeyRepo_WithExpiry(t *testing.T) {
	repo := newTestAPIKeyRepo(t)
	ctx := context.Background()

	expiry := time.Now().Add(24 * time.Hour)
	key := &model.APIKey{
		UserID:    1,
		Name:      "expiry-key",
		KeyHash:   "expiryhash",
		KeyPrefix: "kv_exp",
		ExpiresAt: &expiry,
	}
	require.NoError(t, repo.Create(ctx, key))

	got, err := repo.GetByKeyHash(ctx, "expiryhash")
	require.NoError(t, err)
	assert.NotNil(t, got.ExpiresAt)
}
