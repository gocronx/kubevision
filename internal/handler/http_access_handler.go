package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/gocronx/kubevision/internal/kubernetes/httpaccess"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/repository"
)

const maxHTTPAccessResponseBytes = 32 << 20

const responseTruncatedTrailer = "X-KubeVision-Response-Truncated"

type kubernetesHTTPClients interface {
	RESTConfig(string) (*rest.Config, error)
	DynamicClient(string) (dynamic.Interface, error)
}

type auditRecorder interface {
	Record(model.AuditLog)
}

// HTTPAccessHandler provides read-only access to Pod and Service HTTP proxy
// subresources through an authenticated Kubernetes API server.
type HTTPAccessHandler struct {
	clusters    kubernetesHTTPClients
	clusterRepo repository.ClusterRepo
	roleRepo    repository.RoleRepo
	audit       auditRecorder
	logger      *zap.Logger
}

func NewHTTPAccessHandler(clients kubernetesHTTPClients, clusterRepo repository.ClusterRepo, roleRepo repository.RoleRepo, audit auditRecorder, logger *zap.Logger) *HTTPAccessHandler {
	return &HTTPAccessHandler{clusters: clients, clusterRepo: clusterRepo, roleRepo: roleRepo, audit: audit, logger: logger}
}

// Serve handles GET and HEAD requests for a declared Pod or Service endpoint.
func (h *HTTPAccessHandler) Serve(c *gin.Context) {
	started := time.Now()
	kind, namespace, name, port := c.Param("kind"), c.Param("namespace"), c.Param("name"), c.Query("port")
	normalizedPath := "/"
	defer func() {
		if h.audit == nil {
			return
		}
		h.audit.Record(model.AuditLog{
			CreatedAt: time.Now().UTC(), UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c),
			Action: "http-access", Resource: kind, Name: name, Namespace: namespace,
			Cluster: c.Param("id"), StatusCode: c.Writer.Status(), DurationMs: time.Since(started).Milliseconds(),
			Method: c.Request.Method, Port: port, Path: normalizedPath, ClientIP: c.ClientIP(),
		})
	}()

	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		response.Error(c, bizerr.CodeParamInvalid, "method not allowed")
		return
	}
	if c.Request.ContentLength > 0 || len(c.Request.TransferEncoding) > 0 {
		response.Error(c, bizerr.CodeParamInvalid, "request body is not allowed")
		return
	}
	if kind != "pods" && kind != "services" {
		response.Error(c, bizerr.CodeParamInvalid, "only pods and services are supported")
		return
	}
	if validation.IsDNS1123Subdomain(namespace) != nil || validation.IsDNS1123Subdomain(name) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid namespace or resource name")
		return
	}
	var err error
	normalizedPath, err = normalizedHTTPPath(c.Param("path"))
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid relative path")
		return
	}
	if !middleware.RoleHasPermission(c, h.roleRepo, kind, "get") {
		response.Error(c, bizerr.CodeForbidden, "permission denied")
		return
	}

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster ID")
		return
	}
	clusterRecord, err := h.clusterRepo.GetByID(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.Error(c, bizerr.CodeNotFound, "cluster not found")
		return
	}
	if err := h.validatePort(c, clusterRecord.Name, kind, namespace, name, port); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, err.Error())
		return
	}

	config, err := h.clusters.RESTConfig(clusterRecord.Name)
	if err != nil {
		response.Error(c, bizerr.CodeK8sUnavailable, "kubernetes cluster unavailable")
		return
	}
	client, err := httpaccess.NewClient(config)
	if err != nil {
		response.Error(c, bizerr.CodeK8sUnavailable, "kubernetes cluster unavailable")
		return
	}
	query := c.Request.URL.Query()
	query.Del("port")
	upstream, err := client.RoundTrip(c.Request, kind, namespace, name, port, normalizedPath, query)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("kubernetes HTTP access failed", zap.Error(err))
		}
		response.Error(c, bizerr.CodeK8sUnavailable, "target endpoint unavailable")
		return
	}
	defer upstream.Body.Close()
	if c.Request.Method == http.MethodGet && upstream.ContentLength > maxHTTPAccessResponseBytes {
		response.Error(c, bizerr.CodeK8sUnavailable, "target response exceeds size limit")
		return
	}
	filteredHeaders := httpaccess.FilterResponseHeaders(upstream.Header)
	if c.Request.Method == http.MethodGet {
		filteredHeaders.Del("Content-Length")
		if upstream.ContentLength < 0 {
			filteredHeaders.Set("Connection", "close")
			filteredHeaders.Add("Trailer", responseTruncatedTrailer)
		}
	}
	copyHeaders(c.Writer.Header(), filteredHeaders)
	c.Status(upstream.StatusCode)
	if c.Request.Method == http.MethodHead {
		return
	}
	written, copyErr := io.CopyN(c.Writer, upstream.Body, maxHTTPAccessResponseBytes)
	if copyErr != nil && !errors.Is(copyErr, io.EOF) {
		if h.logger != nil {
			h.logger.Warn("stream HTTP access response", zap.Error(copyErr))
		}
		return
	}
	if written < maxHTTPAccessResponseBytes {
		return
	}
	oneMore := []byte{0}
	count, readErr := upstream.Body.Read(oneMore)
	if count > 0 {
		c.Writer.Header().Set(responseTruncatedTrailer, "true")
		if h.logger != nil {
			h.logger.Warn("truncated Kubernetes HTTP access response", zap.Int64("limit", maxHTTPAccessResponseBytes))
		}
		return
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) && h.logger != nil {
		h.logger.Warn("stream HTTP access response", zap.Error(readErr))
	}
}

func (h *HTTPAccessHandler) validatePort(c *gin.Context, clusterName, kind, namespace, name, port string) error {
	if number, err := strconv.Atoi(port); err == nil {
		if number < 1 || number > 65535 {
			return errors.New("port must be between 1 and 65535")
		}
		return nil
	}
	if port == "" || validation.IsValidPortName(port) != nil {
		return errors.New("invalid port")
	}
	client, err := h.clusters.DynamicClient(clusterName)
	if err != nil {
		return errors.New("kubernetes cluster unavailable")
	}
	resource, err := client.Resource(schema.GroupVersionResource{Version: "v1", Resource: kind}).Namespace(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		return errors.New("resource not found")
	}
	if declaredNamedPort(resource.Object, kind, port) {
		return nil
	}
	return errors.New("named port is not declared by the resource")
}

func declaredNamedPort(object map[string]interface{}, kind, wanted string) bool {
	spec, _ := object["spec"].(map[string]interface{})
	if kind == "services" {
		ports, _ := spec["ports"].([]interface{})
		for _, item := range ports {
			if portMap, ok := item.(map[string]interface{}); ok && portMap["name"] == wanted {
				return true
			}
		}
		return false
	}
	containers, _ := spec["containers"].([]interface{})
	for _, item := range containers {
		container, _ := item.(map[string]interface{})
		ports, _ := container["ports"].([]interface{})
		for _, value := range ports {
			if portMap, ok := value.(map[string]interface{}); ok && portMap["name"] == wanted {
				return true
			}
		}
	}
	return false
}

func normalizedHTTPPath(raw string) (string, error) {
	if raw == "" || raw == "/" {
		return "/", nil
	}
	if strings.Contains(strings.ToLower(raw), "%25") || strings.Contains(raw, "\x00") {
		return "", errors.New("unsafe path")
	}
	parts := strings.Split(strings.TrimPrefix(raw, "/"), "/")
	decoded := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := url.PathUnescape(part)
		if err != nil || value == "." || value == ".." || strings.ContainsAny(value, "/\\\x00") || strings.Contains(value, "%") {
			return "", errors.New("unsafe path")
		}
		if value != "" {
			decoded = append(decoded, value)
		}
	}
	cleaned := "/" + path.Join(decoded...)
	if len(decoded) == 0 {
		return "/", nil
	}
	return cleaned, nil
}

func copyHeaders(destination, source http.Header) {
	for key, values := range source {
		for _, value := range values {
			destination.Add(key, value)
		}
	}
}
