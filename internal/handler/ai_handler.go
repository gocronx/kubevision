package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

type listModelsRequest struct {
	BaseURL string `json:"baseURL"`
	APIKey  string `json:"apiKey"`
}

// ListModels discovers model IDs from an OpenAI-compatible provider. An empty
// API key reuses the persisted key, allowing the UI to refresh models without
// exposing or resending stored credentials.
func (h *AIHandler) ListModels(c *gin.Context) {
	if role := middleware.GetUserRole(c); role != "super-admin" && role != "admin" {
		response.Error(c, bizerr.CodeForbidden, "only administrators can discover AI models")
		return
	}

	var req listModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}
	existing, err := h.svc.Config().Load(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to load AI configuration")
		return
	}
	baseURLChanged := req.BaseURL != "" && normalizedBaseURL(req.BaseURL) != normalizedBaseURL(existing.BaseURL)
	if baseURLChanged && req.APIKey == "" {
		response.Error(c, bizerr.CodeParamMissing, "API key is required when changing the AI provider URL")
		return
	}
	if req.BaseURL != "" {
		existing.BaseURL = req.BaseURL
	}
	if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}
	if existing.APIKey == "" {
		response.Error(c, bizerr.CodeParamMissing, "API key is required")
		return
	}

	models, err := h.svc.ListModels(c.Request.Context(), existing)
	if err != nil {
		response.Error(c, bizerr.CodeInternal, err.Error())
		return
	}
	response.Success(c, models)
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

	if normalizedBaseURL(req.BaseURL) != normalizedBaseURL(existing.BaseURL) && req.APIKey == "" {
		response.Error(c, bizerr.CodeParamMissing, "API key is required when changing the AI provider URL")
		return
	}
	if req.MaxTokens < 1 || req.MaxTokens > 32768 {
		response.Error(c, bizerr.CodeParamInvalid, "maxTokens must be between 1 and 32768")
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

func normalizedBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
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

const (
	maxAIChatBodyBytes = 256 * 1024
	maxAIChatMessages  = 100
	maxAIMessageBytes  = 32 * 1024
)

// Chat handles POST /api/v1/ai/chat and streams the agent's response as SSE.
func (h *AIHandler) Chat(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAIChatBodyBytes)
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
	if len(req.Messages) > maxAIChatMessages {
		response.Error(c, bizerr.CodeParamInvalid, "too many messages")
		return
	}

	history := make([]ai.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		if len(m.Content) > maxAIMessageBytes {
			response.Error(c, bizerr.CodeParamInvalid, "message is too large")
			return
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		history = append(history, ai.Message{Role: m.Role, Content: m.Content})
	}
	if len(history) == 0 || history[len(history)-1].Role != "user" {
		response.Error(c, bizerr.CodeParamInvalid, "the last message must be from the user")
		return
	}

	params := ai.ChatParams{
		ClusterID: req.ClusterID,
		UserID:    middleware.GetUserID(c),
		Username:  middleware.GetUsername(c),
		UserRole:  middleware.GetUserRole(c),
		ClientIP:  c.ClientIP(),
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
	h.svc.ContinueAction(c.Request.Context(), req.SessionID, ai.Actor{
		UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c),
		Role: middleware.GetUserRole(c), ClientIP: c.ClientIP(),
	}, emit)
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
