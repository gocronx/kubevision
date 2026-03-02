package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Dashboard represents a Grafana dashboard summary.
type Dashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Tags  []string `json:"tags"`
}

// Plugin provides Grafana dashboard integration.
type Plugin struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New() *Plugin {
	return &Plugin{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Name() string        { return "grafana" }
func (p *Plugin) Description() string  { return "Grafana dashboard integration for embedded visualizations" }
func (p *Plugin) Version() string      { return "1.0.0" }
func (p *Plugin) Type() string         { return "dashboard" }

func (p *Plugin) Init(config map[string]string) error {
	u, ok := config["url"]
	if !ok || u == "" {
		return fmt.Errorf("grafana: url is required")
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		return fmt.Errorf("grafana: invalid url: %w", err)
	}
	p.baseURL = u
	p.token = config["token"]
	return nil
}

func (p *Plugin) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/health", nil)
	if err != nil {
		return err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("grafana health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *Plugin) Close() error { return nil }

// ListDashboards returns all Grafana dashboards.
func (p *Plugin) ListDashboards(ctx context.Context) ([]Dashboard, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("grafana not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/search?type=dash-db", nil)
	if err != nil {
		return nil, err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grafana API call failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dashboards []Dashboard
	if err := json.Unmarshal(body, &dashboards); err != nil {
		return nil, err
	}

	// Fix relative URLs.
	for i := range dashboards {
		if dashboards[i].URL != "" && dashboards[i].URL[0] == '/' {
			dashboards[i].URL = p.baseURL + dashboards[i].URL
		}
	}

	return dashboards, nil
}

// EmbedURL generates an embeddable iframe URL for a specific dashboard.
func (p *Plugin) EmbedURL(dashboardUID string, panelID int, from, to string) string {
	if from == "" {
		from = "now-1h"
	}
	if to == "" {
		to = "now"
	}
	return fmt.Sprintf("%s/d-solo/%s?panelId=%d&from=%s&to=%s&theme=light",
		p.baseURL, dashboardUID, panelID, url.QueryEscape(from), url.QueryEscape(to))
}
