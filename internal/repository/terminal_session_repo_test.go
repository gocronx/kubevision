package repository

import (
	"context"
	"testing"
	"time"

	"github.com/kubevision/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTerminalSessionRepo(t *testing.T) TerminalSessionRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewTerminalSessionRepo(db)
}

func TestTerminalSessionRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestTerminalSessionRepo(t)
	ctx := context.Background()

	sess := &model.TerminalSession{
		UserID:     1,
		Cluster:    "prod",
		Namespace:  "default",
		Pod:        "nginx-abc",
		Container:  "nginx",
		Recording:  `{"version":2}`,
		DurationMs: 5000,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, sess))
	assert.NotZero(t, sess.ID)

	got, err := repo.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Cluster)
	assert.Equal(t, "nginx-abc", got.Pod)
	assert.NotEmpty(t, got.Recording)
}

func TestTerminalSessionRepo_ListByUser(t *testing.T) {
	repo := newTestTerminalSessionRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &model.TerminalSession{
			UserID:    1,
			Cluster:   "prod",
			Pod:       "pod-" + string(rune('a'+i)),
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		}))
	}
	require.NoError(t, repo.Create(ctx, &model.TerminalSession{
		UserID:    2,
		Cluster:   "dev",
		Pod:       "other-pod",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}))

	sessions, err := repo.ListByUser(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestTerminalSessionRepo_PurgeExpired(t *testing.T) {
	repo := newTestTerminalSessionRepo(t)
	ctx := context.Background()

	// Create expired and non-expired sessions.
	require.NoError(t, repo.Create(ctx, &model.TerminalSession{
		UserID:    1,
		Pod:       "expired-pod",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}))
	require.NoError(t, repo.Create(ctx, &model.TerminalSession{
		UserID:    1,
		Pod:       "active-pod",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}))

	deleted, err := repo.PurgeExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	sessions, err := repo.ListByUser(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "active-pod", sessions[0].Pod)
}
