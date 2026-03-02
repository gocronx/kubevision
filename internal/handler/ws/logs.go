package ws

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	kubexec "github.com/kubevision/kubevision/internal/kubernetes/exec"
	"github.com/kubevision/kubevision/internal/repository"
)

// logMsgType classifies messages on the logs WebSocket channel.
type logMsgType string

const (
	logMsgLine  logMsgType = "log"
	logMsgError logMsgType = "error"
	logMsgClose logMsgType = "close"
)

// logMessage is the JSON envelope for all log WebSocket messages.
type logMessage struct {
	Type logMsgType `json:"type"`
	Data string     `json:"data,omitempty"`
}

// LogsHandler handles WebSocket-based pod log streaming sessions.
type LogsHandler struct {
	clusterManager *cluster.Manager
	clusterRepo    repository.ClusterRepo
	jwtManager     *auth.JWTManager
	userRepo       repository.UserRepo
	logger         *zap.Logger
}

// NewLogsHandler creates a new LogsHandler.
func NewLogsHandler(
	clusterManager *cluster.Manager,
	clusterRepo repository.ClusterRepo,
	jwtManager *auth.JWTManager,
	userRepo repository.UserRepo,
	logger *zap.Logger,
) *LogsHandler {
	return &LogsHandler{
		clusterManager: clusterManager,
		clusterRepo:    clusterRepo,
		jwtManager:     jwtManager,
		userRepo:       userRepo,
		logger:         logger,
	}
}

// HandleLogs is the Gin handler for:
//
//	GET /api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs
//
// Query parameters:
//
//	token      - JWT access token (required)
//	container  - container name (optional)
//	follow     - "true" to stream in real-time (default: false)
//	previous   - "true" to show logs from the previous container instance
//	tailLines  - number of lines from the end to show (omit for all)
//	timestamps - "true" to prefix each line with its timestamp
func (h *LogsHandler) HandleLogs(c *gin.Context) {
	// --- Authentication via query-param token ---
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(400, gin.H{"error": "missing token query parameter"})
		return
	}
	if _, err := h.authenticateToken(c, tokenStr); err != nil {
		c.JSON(401, gin.H{"error": "unauthorized: " + err.Error()})
		return
	}

	// --- Route parameters ---
	clusterIDStr := c.Param("id")
	namespace := c.Param("namespace")
	podName := c.Param("name")

	// --- Query parameters ---
	container := c.Query("container")
	follow := c.Query("follow") == "true"
	previous := c.Query("previous") == "true"
	timestamps := c.Query("timestamps") == "true"

	var tailLines *int64
	if tl := c.Query("tailLines"); tl != "" {
		if n, err := strconv.ParseInt(tl, 10, 64); err == nil && n > 0 {
			tailLines = &n
		}
	}

	// --- Resolve cluster ---
	clusterName, err := resolveClusterName(c.Request.Context(), clusterIDStr, h.clusterRepo)
	if err != nil {
		c.JSON(404, gin.H{"error": "cluster not found: " + err.Error()})
		return
	}

	restConfig, err := h.clusterManager.RESTConfig(clusterName)
	if err != nil {
		c.JSON(404, gin.H{"error": "cluster not available: " + err.Error()})
		return
	}

	clientset, err := kubexec.NewClientset(restConfig)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create client: " + err.Error()})
		return
	}

	// --- Upgrade to WebSocket ---
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("logs ws upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	h.logger.Info("log stream started",
		zap.String("pod", podName),
		zap.String("namespace", namespace),
		zap.String("container", container),
		zap.Bool("follow", follow),
	)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Ping loop keeps the WebSocket alive while logs stream.
	go h.pingLoop(ctx, conn)

	// Drain incoming client messages; a close frame cancels the context.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Create a pipe so we can process the log stream line by line.
	pr, pw := io.Pipe()

	streamDone := make(chan error, 1)
	go func() {
		opts := kubexec.LogOptions{
			Namespace:  namespace,
			PodName:    podName,
			Container:  container,
			Follow:     follow,
			Previous:   previous,
			TailLines:  tailLines,
			Timestamps: timestamps,
		}
		streamDone <- kubexec.StreamLogs(ctx, clientset, opts, pw)
		pw.Close()
	}()

	// Read lines from the log stream and forward them to the WebSocket.
	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for {
		select {
		case <-ctx.Done():
			pr.Close()
			h.sendLogMsg(conn, logMsgClose, "session ended")
			return
		default:
		}

		if !scanner.Scan() {
			break
		}

		h.sendLogMsg(conn, logMsgLine, scanner.Text())
	}

	// Wait for the streaming goroutine and surface any error.
	if err := <-streamDone; err != nil && ctx.Err() == nil {
		h.logger.Warn("log stream ended with error",
			zap.String("pod", podName),
			zap.Error(err),
		)
		h.sendLogMsg(conn, logMsgError, err.Error())
	}

	h.sendLogMsg(conn, logMsgClose, "stream ended")
	h.logger.Info("log stream ended", zap.String("pod", podName))
}

// pingLoop sends WebSocket ping frames to keep the connection alive.
func (h *LogsHandler) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendLogMsg writes a log message to the WebSocket connection.
func (h *LogsHandler) sendLogMsg(conn *websocket.Conn, msgType logMsgType, data string) {
	msg := logMessage{Type: msgType, Data: data}
	raw, _ := json.Marshal(msg)
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, raw)
}

// authenticateToken validates a JWT token string and checks the user in the database.
func (h *LogsHandler) authenticateToken(c *gin.Context, tokenStr string) (*auth.TokenClaims, error) {
	claims, err := h.jwtManager.ParseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	user, err := h.userRepo.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, errAccountDisabled
	}
	if claims.TokenVersion != user.TokenVersion {
		return nil, errTokenRevoked
	}
	return claims, nil
}
