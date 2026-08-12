package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Application represents an ArgoCD application summary.
type Application struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Project        string `json:"project"`
	SyncStatus     string `json:"syncStatus"`
	HealthStatus   string `json:"healthStatus"`
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path"`
	TargetRevision string `json:"targetRevision"`
	URL            string `json:"url"`
}

// Plugin provides ArgoCD GitOps integration.
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

func (p *Plugin) Name() string        { return "argocd" }
func (p *Plugin) Description() string { return "ArgoCD GitOps integration for application sync status" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Type() string        { return "gitops" }

func (p *Plugin) Init(config map[string]string) error {
	u, ok := config["url"]
	if !ok || u == "" {
		return fmt.Errorf("argocd: url is required")
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		return fmt.Errorf("argocd: invalid url: %w", err)
	}
	p.baseURL = u
	p.token = config["token"]
	return nil
}

func (p *Plugin) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/applications", nil)
	if err != nil {
		return err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("argocd health check failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("argocd returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *Plugin) Close() error { return nil }

// ListApplications returns all ArgoCD applications.
func (p *Plugin) ListApplications(ctx context.Context) ([]Application, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("argocd not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/applications", nil)
	if err != nil {
		return nil, err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("argocd API call failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Spec struct {
				Project string `json:"project"`
				Source  struct {
					RepoURL        string `json:"repoURL"`
					Path           string `json:"path"`
					TargetRevision string `json:"targetRevision"`
				} `json:"source"`
			} `json:"spec"`
			Status struct {
				Sync struct {
					Status string `json:"status"`
				} `json:"sync"`
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	apps := make([]Application, len(result.Items))
	for i, item := range result.Items {
		apps[i] = Application{
			Name:           item.Metadata.Name,
			Namespace:      item.Metadata.Namespace,
			Project:        item.Spec.Project,
			SyncStatus:     item.Status.Sync.Status,
			HealthStatus:   item.Status.Health.Status,
			RepoURL:        item.Spec.Source.RepoURL,
			Path:           item.Spec.Source.Path,
			TargetRevision: item.Spec.Source.TargetRevision,
			URL:            fmt.Sprintf("%s/applications/%s", p.baseURL, item.Metadata.Name),
		}
	}

	return apps, nil
}

// GetApplication returns details of a specific ArgoCD application.
func (p *Plugin) GetApplication(ctx context.Context, name string) (*Application, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("argocd not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/applications/%s", p.baseURL, url.PathEscape(name)), nil)
	if err != nil {
		return nil, err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("argocd API call failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("application %q not found", name)
	}

	var item struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Project string `json:"project"`
			Source  struct {
				RepoURL        string `json:"repoURL"`
				Path           string `json:"path"`
				TargetRevision string `json:"targetRevision"`
			} `json:"source"`
		} `json:"spec"`
		Status struct {
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, err
	}

	return &Application{
		Name:           item.Metadata.Name,
		Namespace:      item.Metadata.Namespace,
		Project:        item.Spec.Project,
		SyncStatus:     item.Status.Sync.Status,
		HealthStatus:   item.Status.Health.Status,
		RepoURL:        item.Spec.Source.RepoURL,
		Path:           item.Spec.Source.Path,
		TargetRevision: item.Spec.Source.TargetRevision,
		URL:            fmt.Sprintf("%s/applications/%s", p.baseURL, item.Metadata.Name),
	}, nil
}
