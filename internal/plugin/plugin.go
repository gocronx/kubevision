package plugin

import "context"

// Plugin is the interface all KubeVision plugins must implement.
type Plugin interface {
	// Name returns the unique identifier for this plugin.
	Name() string
	// Description returns a human-readable description.
	Description() string
	// Version returns the plugin version string.
	Version() string
	// Type returns the plugin category: "monitoring", "gitops", or "dashboard".
	Type() string
	// Init initializes the plugin with the given configuration map.
	Init(config map[string]string) error
	// HealthCheck verifies that the plugin's external dependency is reachable.
	HealthCheck(ctx context.Context) error
	// Close releases any resources held by the plugin.
	Close() error
}
