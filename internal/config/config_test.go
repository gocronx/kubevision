package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	t.Run("ServerDefaults", func(t *testing.T) {
		if cfg.Server.Port != 8080 {
			t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
		}
	})

	t.Run("DatabaseDefaults", func(t *testing.T) {
		if cfg.Database.Driver != "sqlite" {
			t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "sqlite")
		}
		if cfg.Database.DSN != "kubevision.db" {
			t.Errorf("Database.DSN = %q, want %q", cfg.Database.DSN, "kubevision.db")
		}
	})

	t.Run("AuthDefaults", func(t *testing.T) {
		if cfg.Auth.JWTSecret != "" {
			t.Errorf("Auth.JWTSecret = %q, want empty string", cfg.Auth.JWTSecret)
		}
		if cfg.Auth.AccessTokenTTL != 30*time.Minute {
			t.Errorf("Auth.AccessTokenTTL = %v, want %v", cfg.Auth.AccessTokenTTL, 30*time.Minute)
		}
		if cfg.Auth.RefreshTokenTTL != 12*time.Hour {
			t.Errorf("Auth.RefreshTokenTTL = %v, want %v", cfg.Auth.RefreshTokenTTL, 12*time.Hour)
		}
	})

	t.Run("KubeDefaults", func(t *testing.T) {
		if cfg.Kube.Kubeconfig != "" {
			t.Errorf("Kube.Kubeconfig = %q, want empty string", cfg.Kube.Kubeconfig)
		}
		if cfg.Kube.InformerResync != 30*time.Minute {
			t.Errorf("Kube.InformerResync = %v, want %v", cfg.Kube.InformerResync, 30*time.Minute)
		}
	})

	t.Run("WebSocketDefaults", func(t *testing.T) {
		if cfg.WebSocket.BroadcastBuffer != 1024 {
			t.Errorf("WebSocket.BroadcastBuffer = %d, want 1024", cfg.WebSocket.BroadcastBuffer)
		}
		if cfg.WebSocket.HeartbeatInterval != 30*time.Second {
			t.Errorf("WebSocket.HeartbeatInterval = %v, want %v", cfg.WebSocket.HeartbeatInterval, 30*time.Second)
		}
	})

	t.Run("AuditDefaults", func(t *testing.T) {
		if !cfg.Audit.Enabled {
			t.Error("Audit.Enabled = false, want true")
		}
		if cfg.Audit.RetentionDays != 90 {
			t.Errorf("Audit.RetentionDays = %d, want 90", cfg.Audit.RetentionDays)
		}
		if cfg.Audit.BatchSize != 100 {
			t.Errorf("Audit.BatchSize = %d, want 100", cfg.Audit.BatchSize)
		}
		if cfg.Audit.FlushInterval != 5*time.Second {
			t.Errorf("Audit.FlushInterval = %v, want %v", cfg.Audit.FlushInterval, 5*time.Second)
		}
	})

	t.Run("EncryptKeyDefault", func(t *testing.T) {
		if cfg.EncryptKey != "" {
			t.Errorf("EncryptKey = %q, want empty string", cfg.EncryptKey)
		}
	})
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/tmp/this-file-does-not-exist-kubevision-test.yaml")
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	// Should still get defaults (except auto-generated secrets).
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "sqlite")
	}

	// Auto-generated secrets should be non-empty.
	if cfg.Auth.JWTSecret == "" {
		t.Error("Auth.JWTSecret should be auto-generated, got empty string")
	}
	if cfg.EncryptKey == "" {
		t.Error("EncryptKey should be auto-generated, got empty string")
	}
}

func TestLoadEmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Auth.JWTSecret == "" {
		t.Error("Auth.JWTSecret should be auto-generated, got empty string")
	}
}

func TestLoadValidYAML(t *testing.T) {
	yamlContent := `
server:
  port: 9090
database:
  driver: postgres
  dsn: "host=localhost user=test dbname=kv"
auth:
  jwt_secret: "yaml-jwt-secret-with-at-least-32-characters"
  access_token_ttl: 30m
  refresh_token_ttl: 72h
kubernetes:
  kubeconfig: "/home/user/.kube/config"
  informer_resync: 1h
websocket:
  broadcast_buffer: 2048
  heartbeat_interval: 15s
audit:
  enabled: false
  retention_days: 30
  batch_size: 50
  flush_interval: 10s
encrypt_key: "my-encrypt-key"
`
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Server.Port", cfg.Server.Port, 9090},
		{"Database.Driver", cfg.Database.Driver, "postgres"},
		{"Database.DSN", cfg.Database.DSN, "host=localhost user=test dbname=kv"},
		{"Auth.JWTSecret", cfg.Auth.JWTSecret, "yaml-jwt-secret-with-at-least-32-characters"},
		{"Auth.AccessTokenTTL", cfg.Auth.AccessTokenTTL, 30 * time.Minute},
		{"Auth.RefreshTokenTTL", cfg.Auth.RefreshTokenTTL, 72 * time.Hour},
		{"Kube.Kubeconfig", cfg.Kube.Kubeconfig, "/home/user/.kube/config"},
		{"Kube.InformerResync", cfg.Kube.InformerResync, 1 * time.Hour},
		{"WebSocket.BroadcastBuffer", cfg.WebSocket.BroadcastBuffer, 2048},
		{"WebSocket.HeartbeatInterval", cfg.WebSocket.HeartbeatInterval, 15 * time.Second},
		{"Audit.Enabled", cfg.Audit.Enabled, false},
		{"Audit.RetentionDays", cfg.Audit.RetentionDays, 30},
		{"Audit.BatchSize", cfg.Audit.BatchSize, 50},
		{"Audit.FlushInterval", cfg.Audit.FlushInterval, 10 * time.Second},
		{"EncryptKey", cfg.EncryptKey, "my-encrypt-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(cfgFile, []byte("{{invalid yaml}}"), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("Load should return error for invalid YAML, got nil")
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "ServerPort",
			envVars: map[string]string{
				"KUBEVISION_SERVER_PORT": "3000",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 3000 {
					t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
				}
			},
		},
		{
			name: "DatabaseDriverAndDSN",
			envVars: map[string]string{
				"KUBEVISION_DB_DRIVER":             "postgres",
				"KUBEVISION_DB_DSN":                "postgres://localhost/kubevision",
				"KUBEVISION_DB_MAX_OPEN_CONNS":     "40",
				"KUBEVISION_DB_MAX_IDLE_CONNS":     "8",
				"KUBEVISION_DB_CONN_MAX_LIFETIME":  "45m",
				"KUBEVISION_DB_CONN_MAX_IDLE_TIME": "10m",
				"KUBEVISION_DB_PING_TIMEOUT":       "3s",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Database.Driver != "postgres" {
					t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, "postgres")
				}
				if cfg.Database.DSN != "postgres://localhost/kubevision" {
					t.Errorf("Database.DSN = %q", cfg.Database.DSN)
				}
				if cfg.Database.MaxOpenConns != 40 || cfg.Database.MaxIdleConns != 8 {
					t.Errorf("Database pool = %d/%d", cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
				}
				if cfg.Database.ConnMaxLifetime != 45*time.Minute || cfg.Database.ConnMaxIdleTime != 10*time.Minute || cfg.Database.PingTimeout != 3*time.Second {
					t.Errorf("Database durations were not overridden: %+v", cfg.Database)
				}
			},
		},
		{
			name: "JWTSecret",
			envVars: map[string]string{
				"KUBEVISION_JWT_SECRET": "env-jwt-secret-with-at-least-32-characters",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Auth.JWTSecret != "env-jwt-secret-with-at-least-32-characters" {
					t.Errorf("Auth.JWTSecret = %q, want %q", cfg.Auth.JWTSecret, "env-jwt-secret-with-at-least-32-characters")
				}
			},
		},
		{
			name: "TokenTTLs",
			envVars: map[string]string{
				"KUBEVISION_ACCESS_TOKEN_TTL":  "1h",
				"KUBEVISION_REFRESH_TOKEN_TTL": "24h",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Auth.AccessTokenTTL != 1*time.Hour {
					t.Errorf("Auth.AccessTokenTTL = %v, want %v", cfg.Auth.AccessTokenTTL, 1*time.Hour)
				}
				if cfg.Auth.RefreshTokenTTL != 24*time.Hour {
					t.Errorf("Auth.RefreshTokenTTL = %v, want %v", cfg.Auth.RefreshTokenTTL, 24*time.Hour)
				}
			},
		},
		{
			name: "KubeconfigEnv",
			envVars: map[string]string{
				"KUBECONFIG": "/env/kubeconfig",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Kube.Kubeconfig != "/env/kubeconfig" {
					t.Errorf("Kube.Kubeconfig = %q, want %q", cfg.Kube.Kubeconfig, "/env/kubeconfig")
				}
			},
		},
		{
			name: "KubevisionKubeconfigOverridesKubeconfig",
			envVars: map[string]string{
				"KUBECONFIG":            "/generic/kubeconfig",
				"KUBEVISION_KUBECONFIG": "/specific/kubeconfig",
			},
			check: func(t *testing.T, cfg *Config) {
				// KUBEVISION_KUBECONFIG is checked after KUBECONFIG, so it wins.
				if cfg.Kube.Kubeconfig != "/specific/kubeconfig" {
					t.Errorf("Kube.Kubeconfig = %q, want %q", cfg.Kube.Kubeconfig, "/specific/kubeconfig")
				}
			},
		},
		{
			name: "InformerResync",
			envVars: map[string]string{
				"KUBEVISION_INFORMER_RESYNC": "2h",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Kube.InformerResync != 2*time.Hour {
					t.Errorf("Kube.InformerResync = %v, want %v", cfg.Kube.InformerResync, 2*time.Hour)
				}
			},
		},
		{
			name: "WebSocketSettings",
			envVars: map[string]string{
				"KUBEVISION_WS_BROADCAST_BUFFER":   "512",
				"KUBEVISION_WS_HEARTBEAT_INTERVAL": "10s",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.WebSocket.BroadcastBuffer != 512 {
					t.Errorf("WebSocket.BroadcastBuffer = %d, want 512", cfg.WebSocket.BroadcastBuffer)
				}
				if cfg.WebSocket.HeartbeatInterval != 10*time.Second {
					t.Errorf("WebSocket.HeartbeatInterval = %v, want %v", cfg.WebSocket.HeartbeatInterval, 10*time.Second)
				}
			},
		},
		{
			name: "AuditEnabledTrue",
			envVars: map[string]string{
				"KUBEVISION_AUDIT_ENABLED": "true",
			},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Audit.Enabled {
					t.Error("Audit.Enabled = false, want true")
				}
			},
		},
		{
			name: "AuditEnabledFalse",
			envVars: map[string]string{
				"KUBEVISION_AUDIT_ENABLED": "false",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Audit.Enabled {
					t.Error("Audit.Enabled = true, want false")
				}
			},
		},
		{
			name: "AuditEnabledOne",
			envVars: map[string]string{
				"KUBEVISION_AUDIT_ENABLED": "1",
			},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Audit.Enabled {
					t.Error("Audit.Enabled = false, want true (for value \"1\")")
				}
			},
		},
		{
			name: "AuditRetentionDays",
			envVars: map[string]string{
				"KUBEVISION_AUDIT_RETENTION_DAYS": "365",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Audit.RetentionDays != 365 {
					t.Errorf("Audit.RetentionDays = %d, want 365", cfg.Audit.RetentionDays)
				}
			},
		},
		{
			name: "EncryptKey",
			envVars: map[string]string{
				"KUBEVISION_ENCRYPT_KEY": "env-encrypt-key",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.EncryptKey != "env-encrypt-key" {
					t.Errorf("EncryptKey = %q, want %q", cfg.EncryptKey, "env-encrypt-key")
				}
			},
		},
		{
			name: "InvalidPortIsIgnored",
			envVars: map[string]string{
				"KUBEVISION_SERVER_PORT": "not-a-number",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 8080 {
					t.Errorf("Server.Port = %d, want 8080 (invalid value should be ignored)", cfg.Server.Port)
				}
			},
		},
		{
			name: "InvalidDurationIsIgnored",
			envVars: map[string]string{
				"KUBEVISION_ACCESS_TOKEN_TTL": "not-a-duration",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Auth.AccessTokenTTL != 30*time.Minute {
					t.Errorf("Auth.AccessTokenTTL = %v, want %v (invalid value should be ignored)", cfg.Auth.AccessTokenTTL, 30*time.Minute)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv automatically restores the env var when the test finishes.
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := Load("")
			if err != nil {
				t.Fatalf("Load returned unexpected error: %v", err)
			}
			tt.check(t, cfg)
		})
	}
}

func TestAutoGeneratedSecrets(t *testing.T) {
	t.Run("JWTSecretAutoGenerated", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		if cfg.Auth.JWTSecret == "" {
			t.Error("Auth.JWTSecret should be auto-generated when empty")
		}
		// A 32-byte random hex string should be 64 hex chars.
		if len(cfg.Auth.JWTSecret) != 64 {
			t.Errorf("Auth.JWTSecret length = %d, want 64 hex chars", len(cfg.Auth.JWTSecret))
		}
	})

	t.Run("EncryptKeyAutoGenerated", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		if cfg.EncryptKey == "" {
			t.Error("EncryptKey should be auto-generated when empty")
		}
		if len(cfg.EncryptKey) != 64 {
			t.Errorf("EncryptKey length = %d, want 64 hex chars", len(cfg.EncryptKey))
		}
	})

	t.Run("ProvidedSecretNotOverwritten", func(t *testing.T) {
		t.Setenv("KUBEVISION_JWT_SECRET", "user-provided-secret-with-at-least-32-characters")
		t.Setenv("KUBEVISION_ENCRYPT_KEY", "user-provided-key")

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		if cfg.Auth.JWTSecret != "user-provided-secret-with-at-least-32-characters" {
			t.Errorf("Auth.JWTSecret = %q, want %q", cfg.Auth.JWTSecret, "user-provided-secret-with-at-least-32-characters")
		}
		if cfg.EncryptKey != "user-provided-key" {
			t.Errorf("EncryptKey = %q, want %q", cfg.EncryptKey, "user-provided-key")
		}
	})

	t.Run("ConfiguredSecretsFile", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "runtime-secrets.yaml")
		t.Setenv("KUBEVISION_SECRETS_FILE", path)

		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		if cfg.Auth.JWTSecret == "" || cfg.EncryptKey == "" {
			t.Fatal("expected generated secrets")
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat configured secrets file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("secrets file mode = %o, want 600", info.Mode().Perm())
		}
	})

	t.Run("GeneratedSecretsAreUnique", func(t *testing.T) {
		// Run in a temp dir so each Load creates a fresh secrets file.
		tmp := t.TempDir()
		origDir, _ := os.Getwd()
		_ = os.Chdir(tmp)
		defer func() { _ = os.Chdir(origDir) }()

		cfg1, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		// Remove the persisted secrets file so the second Load generates new ones.
		_ = os.Remove(secretsFile)

		cfg2, err := Load("")
		if err != nil {
			t.Fatalf("Load returned unexpected error: %v", err)
		}
		if cfg1.Auth.JWTSecret == cfg2.Auth.JWTSecret {
			t.Error("Two calls to Load should produce different auto-generated JWT secrets")
		}
		if cfg1.EncryptKey == cfg2.EncryptKey {
			t.Error("Two calls to Load should produce different auto-generated encrypt keys")
		}
	})
}

func TestEnvOverridesYAMLValues(t *testing.T) {
	yamlContent := `
server:
  port: 9090
database:
  driver: postgres
`
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	t.Setenv("KUBEVISION_SERVER_PORT", "4000")
	t.Setenv("KUBEVISION_DB_DRIVER", "mysql")

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	// Env vars should override YAML values.
	if cfg.Server.Port != 4000 {
		t.Errorf("Server.Port = %d, want 4000 (env should override YAML)", cfg.Server.Port)
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("Database.Driver = %q, want %q (env should override YAML)", cfg.Database.Driver, "mysql")
	}
}

func TestRandomHex(t *testing.T) {
	t.Run("CorrectLength", func(t *testing.T) {
		hex, err := randomHex(16)
		if err != nil {
			t.Fatalf("randomHex returned unexpected error: %v", err)
		}
		if len(hex) != 32 {
			t.Errorf("randomHex(16) length = %d, want 32", len(hex))
		}
	})

	t.Run("DifferentCalls", func(t *testing.T) {
		hex1, _ := randomHex(32)
		hex2, _ := randomHex(32)
		if hex1 == hex2 {
			t.Error("Two calls to randomHex should produce different results")
		}
	})
}
