package repository

import (
	"context"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuditRepo(t *testing.T) AuditRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewAuditRepo(db)
}

func TestAuditRepo_BatchCreateAndList(t *testing.T) {
	repo := newTestAuditRepo(t)
	ctx := context.Background()

	logs := []model.AuditLog{
		{CreatedAt: time.Now(), UserID: 1, Username: "admin", Action: "create", Resource: "pods", StatusCode: 201},
		{CreatedAt: time.Now(), UserID: 1, Username: "admin", Action: "delete", Resource: "services", StatusCode: 200},
		{CreatedAt: time.Now(), UserID: 2, Username: "dev", Action: "update", Resource: "deployments", StatusCode: 200},
	}

	err := repo.BatchCreate(ctx, logs)
	require.NoError(t, err)

	results, total, err := repo.List(ctx, AuditFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, results, 3)
}

func TestAuditRepo_ListWithFilter(t *testing.T) {
	repo := newTestAuditRepo(t)
	ctx := context.Background()

	logs := []model.AuditLog{
		{CreatedAt: time.Now(), UserID: 1, Username: "admin", Action: "create", Resource: "pods", Cluster: "prod", StatusCode: 201},
		{CreatedAt: time.Now(), UserID: 1, Username: "admin", Action: "delete", Resource: "pods", Cluster: "dev", StatusCode: 200},
		{CreatedAt: time.Now(), UserID: 2, Username: "dev", Action: "create", Resource: "services", Cluster: "prod", StatusCode: 201},
	}
	require.NoError(t, repo.BatchCreate(ctx, logs))

	// Filter by action.
	results, total, err := repo.List(ctx, AuditFilter{Action: "create", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Filter by cluster.
	results, total, err = repo.List(ctx, AuditFilter{Cluster: "prod", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)
}

func TestAuditRepo_ListWithPagination(t *testing.T) {
	repo := newTestAuditRepo(t)
	ctx := context.Background()

	logs := make([]model.AuditLog, 5)
	for i := range logs {
		logs[i] = model.AuditLog{
			CreatedAt:  time.Now(),
			UserID:     1,
			Username:   "admin",
			Action:     "create",
			Resource:   "pods",
			StatusCode: 201,
		}
	}
	require.NoError(t, repo.BatchCreate(ctx, logs))

	results, total, err := repo.List(ctx, AuditFilter{Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, results, 2)

	results, _, err = repo.List(ctx, AuditFilter{Offset: 3, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestAuditRepo_Purge(t *testing.T) {
	repo := newTestAuditRepo(t)
	ctx := context.Background()

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now()

	logs := []model.AuditLog{
		{CreatedAt: old, UserID: 1, Username: "admin", Action: "create", Resource: "pods", StatusCode: 201},
		{CreatedAt: old, UserID: 1, Username: "admin", Action: "delete", Resource: "pods", StatusCode: 200},
		{CreatedAt: recent, UserID: 1, Username: "admin", Action: "update", Resource: "pods", StatusCode: 200},
	}
	require.NoError(t, repo.BatchCreate(ctx, logs))

	cutoff := time.Now().Add(-24 * time.Hour)
	deleted, err := repo.Purge(ctx, cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	results, total, err := repo.List(ctx, AuditFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}
