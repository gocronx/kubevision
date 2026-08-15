package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

// TicketHandler issues short-lived credentials for WebSocket upgrades.
type TicketHandler struct {
	jwtManager *auth.JWTManager
}

func NewTicketHandler(jwtManager *auth.JWTManager) *TicketHandler {
	return &TicketHandler{jwtManager: jwtManager}
}

// Create handles POST /api/v1/ws/ticket. The route group authenticates the
// request before this handler runs.
func (h *TicketHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 || h.jwtManager == nil {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}
	ticket, err := h.jwtManager.GenerateWebSocketTicket(userID)
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to create websocket ticket")
		return
	}
	response.Success(c, gin.H{"ticket": ticket, "expiresIn": 30})
}
