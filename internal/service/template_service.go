package service

import (
	"context"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// TemplateResponse is the API response for a single template.
type TemplateResponse struct {
	ID           uint      `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	ResourceType string    `json:"resourceType"`
	Content      string    `json:"content"`
	IsBuiltin    bool      `json:"isBuiltin"`
}

// CreateTemplateRequest is the request body for creating a template.
type CreateTemplateRequest struct {
	Name         string `json:"name"         binding:"required"`
	Category     string `json:"category"     binding:"required"`
	ResourceType string `json:"resourceType" binding:"required"`
	Content      string `json:"content"      binding:"required"`
}

// TemplateService encapsulates business logic for resource templates.
type TemplateService struct {
	repo repository.TemplateRepo
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(repo repository.TemplateRepo) *TemplateService {
	return &TemplateService{repo: repo}
}

// List returns all templates, optionally filtered by category.
func (s *TemplateService) List(ctx context.Context, category string) ([]TemplateResponse, error) {
	templates, err := s.repo.List(ctx, category)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list templates")
	}
	result := make([]TemplateResponse, len(templates))
	for i := range templates {
		result[i] = toTemplateResponse(&templates[i])
	}
	return result, nil
}

// Get returns a single template by ID.
func (s *TemplateService) Get(ctx context.Context, id uint) (*TemplateResponse, error) {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to get template")
	}
	if tmpl == nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "template not found")
	}
	resp := toTemplateResponse(tmpl)
	return &resp, nil
}

// Create adds a new user-defined template.
func (s *TemplateService) Create(ctx context.Context, req *CreateTemplateRequest) (*TemplateResponse, error) {
	tmpl := &model.Template{
		Name:         req.Name,
		Category:     req.Category,
		ResourceType: req.ResourceType,
		Content:      req.Content,
		IsBuiltin:    false,
	}
	if err := s.repo.Create(ctx, tmpl); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to create template")
	}
	resp := toTemplateResponse(tmpl)
	return &resp, nil
}

// Delete removes a template. Built-in templates cannot be deleted.
func (s *TemplateService) Delete(ctx context.Context, id uint) error {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to get template")
	}
	if tmpl == nil {
		return bizerr.New(bizerr.CodeNotFound, "template not found")
	}
	if tmpl.IsBuiltin {
		return bizerr.New(bizerr.CodeForbidden, "cannot delete built-in template")
	}
	return s.repo.Delete(ctx, id)
}

// SeedBuiltinTemplates inserts built-in templates if they don't already exist.
func (s *TemplateService) SeedBuiltinTemplates(ctx context.Context) error {
	existing, err := s.repo.List(ctx, "")
	if err != nil {
		return err
	}
	// Index existing by name to avoid duplicates.
	nameSet := make(map[string]bool, len(existing))
	for _, t := range existing {
		nameSet[t.Name] = true
	}

	for _, tmpl := range builtinTemplates {
		if nameSet[tmpl.Name] {
			continue
		}
		if err := s.repo.Create(ctx, &tmpl); err != nil {
			return err
		}
	}
	return nil
}

func toTemplateResponse(t *model.Template) TemplateResponse {
	return TemplateResponse{
		ID:           t.ID,
		CreatedAt:    t.CreatedAt,
		Name:         t.Name,
		Category:     t.Category,
		ResourceType: t.ResourceType,
		Content:      t.Content,
		IsBuiltin:    t.IsBuiltin,
	}
}

// builtinTemplates defines the built-in resource templates shipped with KubeVision.
var builtinTemplates = []model.Template{
	{
		Name:         "Web Application",
		Category:     "workloads",
		ResourceType: "deployments",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "my-web-app",
    "namespace": "default",
    "labels": {
      "app": "my-web-app"
    }
  },
  "spec": {
    "replicas": 2,
    "selector": {
      "matchLabels": {
        "app": "my-web-app"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "my-web-app"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "web",
            "image": "nginx:latest",
            "ports": [
              {
                "containerPort": 80
              }
            ],
            "resources": {
              "requests": {
                "cpu": "100m",
                "memory": "128Mi"
              },
              "limits": {
                "cpu": "500m",
                "memory": "256Mi"
              }
            }
          }
        ]
      }
    }
  }
}`,
	},
	{
		Name:         "Backend API Service",
		Category:     "workloads",
		ResourceType: "deployments",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "my-api",
    "namespace": "default",
    "labels": {
      "app": "my-api"
    }
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "my-api"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "my-api"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "api",
            "image": "my-registry/my-api:latest",
            "ports": [
              {
                "containerPort": 8080
              }
            ],
            "env": [
              {
                "name": "PORT",
                "value": "8080"
              }
            ],
            "readinessProbe": {
              "httpGet": {
                "path": "/healthz",
                "port": 8080
              },
              "initialDelaySeconds": 5,
              "periodSeconds": 10
            },
            "livenessProbe": {
              "httpGet": {
                "path": "/healthz",
                "port": 8080
              },
              "initialDelaySeconds": 15,
              "periodSeconds": 20
            },
            "resources": {
              "requests": {
                "cpu": "200m",
                "memory": "256Mi"
              },
              "limits": {
                "cpu": "1",
                "memory": "512Mi"
              }
            }
          }
        ]
      }
    }
  }
}`,
	},
	{
		Name:         "ClusterIP Service",
		Category:     "network",
		ResourceType: "services",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "my-service",
    "namespace": "default"
  },
  "spec": {
    "type": "ClusterIP",
    "selector": {
      "app": "my-web-app"
    },
    "ports": [
      {
        "name": "http",
        "port": 80,
        "targetPort": 80,
        "protocol": "TCP"
      }
    ]
  }
}`,
	},
	{
		Name:         "LoadBalancer Service",
		Category:     "network",
		ResourceType: "services",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "my-loadbalancer",
    "namespace": "default"
  },
  "spec": {
    "type": "LoadBalancer",
    "selector": {
      "app": "my-web-app"
    },
    "ports": [
      {
        "name": "http",
        "port": 80,
        "targetPort": 80,
        "protocol": "TCP"
      },
      {
        "name": "https",
        "port": 443,
        "targetPort": 443,
        "protocol": "TCP"
      }
    ]
  }
}`,
	},
	{
		Name:         "NodePort Service",
		Category:     "network",
		ResourceType: "services",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "my-nodeport",
    "namespace": "default"
  },
  "spec": {
    "type": "NodePort",
    "selector": {
      "app": "my-web-app"
    },
    "ports": [
      {
        "name": "http",
        "port": 80,
        "targetPort": 80,
        "nodePort": 30080,
        "protocol": "TCP"
      }
    ]
  }
}`,
	},
	{
		Name:         "CronJob",
		Category:     "workloads",
		ResourceType: "cronjobs",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "batch/v1",
  "kind": "CronJob",
  "metadata": {
    "name": "my-cronjob",
    "namespace": "default"
  },
  "spec": {
    "schedule": "0 */6 * * *",
    "jobTemplate": {
      "spec": {
        "template": {
          "spec": {
            "containers": [
              {
                "name": "task",
                "image": "busybox:latest",
                "command": ["/bin/sh", "-c", "echo Hello from CronJob"]
              }
            ],
            "restartPolicy": "OnFailure"
          }
        }
      }
    }
  }
}`,
	},
	{
		Name:         "ConfigMap",
		Category:     "config",
		ResourceType: "configmaps",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
    "name": "my-config",
    "namespace": "default"
  },
  "data": {
    "APP_ENV": "production",
    "APP_PORT": "8080",
    "config.yaml": "key: value\nother: setting"
  }
}`,
	},
	{
		Name:         "Ingress (HTTP)",
		Category:     "network",
		ResourceType: "ingresses",
		IsBuiltin:    true,
		Content: `{
  "apiVersion": "networking.k8s.io/v1",
  "kind": "Ingress",
  "metadata": {
    "name": "my-ingress",
    "namespace": "default",
    "annotations": {
      "nginx.ingress.kubernetes.io/rewrite-target": "/"
    }
  },
  "spec": {
    "ingressClassName": "nginx",
    "rules": [
      {
        "host": "myapp.example.com",
        "http": {
          "paths": [
            {
              "path": "/",
              "pathType": "Prefix",
              "backend": {
                "service": {
                  "name": "my-service",
                  "port": {
                    "number": 80
                  }
                }
              }
            }
          ]
        }
      }
    ]
  }
}`,
	},
}
