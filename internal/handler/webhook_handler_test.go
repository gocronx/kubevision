package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub WebhookRepo for handler tests
// ---------------------------------------------------------------------------

type stubWebhookRepo struct {
	webhooks map[uint]*model.Webhook
	nextID   uint
}

func newStubWebhookRepo() *stubWebhookRepo {
	return &stubWebhookRepo{
		webhooks: make(map[uint]*model.Webhook),
		nextID:   1,
	}
}

func (r *stubWebhookRepo) Create(_ context.Context, wh *model.Webhook) error {
	wh.ID = r.nextID
	r.nextID++
	cp := *wh
	r.webhooks[wh.ID] = &cp
	return nil
}

func (r *stubWebhookRepo) GetByID(_ context.Context, id uint) (*model.Webhook, error) {
	wh, ok := r.webhooks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *wh
	return &cp, nil
}

func (r *stubWebhookRepo) Update(_ context.Context, wh *model.Webhook) error {
	if _, ok := r.webhooks[wh.ID]; !ok {
		return errors.New("not found")
	}
	cp := *wh
	r.webhooks[wh.ID] = &cp
	return nil
}

func (r *stubWebhookRepo) Delete(_ context.Context, id uint) error {
	delete(r.webhooks, id)
	return nil
}

func (r *stubWebhookRepo) List(_ context.Context) ([]model.Webhook, error) {
	var result []model.Webhook
	for _, wh := range r.webhooks {
		result = append(result, *wh)
	}
	return result, nil
}

func (r *stubWebhookRepo) ListActive(_ context.Context) ([]model.Webhook, error) {
	var result []model.Webhook
	for _, wh := range r.webhooks {
		if wh.IsActive {
			result = append(result, *wh)
		}
	}
	return result, nil
}

var _ repository.WebhookRepo = (*stubWebhookRepo)(nil)

func setupWebhookHandler(t *testing.T) (*gin.Engine, *WebhookHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newStubWebhookRepo()
	svc := service.NewWebhookService(repo, nil)
	handler := NewWebhookHandler(svc)

	router := gin.New()
	router.GET("/api/v1/webhooks", handler.List)
	router.POST("/api/v1/webhooks", handler.Create)
	router.PUT("/api/v1/webhooks/:id", handler.Update)
	router.DELETE("/api/v1/webhooks/:id", handler.Delete)
	router.POST("/api/v1/webhooks/:id/test", handler.Test)
	return router, handler
}

func TestWebhookHandler_Create_Success(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	body := service.WebhookRequest{
		Name:     "hook-1",
		URL:      "https://8.8.8.8/webhook",
		Events:   []string{"create"},
		IsActive: true,
	}
	w := performRequest(router, "POST", "/api/v1/webhooks", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestWebhookHandler_Create_MissingName(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	body := map[string]string{"url": "https://example.com"}
	w := performRequest(router, "POST", "/api/v1/webhooks", body)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestWebhookHandler_Create_MissingURL(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	body := map[string]string{"name": "hook"}
	w := performRequest(router, "POST", "/api/v1/webhooks", body)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestWebhookHandler_List_Empty(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	w := performRequest(router, "GET", "/api/v1/webhooks", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestWebhookHandler_List_WithData(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	// Create two webhooks.
	performRequest(router, "POST", "/api/v1/webhooks", service.WebhookRequest{
		Name: "wh1", URL: "https://8.8.8.8/a", IsActive: true,
	})
	performRequest(router, "POST", "/api/v1/webhooks", service.WebhookRequest{
		Name: "wh2", URL: "https://1.1.1.1/b", IsActive: true,
	})

	w := performRequest(router, "GET", "/api/v1/webhooks", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var whs []service.WebhookResponse
	require.NoError(t, json.Unmarshal(resp.Data, &whs))
	assert.Len(t, whs, 2)
}

func TestWebhookHandler_Update_InvalidID(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	body := service.WebhookRequest{Name: "wh", URL: "https://new.com"}
	w := performRequest(router, "PUT", "/api/v1/webhooks/abc", body)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestWebhookHandler_Delete_InvalidID(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	w := performRequest(router, "DELETE", "/api/v1/webhooks/abc", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestWebhookHandler_Delete_Success(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	// Create then delete.
	performRequest(router, "POST", "/api/v1/webhooks", service.WebhookRequest{
		Name: "del-me", URL: "https://8.8.4.4/delete", IsActive: true,
	})

	w := performRequest(router, "DELETE", "/api/v1/webhooks/1", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestWebhookHandler_Test_InvalidID(t *testing.T) {
	router, _ := setupWebhookHandler(t)

	w := performRequest(router, "POST", "/api/v1/webhooks/abc/test", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}
