package packages

import (
	"context"
	"time"
)

const (
	PermissionRead     = "package-releases:read"
	PermissionRollback = "package-releases:rollback"
	PermissionRemove   = "package-releases:remove"
)

type Actor struct {
	UserID   uint
	Username string
	Role     string
	ClientIP string
}

type ResourceRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type Release struct {
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace"`
	Revision     int                    `json:"revision"`
	Status       string                 `json:"status"`
	Chart        string                 `json:"chart"`
	ChartVersion string                 `json:"chartVersion"`
	AppVersion   string                 `json:"appVersion"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Notes        string                 `json:"notes,omitempty"`
	Values       map[string]interface{} `json:"values,omitempty"`
	Resources    []ResourceRef          `json:"resources,omitempty"`
}

type ListOptions struct {
	Namespace string
	State     string
	Label     string
	Limit     int
}

type RollbackOptions struct {
	Revision int           `json:"revision" binding:"required,min=1"`
	Wait     bool          `json:"wait"`
	Atomic   bool          `json:"atomic"`
	Timeout  time.Duration `json:"-"`
}

type RemoveOptions struct {
	Confirmation string        `json:"confirmation" binding:"required"`
	KeepHistory  bool          `json:"keepHistory"`
	Wait         bool          `json:"wait"`
	Timeout      time.Duration `json:"-"`
}

type Adapter interface {
	List(context.Context, string, ListOptions) ([]Release, error)
	Get(context.Context, string, string, string, int) (*Release, error)
	History(context.Context, string, string, string) ([]Release, error)
	Rollback(context.Context, string, string, string, RollbackOptions) error
	Remove(context.Context, string, string, string, RemoveOptions) error
}

type Authorizer interface {
	Allowed(context.Context, Actor, string, string, string) bool
}

type AuditEvent struct {
	Actor     Actor
	Action    string
	Cluster   string
	Namespace string
	Release   string
	Revision  int
	Options   string
	Outcome   string
	Duration  time.Duration
}

type Auditor interface{ RecordPackageAudit(AuditEvent) }
