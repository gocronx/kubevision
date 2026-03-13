package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// WebhookEvent carries the data dispatched to registered webhooks on a
// Kubernetes resource change.
type WebhookEvent struct {
	EventType string    `json:"eventType"`
	Cluster   string    `json:"cluster"`
	Resource  string    `json:"resource"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

// WebhookRequest is the API request body for creating or updating a webhook.
type WebhookRequest struct {
	Name      string   `json:"name"      binding:"required"`
	URL       string   `json:"url"       binding:"required"`
	Secret    string   `json:"secret"`
	Events    []string `json:"events"`
	Clusters  []string `json:"clusters"`
	Resources []string `json:"resources"`
	IsActive  bool     `json:"isActive"`
}

// WebhookResponse is the API response for a single webhook record.
type WebhookResponse struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Clusters  []string  `json:"clusters"`
	Resources []string  `json:"resources"`
	IsActive  bool      `json:"isActive"`
}

// webhookDispatchSemaphore limits the number of concurrent outbound webhook
// HTTP calls across the entire service to prevent runaway goroutine growth.
const webhookDispatchConcurrency = 10

// WebhookService encapsulates webhook CRUD and event dispatch logic.
type WebhookService struct {
	webhookRepo repository.WebhookRepo
	logger      *zap.Logger
	httpClient  *http.Client
	// semaphore is a buffered channel used to cap concurrent dispatch goroutines.
	semaphore chan struct{}
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(
	webhookRepo repository.WebhookRepo,
	logger *zap.Logger,
) *WebhookService {
	return &WebhookService{
		webhookRepo: webhookRepo,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		semaphore: make(chan struct{}, webhookDispatchConcurrency),
	}
}

// Create creates a new webhook record.
func (s *WebhookService) Create(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error) {
	if err := validateWebhookURL(req.URL); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, err.Error())
	}

	eventsJSON, err := json.Marshal(req.Events)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to encode events")
	}
	clustersJSON, err := json.Marshal(req.Clusters)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to encode clusters")
	}
	resourcesJSON, err := json.Marshal(req.Resources)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to encode resources")
	}

	wh := &model.Webhook{
		Name:      req.Name,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    string(eventsJSON),
		Clusters:  string(clustersJSON),
		Resources: string(resourcesJSON),
		IsActive:  req.IsActive,
	}

	if err := s.webhookRepo.Create(ctx, wh); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to create webhook")
	}
	return toWebhookResponse(wh), nil
}

// GetByID retrieves a single webhook.
func (s *WebhookService) GetByID(ctx context.Context, id uint) (*WebhookResponse, error) {
	wh, err := s.webhookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "webhook not found")
	}
	return toWebhookResponse(wh), nil
}

// Update replaces an existing webhook's fields.
func (s *WebhookService) Update(ctx context.Context, id uint, req *WebhookRequest) (*WebhookResponse, error) {
	if err := validateWebhookURL(req.URL); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, err.Error())
	}

	wh, err := s.webhookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "webhook not found")
	}

	eventsJSON, _ := json.Marshal(req.Events)
	clustersJSON, _ := json.Marshal(req.Clusters)
	resourcesJSON, _ := json.Marshal(req.Resources)

	wh.Name = req.Name
	wh.URL = req.URL
	wh.Events = string(eventsJSON)
	wh.Clusters = string(clustersJSON)
	wh.Resources = string(resourcesJSON)
	wh.IsActive = req.IsActive
	if req.Secret != "" {
		wh.Secret = req.Secret
	}

	if err := s.webhookRepo.Update(ctx, wh); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to update webhook")
	}
	return toWebhookResponse(wh), nil
}

// Delete removes a webhook.
func (s *WebhookService) Delete(ctx context.Context, id uint) error {
	if _, err := s.webhookRepo.GetByID(ctx, id); err != nil {
		return bizerr.New(bizerr.CodeNotFound, "webhook not found")
	}
	if err := s.webhookRepo.Delete(ctx, id); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to delete webhook")
	}
	return nil
}

// List returns all webhooks.
func (s *WebhookService) List(ctx context.Context) ([]WebhookResponse, error) {
	whs, err := s.webhookRepo.List(ctx)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list webhooks")
	}
	result := make([]WebhookResponse, len(whs))
	for i := range whs {
		result[i] = *toWebhookResponse(&whs[i])
	}
	return result, nil
}

// TestWebhook sends a synthetic test payload to the webhook URL immediately.
func (s *WebhookService) TestWebhook(ctx context.Context, webhookID uint) error {
	wh, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "webhook not found")
	}

	testEvent := WebhookEvent{
		EventType: "test",
		Cluster:   "test-cluster",
		Resource:  "pods",
		Name:      "test-pod",
		Namespace: "default",
		Action:    "test",
		Timestamp: time.Now(),
	}

	if err := s.sendPayload(wh, testEvent); err != nil {
		return bizerr.New(bizerr.CodeInternal, fmt.Sprintf("webhook test failed: %v", err))
	}
	return nil
}

// DispatchEvent matches the event against all active webhooks and sends HTTP
// POSTs in the background for each matching webhook. Implements non-blocking
// dispatch so the informer pipeline is never stalled.
func (s *WebhookService) DispatchEvent(event WebhookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	whs, err := s.webhookRepo.ListActive(ctx)
	cancel()
	if err != nil {
		s.logger.Warn("webhook dispatch: failed to list active webhooks", zap.Error(err))
		return
	}

	for i := range whs {
		wh := &whs[i]
		if !s.matchesWebhook(wh, event) {
			continue
		}
		// Acquire the semaphore slot before spawning so we cap total
		// concurrent outbound HTTP calls to webhookDispatchConcurrency.
		s.semaphore <- struct{}{}
		go func(w *model.Webhook, ev WebhookEvent) {
			defer func() { <-s.semaphore }()
			if err := s.sendPayload(w, ev); err != nil {
				s.logger.Warn("webhook dispatch failed",
					zap.Uint("webhookID", w.ID),
					zap.String("url", w.URL),
					zap.Error(err),
				)
			}
		}(wh, event)
	}
}

// OnResourceEvent implements informer.EventListener so WebhookService can be
// registered directly with the informer manager.
func (s *WebhookService) OnResourceEvent(event informer.ResourceEvent) {
	s.DispatchEvent(WebhookEvent{
		EventType: event.Type,
		Cluster:   event.ClusterID,
		Resource:  event.Resource,
		Name:      event.Name,
		Namespace: event.Namespace,
		Action:    event.Type,
		Timestamp: time.Now(),
	})
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// privateIPNets contains all IANA-reserved / private IP ranges that must not
// be reachable from a user-supplied webhook URL.
var privateIPNets = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // loopback
		"10.0.0.0/8",     // RFC-1918
		"172.16.0.0/12",  // RFC-1918
		"192.168.0.0/16", // RFC-1918
		"169.254.0.0/16", // link-local / AWS metadata
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique-local
		"fe80::/10",      // IPv6 link-local
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, _ := net.ParseCIDR(cidr)
		nets = append(nets, ipNet)
	}
	return nets
}()

// validateWebhookURL returns an error when u is not a safe, publicly-routable
// HTTP/HTTPS URL. It rejects private/internal IP addresses to prevent SSRF.
func validateWebhookURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}

	hostname := parsed.Hostname()
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("webhook URL hostname could not be resolved: %w", err)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, ipNet := range privateIPNets {
			if ipNet.Contains(ip) {
				return fmt.Errorf("webhook URL resolves to a private/internal IP address")
			}
		}
	}
	return nil
}

// matchesWebhook checks whether the event satisfies the webhook's filter
// criteria (events, clusters, resources). An empty filter means "match all".
func (s *WebhookService) matchesWebhook(wh *model.Webhook, event WebhookEvent) bool {
	if wh.Events != "" && wh.Events != "[]" && wh.Events != "null" {
		var events []string
		if err := json.Unmarshal([]byte(wh.Events), &events); err == nil && len(events) > 0 {
			if !containsString(events, event.EventType) && !containsString(events, event.Action) {
				return false
			}
		}
	}

	if wh.Clusters != "" && wh.Clusters != "[]" && wh.Clusters != "null" {
		var clusters []string
		if err := json.Unmarshal([]byte(wh.Clusters), &clusters); err == nil && len(clusters) > 0 {
			if !containsString(clusters, event.Cluster) {
				return false
			}
		}
	}

	if wh.Resources != "" && wh.Resources != "[]" && wh.Resources != "null" {
		var resources []string
		if err := json.Unmarshal([]byte(wh.Resources), &resources); err == nil && len(resources) > 0 {
			if !containsString(resources, event.Resource) {
				return false
			}
		}
	}

	return true
}

// sendPayload marshals the event, signs it, and POSTs it to the webhook URL.
func (s *WebhookService) sendPayload(wh *model.Webhook, event WebhookEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KubeVision-Webhook/1.0")

	if wh.Secret != "" {
		sig := computeHMAC(body, wh.Secret)
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	// Drain the body so the underlying TCP connection can be reused by the
	// http.Client's connection pool instead of being discarded.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx response: %d", resp.StatusCode)
	}
	return nil
}

// computeHMAC signs the payload with HMAC-SHA256 using the given secret.
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// containsString checks whether a string slice contains the target value.
func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// toWebhookResponse converts a model.Webhook to its API response shape.
func toWebhookResponse(wh *model.Webhook) *WebhookResponse {
	var events, clusters, resources []string
	_ = json.Unmarshal([]byte(wh.Events), &events)
	_ = json.Unmarshal([]byte(wh.Clusters), &clusters)
	_ = json.Unmarshal([]byte(wh.Resources), &resources)
	if events == nil {
		events = []string{}
	}
	if clusters == nil {
		clusters = []string{}
	}
	if resources == nil {
		resources = []string{}
	}
	return &WebhookResponse{
		ID:        wh.ID,
		CreatedAt: wh.CreatedAt,
		UpdatedAt: wh.UpdatedAt,
		Name:      wh.Name,
		URL:       wh.URL,
		Events:    events,
		Clusters:  clusters,
		Resources: resources,
		IsActive:  wh.IsActive,
	}
}
