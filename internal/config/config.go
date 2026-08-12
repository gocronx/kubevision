package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application.
type Config struct {
	Server     ServerConfig    `yaml:"server"`
	Database   DatabaseConfig  `yaml:"database"`
	Auth       AuthConfig      `yaml:"auth"`
	Kube       KubeConfig      `yaml:"kubernetes"`
	WebSocket  WebSocketConfig `yaml:"websocket"`
	Audit      AuditConfig     `yaml:"audit"`
	OAuth      OAuthConfig     `yaml:"oauth"`
	Plugins    PluginsConfig   `yaml:"plugins"`
	Registry   RegistryConfig  `yaml:"registry"`
	EncryptKey string          `yaml:"encrypt_key"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret       string              `yaml:"jwt_secret"`
	AccessTokenTTL  time.Duration       `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration       `yaml:"refresh_token_ttl"`
	PublicKey       PublicKeyAuthConfig `yaml:"public_key"`
}

// PublicKeyAuthConfig defines the stable relying-party security boundary.
// Changing RPID or Origins can make existing credentials unusable.
type PublicKeyAuthConfig struct {
	Enabled          bool          `yaml:"enabled"`
	RPID             string        `yaml:"rp_id"`
	RPDisplayName    string        `yaml:"rp_display_name"`
	Origins          []string      `yaml:"origins"`
	UserVerification string        `yaml:"user_verification"`
	CounterPolicy    string        `yaml:"counter_policy"`
	ChallengeTTL     time.Duration `yaml:"challenge_ttl"`
}

// KubeConfig holds Kubernetes client settings.
type KubeConfig struct {
	Kubeconfig           string        `yaml:"kubeconfig"`
	InformerResync       time.Duration `yaml:"informer_resync"`
	CRDDiscoveryInterval time.Duration `yaml:"crd_discovery_interval"`
}

// OAuthProvider defines a single OAuth2/OIDC provider configuration.
type OAuthProvider struct {
	Name         string   `yaml:"name"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Issuer       string   `yaml:"issuer"`
	AuthURL      string   `yaml:"auth_url"`
	TokenURL     string   `yaml:"token_url"`
	UserInfoURL  string   `yaml:"userinfo_url"`
	Scopes       []string `yaml:"scopes"`
	RedirectURL  string   `yaml:"redirect_url"`
}

// OAuthConfig holds OAuth/OIDC settings.
type OAuthConfig struct {
	Enabled   bool            `yaml:"enabled"`
	Providers []OAuthProvider `yaml:"providers"`
}

// PluginsConfig holds plugin integration settings.
type PluginsConfig struct {
	Prometheus PluginEndpoint `yaml:"prometheus"`
	Grafana    PluginEndpoint `yaml:"grafana"`
	ArgoCD     PluginEndpoint `yaml:"argocd"`
}

// PluginEndpoint holds connection info for an external service plugin.
type PluginEndpoint struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

// RegistryConfig controls read-only image tag discovery and its outbound boundary.
type RegistryConfig struct {
	AllowedRegistries []string      `yaml:"allowed_registries"`
	AllowedAuthHosts  []string      `yaml:"allowed_auth_hosts"`
	AllowPrivate      bool          `yaml:"allow_private"`
	AllowHTTP         bool          `yaml:"allow_http"`
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`
	HeaderTimeout     time.Duration `yaml:"response_header_timeout"`
	TotalTimeout      time.Duration `yaml:"total_timeout"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
	MaxResponseBytes  int64         `yaml:"max_response_bytes"`
	MaxCacheEntries   int           `yaml:"max_cache_entries"`
	MaxTagsPerPage    int           `yaml:"max_tags_per_page"`
}

// WebSocketConfig holds WebSocket settings.
type WebSocketConfig struct {
	BroadcastBuffer   int           `yaml:"broadcast_buffer"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

// AuditConfig holds audit logging settings.
type AuditConfig struct {
	Enabled       bool          `yaml:"enabled"`
	RetentionDays int           `yaml:"retention_days"`
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	Sync          bool          `yaml:"sync"`
}

// Default returns a Config populated with sane defaults.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "kubevision.db",
		},
		Auth: AuthConfig{
			JWTSecret:       "",
			AccessTokenTTL:  30 * time.Minute,
			RefreshTokenTTL: 12 * time.Hour,
			PublicKey: PublicKeyAuthConfig{
				RPDisplayName:    "KubeVision",
				UserVerification: "required",
				CounterPolicy:    "deny",
				ChallengeTTL:     5 * time.Minute,
			},
		},
		Kube: KubeConfig{
			Kubeconfig:           "",
			InformerResync:       30 * time.Minute,
			CRDDiscoveryInterval: 5 * time.Minute,
		},
		WebSocket: WebSocketConfig{
			BroadcastBuffer:   1024,
			HeartbeatInterval: 30 * time.Second,
		},
		Audit: AuditConfig{
			Enabled:       true,
			RetentionDays: 90,
			BatchSize:     100,
			FlushInterval: 5 * time.Second,
		},
		Registry: RegistryConfig{
			AllowedRegistries: []string{"docker.io", "registry-1.docker.io"},
			AllowedAuthHosts:  []string{"auth.docker.io"},
			ConnectTimeout:    3 * time.Second,
			HeaderTimeout:     5 * time.Second,
			TotalTimeout:      10 * time.Second,
			CacheTTL:          30 * time.Second,
			MaxResponseBytes:  2 << 20,
			MaxCacheEntries:   128,
			MaxTagsPerPage:    100,
		},
		EncryptKey: "",
	}
}

// Load reads configuration from a YAML file (if it exists) and then applies
// environment variable overrides. It returns a fully resolved Config.
func Load(path string) (*Config, error) {
	cfg := Default()

	// Try to load from YAML file.
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read config file: %w", err)
			}
			// File does not exist; continue with defaults.
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config file: %w", err)
			}
		}
	}

	// Apply environment variable overrides.
	applyEnvOverrides(cfg)

	// Load previously persisted secrets (auto-generated on first run).
	loadSecrets(cfg)

	// Auto-generate secrets if not provided and persist them so they survive restarts.
	secretsChanged := false
	if cfg.Auth.JWTSecret == "" {
		secret, err := randomHex(32)
		if err != nil {
			return nil, fmt.Errorf("generate jwt secret: %w", err)
		}
		cfg.Auth.JWTSecret = secret
		secretsChanged = true
	} else if len(cfg.Auth.JWTSecret) < 32 {
		// The configured secret is too short to be safe; replace it with a
		// freshly generated one and persist it so the change survives restarts.
		secret, err := randomHex(32)
		if err != nil {
			return nil, fmt.Errorf("generate jwt secret: %w", err)
		}
		cfg.Auth.JWTSecret = secret
		secretsChanged = true
	}
	if cfg.EncryptKey == "" {
		key, err := randomHex(32)
		if err != nil {
			return nil, fmt.Errorf("generate encrypt key: %w", err)
		}
		cfg.EncryptKey = key
		secretsChanged = true
	}

	// Persist auto-generated secrets to a file so encrypted data can be
	// decrypted after application restarts.
	if secretsChanged {
		if err := persistSecrets(cfg); err != nil {
			return nil, fmt.Errorf("persist secrets: %w", err)
		}
	}

	return cfg, nil
}

const secretsFile = ".kubevision-secrets.yaml"

// secretsData is the minimal struct written to the secrets file.
type secretsData struct {
	Auth struct {
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"auth"`
	EncryptKey string `yaml:"encrypt_key"`
}

func persistSecrets(cfg *Config) error {
	// Try to load existing secrets first so we don't overwrite user-provided values.
	existing := secretsData{}
	if data, err := os.ReadFile(secretsFile); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	if existing.Auth.JWTSecret == "" {
		existing.Auth.JWTSecret = cfg.Auth.JWTSecret
	}
	if existing.EncryptKey == "" {
		existing.EncryptKey = cfg.EncryptKey
	}

	out, err := yaml.Marshal(&existing)
	if err != nil {
		return err
	}
	return os.WriteFile(secretsFile, out, 0600)
}

func loadSecrets(cfg *Config) {
	data, err := os.ReadFile(secretsFile)
	if err != nil {
		return
	}
	var s secretsData
	if err := yaml.Unmarshal(data, &s); err != nil {
		return
	}
	if cfg.Auth.JWTSecret == "" && s.Auth.JWTSecret != "" {
		cfg.Auth.JWTSecret = s.Auth.JWTSecret
	}
	if cfg.EncryptKey == "" && s.EncryptKey != "" {
		cfg.EncryptKey = s.EncryptKey
	}
}

// applyEnvOverrides reads well-known environment variables and overrides config
// values when they are set.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("KUBEVISION_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("KUBEVISION_DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("KUBEVISION_DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("KUBEVISION_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("KUBEVISION_ACCESS_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.AccessTokenTTL = d
		}
	}
	if v := os.Getenv("KUBEVISION_REFRESH_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.RefreshTokenTTL = d
		}
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_ENABLED"); v != "" {
		cfg.Auth.PublicKey.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_RP_ID"); v != "" {
		cfg.Auth.PublicKey.RPID = strings.TrimSpace(v)
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_RP_NAME"); v != "" {
		cfg.Auth.PublicKey.RPDisplayName = strings.TrimSpace(v)
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_ORIGINS"); v != "" {
		cfg.Auth.PublicKey.Origins = splitCSV(v)
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_UV"); v != "" {
		cfg.Auth.PublicKey.UserVerification = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_COUNTER_POLICY"); v != "" {
		cfg.Auth.PublicKey.CounterPolicy = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("KUBEVISION_PUBLIC_KEY_CHALLENGE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.PublicKey.ChallengeTTL = d
		}
	}
	if v := os.Getenv("KUBECONFIG"); v != "" {
		cfg.Kube.Kubeconfig = v
	}
	if v := os.Getenv("KUBEVISION_KUBECONFIG"); v != "" {
		cfg.Kube.Kubeconfig = v
	}
	if v := os.Getenv("KUBEVISION_INFORMER_RESYNC"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Kube.InformerResync = d
		}
	}
	if v := os.Getenv("KUBEVISION_WS_BROADCAST_BUFFER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WebSocket.BroadcastBuffer = n
		}
	}
	if v := os.Getenv("KUBEVISION_WS_HEARTBEAT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.WebSocket.HeartbeatInterval = d
		}
	}
	if v := os.Getenv("KUBEVISION_AUDIT_ENABLED"); v != "" {
		cfg.Audit.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("KUBEVISION_AUDIT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Audit.RetentionDays = n
		}
	}
	if v := os.Getenv("KUBEVISION_ENCRYPT_KEY"); v != "" {
		cfg.EncryptKey = v
	}
	if v := os.Getenv("KUBEVISION_CRD_DISCOVERY_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Kube.CRDDiscoveryInterval = d
		}
	}
	if v := os.Getenv("KUBEVISION_OAUTH_ENABLED"); v != "" {
		cfg.OAuth.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("KUBEVISION_REGISTRY_ALLOWED"); v != "" {
		cfg.Registry.AllowedRegistries = splitCSV(v)
	}
	if v := os.Getenv("KUBEVISION_REGISTRY_AUTH_HOSTS"); v != "" {
		cfg.Registry.AllowedAuthHosts = splitCSV(v)
	}
	if v := os.Getenv("KUBEVISION_REGISTRY_ALLOW_PRIVATE"); v != "" {
		cfg.Registry.AllowPrivate = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("KUBEVISION_REGISTRY_ALLOW_HTTP"); v != "" {
		cfg.Registry.AllowHTTP = strings.EqualFold(v, "true") || v == "1"
	}
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// randomHex returns a random hex string of n bytes (2n hex chars).
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
