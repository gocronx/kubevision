package repository

import (
	"context"
	"time"

	"github.com/kubevision/kubevision/internal/model"
)

// --------------------------------------------------------------------------
// Kubernetes resource interfaces
// --------------------------------------------------------------------------

// K8sResourceRepo abstracts access to Kubernetes resources (via informer cache
// or direct API calls).
type K8sResourceRepo interface {
	// List returns resources of the given kind in the specified namespace.
	// An empty namespace means cluster-scoped or all-namespaces.
	List(ctx context.Context, clusterID, kind, namespace string, opts ListOptions) (*ResourceList, error)

	// Get retrieves a single resource by name.
	Get(ctx context.Context, clusterID, kind, namespace, name string) (*Resource, error)

	// Create creates a new resource from an unstructured object.
	Create(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*Resource, error)

	// Update replaces an existing resource with the provided unstructured object.
	Update(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*Resource, error)

	// Delete removes a resource.
	Delete(ctx context.Context, clusterID, kind, namespace, name string) error

	// Patch applies a strategic merge patch to a resource.
	Patch(ctx context.Context, clusterID, kind, namespace, name string, patchData []byte) (*Resource, error)

	// DryRunCreate performs a server-side dry-run create and returns what the
	// API server would create (with defaults filled in) without persisting it.
	DryRunCreate(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*Resource, error)

	// DryRunUpdate performs a server-side dry-run update. It returns the live
	// current resource alongside the dry-run result so the caller can diff them.
	DryRunUpdate(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*Resource, *Resource, error)
}

// ListOptions holds optional query parameters for listing resources.
type ListOptions struct {
	LabelSelector string `form:"labelSelector"`
	FieldSelector string `form:"fieldSelector"`
	Limit         int64  `form:"limit"`
	Continue      string `form:"continue"`
}

// ResourceList holds a list of resources with pagination metadata.
type ResourceList struct {
	Items    []Resource `json:"items"`
	Total    int64      `json:"total"`
	Continue string     `json:"continue,omitempty"`
	Stale    bool       `json:"stale,omitempty"`
}

// Resource is a generic holder for a Kubernetes resource.
type Resource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace,omitempty"`
	Raw        map[string]interface{} `json:"raw,omitempty"`
}

// ResourceRegistry manages which Kubernetes resource types are known and
// watchable.
type ResourceRegistry interface {
	// Register adds a resource type to the registry.
	Register(group, version, kind string, namespaced bool)

	// Lookup returns resource metadata by kind.
	Lookup(kind string) (*ResourceMeta, bool)

	// All returns all registered resource metadata.
	All() []ResourceMeta
}

// ResourceMeta describes a registered Kubernetes resource type.
type ResourceMeta struct {
	Group      string `json:"group"`
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Namespaced bool   `json:"namespaced"`
}

// --------------------------------------------------------------------------
// Cluster management interface
// --------------------------------------------------------------------------

// ClusterManager handles the lifecycle of Kubernetes cluster connections.
type ClusterManager interface {
	// Add registers a new cluster.
	Add(ctx context.Context, cluster *ClusterInfo) error

	// Remove deregisters a cluster and cleans up resources.
	Remove(ctx context.Context, clusterID string) error

	// Get returns information about a specific cluster.
	Get(ctx context.Context, clusterID string) (*ClusterInfo, error)

	// List returns all registered clusters.
	List(ctx context.Context) ([]*ClusterInfo, error)

	// HealthCheck probes the cluster API server.
	HealthCheck(ctx context.Context, clusterID string) (*ClusterHealth, error)
}

// ClusterInfo holds metadata about a registered Kubernetes cluster.
type ClusterInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	APIServer  string `json:"apiServer"`
	Kubeconfig string `json:"-"` // never expose in API responses
}

// ClusterHealth represents the health status of a cluster.
type ClusterHealth struct {
	ClusterID string `json:"clusterId"`
	Healthy   bool   `json:"healthy"`
	Message   string `json:"message,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}

// --------------------------------------------------------------------------
// Persistent store interfaces (backed by GORM / database)
// --------------------------------------------------------------------------

// UserRepo handles user CRUD operations.
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]model.User, error)
}

// ClusterRepo handles cluster record CRUD in the database.
type ClusterRepo interface {
	Create(ctx context.Context, cluster *model.Cluster) error
	GetByID(ctx context.Context, id uint) (*model.Cluster, error)
	GetByName(ctx context.Context, name string) (*model.Cluster, error)
	Update(ctx context.Context, cluster *model.Cluster) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]model.Cluster, error)
}

// RoleRepo handles RBAC role CRUD operations.
type RoleRepo interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id uint) (*model.Role, error)
	GetByName(ctx context.Context, name string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]model.Role, error)
}

// AuditRepo handles audit log persistence.
type AuditRepo interface {
	// BatchCreate inserts multiple audit log entries at once.
	BatchCreate(ctx context.Context, logs []model.AuditLog) error

	// List returns paginated audit logs with optional filters.
	List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error)

	// Purge deletes entries older than the given time.
	Purge(ctx context.Context, before time.Time) (int64, error)
}

// AuditFilter defines optional query parameters for listing audit logs.
type AuditFilter struct {
	UserID  uint
	Action  string
	Cluster string
	Since   time.Time
	Offset  int
	Limit   int
}

// APIKeyRepo handles API key CRUD operations.
type APIKeyRepo interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	ListByUser(ctx context.Context, userID uint) ([]model.APIKey, error)
	Delete(ctx context.Context, id uint) error
}

// TemplateRepo handles resource template CRUD operations.
type TemplateRepo interface {
	Create(ctx context.Context, tmpl *model.Template) error
	GetByID(ctx context.Context, id uint) (*model.Template, error)
	Update(ctx context.Context, tmpl *model.Template) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, category string) ([]model.Template, error)
}

// SettingRepo handles system settings persistence.
type SettingRepo interface {
	Get(ctx context.Context, key string) (*model.Setting, error)
	Set(ctx context.Context, setting *model.Setting) error
	List(ctx context.Context, category string) ([]model.Setting, error)
	Delete(ctx context.Context, key string) error
}

// WebhookRepo handles webhook CRUD operations.
type WebhookRepo interface {
	Create(ctx context.Context, webhook *model.Webhook) error
	GetByID(ctx context.Context, id uint) (*model.Webhook, error)
	Update(ctx context.Context, webhook *model.Webhook) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]model.Webhook, error)
	ListActive(ctx context.Context) ([]model.Webhook, error)
}

// FavoriteRepo handles user favorites CRUD operations.
type FavoriteRepo interface {
	Create(ctx context.Context, fav *model.Favorite) error
	Delete(ctx context.Context, id uint) error
	ListByUser(ctx context.Context, userID uint) ([]model.Favorite, error)
	UpdateSortOrder(ctx context.Context, id uint, sortOrder int) error
	GetByUserAndResource(ctx context.Context, userID uint, clusterID, resourceType, resourceName, namespace string) (*model.Favorite, error)
}

// TerminalSessionRepo handles terminal session recording persistence.
type TerminalSessionRepo interface {
	Create(ctx context.Context, session *model.TerminalSession) error
	GetByID(ctx context.Context, id uint) (*model.TerminalSession, error)
	ListByUser(ctx context.Context, userID uint) ([]model.TerminalSession, error)
	PurgeExpired(ctx context.Context) (int64, error)
}
