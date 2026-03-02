package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock AuditRepo for handler tests
// ---------------------------------------------------------------------------

type stubAuditRepo struct {
	logs []model.AuditLog
}

func newStubAuditRepo(logs ...model.AuditLog) *stubAuditRepo {
	return &stubAuditRepo{logs: logs}
}

func (r *stubAuditRepo) BatchCreate(_ context.Context, logs []model.AuditLog) error {
	r.logs = append(r.logs, logs...)
	return nil
}

func (r *stubAuditRepo) List(_ context.Context, filter repository.AuditFilter) ([]model.AuditLog, int64, error) {
	var filtered []model.AuditLog
	for _, l := range r.logs {
		if filter.Action != "" && l.Action != filter.Action {
			continue
		}
		if filter.Cluster != "" && l.Cluster != filter.Cluster {
			continue
		}
		filtered = append(filtered, l)
	}
	total := int64(len(filtered))

	// Apply pagination.
	start := filter.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (r *stubAuditRepo) Purge(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

var _ repository.AuditRepo = (*stubAuditRepo)(nil)

func setupAuditHandler(t *testing.T, role string) (*gin.Engine, *AuditHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newStubAuditRepo(
		model.AuditLog{UserID: 1, Action: "create", Cluster: "prod", Resource: "pods", CreatedAt: time.Now()},
		model.AuditLog{UserID: 2, Action: "delete", Cluster: "staging", Resource: "deployments", CreatedAt: time.Now()},
	)
	handler := NewAuditHandler(repo)

	router := gin.New()
	router.Use(fakeAuth(1, role))
	router.GET("/api/v1/audit-logs", handler.List)
	return router, handler
}

func TestAuditHandler_List_AdminSuccess(t *testing.T) {
	router, _ := setupAuditHandler(t, "admin")

	w := performRequest(router, "GET", "/api/v1/audit-logs", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestAuditHandler_List_OpsAllowed(t *testing.T) {
	router, _ := setupAuditHandler(t, "ops")

	w := performRequest(router, "GET", "/api/v1/audit-logs", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestAuditHandler_List_ViewerForbidden(t *testing.T) {
	router, _ := setupAuditHandler(t, "viewer")

	w := performRequest(router, "GET", "/api/v1/audit-logs", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40300, resp.Code)
}

func TestAuditHandler_List_NoRoleForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newStubAuditRepo()
	handler := NewAuditHandler(repo)

	router := gin.New()
	// No fakeAuth — role is empty.
	router.GET("/api/v1/audit-logs", handler.List)

	w := performRequest(router, "GET", "/api/v1/audit-logs", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40300, resp.Code)
}

func TestAuditHandler_List_FilterByAction(t *testing.T) {
	router, _ := setupAuditHandler(t, "admin")

	w := performRequest(router, "GET", "/api/v1/audit-logs?action=create", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var logs []model.AuditLog
	require.NoError(t, json.Unmarshal(resp.Data, &logs))
	assert.Len(t, logs, 1)
}

func TestAuditHandler_List_FilterByCluster(t *testing.T) {
	router, _ := setupAuditHandler(t, "admin")

	w := performRequest(router, "GET", "/api/v1/audit-logs?cluster=staging", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var logs []model.AuditLog
	require.NoError(t, json.Unmarshal(resp.Data, &logs))
	assert.Len(t, logs, 1)
}

func TestAuditHandler_List_Pagination(t *testing.T) {
	router, _ := setupAuditHandler(t, "admin")

	w := performRequest(router, "GET", "/api/v1/audit-logs?limit=1&offset=0", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var logs []model.AuditLog
	require.NoError(t, json.Unmarshal(resp.Data, &logs))
	assert.Len(t, logs, 1)
}
