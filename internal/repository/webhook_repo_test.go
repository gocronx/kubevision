package repository

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWebhookRepo(t *testing.T) WebhookRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewWebhookRepo(db)
}

func TestWebhookRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestWebhookRepo(t)
	ctx := context.Background()

	wh := &model.Webhook{
		Name:     "test-webhook",
		URL:      "https://example.com/hook",
		Secret:   "secret123",
		Events:   `["create","delete"]`,
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, wh))
	assert.NotZero(t, wh.ID)

	got, err := repo.GetByID(ctx, wh.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-webhook", got.Name)
	assert.Equal(t, "https://example.com/hook", got.URL)
	assert.True(t, got.IsActive)
}

func TestWebhookRepo_Update(t *testing.T) {
	repo := newTestWebhookRepo(t)
	ctx := context.Background()

	wh := &model.Webhook{Name: "wh1", URL: "https://old.com", IsActive: true}
	require.NoError(t, repo.Create(ctx, wh))

	wh.URL = "https://new.com"
	wh.IsActive = false
	require.NoError(t, repo.Update(ctx, wh))

	got, err := repo.GetByID(ctx, wh.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://new.com", got.URL)
	assert.False(t, got.IsActive)
}

func TestWebhookRepo_Delete(t *testing.T) {
	repo := newTestWebhookRepo(t)
	ctx := context.Background()

	wh := &model.Webhook{Name: "delete-me", URL: "https://del.com", IsActive: true}
	require.NoError(t, repo.Create(ctx, wh))
	require.NoError(t, repo.Delete(ctx, wh.ID))

	_, err := repo.GetByID(ctx, wh.ID)
	require.Error(t, err)
}

func TestWebhookRepo_List(t *testing.T) {
	repo := newTestWebhookRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &model.Webhook{Name: "wh1", URL: "https://a.com", IsActive: true}))
	require.NoError(t, repo.Create(ctx, &model.Webhook{Name: "wh2", URL: "https://b.com", IsActive: false}))

	all, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestWebhookRepo_ListActive(t *testing.T) {
	repo := newTestWebhookRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &model.Webhook{Name: "active", URL: "https://a.com", IsActive: true}))
	inactive := &model.Webhook{Name: "inactive", URL: "https://b.com", IsActive: true}
	require.NoError(t, repo.Create(ctx, inactive))
	// Update to inactive (GORM ignores false on Create due to default:true).
	inactive.IsActive = false
	require.NoError(t, repo.Update(ctx, inactive))

	active, err := repo.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "active", active[0].Name)
}
