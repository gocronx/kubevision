package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Plugin provides Prometheus metrics integration.
type Plugin struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Plugin {
	return &Plugin{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Name() string        { return "prometheus" }
func (p *Plugin) Description() string  { return "Prometheus metrics integration for pods and nodes" }
func (p *Plugin) Version() string      { return "1.0.0" }
func (p *Plugin) Type() string         { return "monitoring" }

func (p *Plugin) Init(config map[string]string) error {
	u, ok := config["url"]
	if !ok || u == "" {
		return fmt.Errorf("prometheus: url is required")
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		return fmt.Errorf("prometheus: invalid url: %w", err)
	}
	p.baseURL = u
	return nil
}

func (p *Plugin) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/-/healthy", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *Plugin) Close() error { return nil }

// QueryResult holds the result of a Prometheus instant query.
type QueryResult struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// Query executes a PromQL query against the Prometheus API.
func (p *Plugin) Query(ctx context.Context, query string) (*QueryResult, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("prometheus not configured")
	}

	u := fmt.Sprintf("%s/api/v1/query?query=%s", p.baseURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// QueryRange executes a range query against the Prometheus API.
func (p *Plugin) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("prometheus not configured")
	}

	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
		p.baseURL, url.QueryEscape(query), start.Unix(), end.Unix(), step.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// escapePromQLLabelValue escapes special characters inside a PromQL label
// value string so that user-controlled input cannot break out of the
// double-quoted label selector syntax.
//
// Characters that must be escaped inside double-quoted PromQL strings:
//   - backslash  (\) — must be first to avoid double-escaping
//   - double quote (") — closes the label value prematurely
//   - newline (\n)    — terminates the label value
func escapePromQLLabelValue(s string) string {
	// Order matters: escape backslashes first.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// PodCPU returns the CPU usage rate for a specific pod.
func (p *Plugin) PodCPU(ctx context.Context, namespace, pod string) (*QueryResult, error) {
	query := fmt.Sprintf(
		`rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!=""}[5m])`,
		escapePromQLLabelValue(namespace),
		escapePromQLLabelValue(pod),
	)
	return p.Query(ctx, query)
}

// PodMemory returns the memory usage for a specific pod.
func (p *Plugin) PodMemory(ctx context.Context, namespace, pod string) (*QueryResult, error) {
	query := fmt.Sprintf(
		`container_memory_working_set_bytes{namespace="%s",pod="%s",container!=""}`,
		escapePromQLLabelValue(namespace),
		escapePromQLLabelValue(pod),
	)
	return p.Query(ctx, query)
}
