package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gocronx/kubevision/internal/ai"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

// AIHandler exposes the AI assistant: streaming chat, mutation approval, and
// configuration management.
type AIHandler struct {
	svc *ai.Service
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(svc *ai.Service) *AIHandler {
	return &AIHandler{svc: svc}
}

// configView is the API-safe representation of the AI configuration. It never
// exposes the API key; it only reports whether one is set.
type configView struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"baseURL"`
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
	HasAPIKey bool   `json:"hasApiKey"`
}

// GetConfig handles GET /api/v1/ai/config. Any authenticated user may read it so
// the UI knows whether the assistant is available; the key is never returned.
func (h *AIHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.Config().Load(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to load AI configuration")
		return
	}
	response.Success(c, configView{
		Enabled:   cfg.Enabled,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		HasAPIKey: cfg.APIKey != "",
	})
}

type updateConfigRequest struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
}

// UpdateConfig handles PUT /api/v1/ai/config. Restricted to admins. An empty
// apiKey preserves the previously stored key so the UI need not re-send it.
func (h *AIHandler) UpdateConfig(c *gin.Context) {
	if role := middleware.GetUserRole(c); role != "super-admin" && role != "admin" {
		response.Error(c, bizerr.CodeForbidden, "only administrators can change AI settings")
		return
	}

	var req updateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	existing, err := h.svc.Config().Load(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to load AI configuration")
		return
	}

	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = existing.APIKey // keep the stored key
	}

	cfg := ai.Config{
		Enabled:   req.Enabled,
		BaseURL:   req.BaseURL,
		APIKey:    apiKey,
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
	}
	if err := h.svc.Config().Save(c.Request.Context(), cfg); err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to save AI configuration")
		return
	}

	response.Success(c, configView{
		Enabled:   cfg.Enabled,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		HasAPIKey: cfg.APIKey != "",
	})
}

type pageContext struct {
	Page         string `json:"page"`
	Namespace    string `json:"namespace"`
	ResourceName string `json:"resourceName"`
	ResourceKind string `json:"resourceKind"`
}

type chatRequest struct {
	ClusterID   uint          `json:"clusterId"`
	Messages    []chatMessage `json:"messages"`
	PageContext *pageContext  `json:"pageContext"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat handles POST /api/v1/ai/chat and streams the agent's response as SSE.
func (h *AIHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}
	if req.ClusterID == 0 {
		response.Error(c, bizerr.CodeParamMissing, "clusterId is required")
		return
	}
	if len(req.Messages) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "messages are required")
		return
	}

	history := make([]ai.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		history = append(history, ai.Message{Role: m.Role, Content: m.Content})
	}

	params := ai.ChatParams{
		ClusterID: req.ClusterID,
		UserRole:  middleware.GetUserRole(c),
		History:   history,
	}
	if req.PageContext != nil {
		params.Page = req.PageContext.Page
		params.Namespace = req.PageContext.Namespace
		params.ResourceKind = req.PageContext.ResourceKind
		params.ResourceName = req.PageContext.ResourceName
	}

	emit := newSSEWriter(c)
	h.svc.Chat(c.Request.Context(), params, emit)
}

type continueActionRequest struct {
	SessionID string `json:"sessionId"`
}

// ContinueAction handles POST /api/v1/ai/continue-action: the user has approved
// a pending mutation and the agent resumes, streaming the result as SSE.
func (h *AIHandler) ContinueAction(c *gin.Context) {
	var req continueActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}
	if req.SessionID == "" {
		response.Error(c, bizerr.CodeParamMissing, "sessionId is required")
		return
	}

	emit := newSSEWriter(c)
	h.svc.ContinueAction(c.Request.Context(), req.SessionID, emit)
}

// newSSEWriter prepares the response for Server-Sent Events and returns an emit
// function that serializes each event to the wire and flushes immediately.
func newSSEWriter(c *gin.Context) ai.EmitFunc {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)
	return func(ev ai.SSEEvent) {
		data, err := json.Marshal(ev.Data)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Event, data)
		if flusher != nil {
			flusher.Flush()
		}
	}
}
