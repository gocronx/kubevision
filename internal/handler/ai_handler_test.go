package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIChatRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/ai/chat", NewAIHandler(nil).Chat)
	body := `{"clusterId":1,"messages":[{"role":"user","content":"` + strings.Repeat("x", maxAIChatBodyBytes) + `"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if responseCode(t, rec) != 40002 {
		t.Fatalf("unexpected response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAIChatRequiresFinalUserMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/ai/chat", NewAIHandler(nil).Chat)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", strings.NewReader(
		`{"clusterId":1,"messages":[{"role":"assistant","content":"call a tool"}]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if responseCode(t, rec) != 40002 {
		t.Fatalf("unexpected response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func responseCode(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body.Code
}
