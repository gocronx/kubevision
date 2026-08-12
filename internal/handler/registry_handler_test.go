package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/registry"
)

func TestRegistryHandlerRejectsInvalidReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &fakeHandlerClient{}
	h := NewRegistryHandler(registry.NewService(client, time.Minute, 1, 10))
	r := gin.New()
	r.GET("/api/v1/registry-tags", h.ListTags)
	w := performRequest(r, http.MethodGet, "/api/v1/registry-tags?image=https%3A%2F%2Fevil.example%2Fapp", nil)
	var result authAPIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 40002 {
		t.Fatalf("code=%d body=%s", result.Code, w.Body.String())
	}
}

type fakeHandlerClient struct{}

func (*fakeHandlerClient) Tags(_ context.Context, _ registry.Reference, _ int, _ string) ([]string, string, error) {
	return []string{"v1"}, "", nil
}
