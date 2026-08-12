package directory

import (
	"testing"
	"time"
)

func validTestConfig() Config {
	return Config{ServerURL: "ldap://directory.example.test:389", StartTLS: true, UserBaseDN: "ou=people,dc=example,dc=test", UserFilter: "(uid={{username}})", StableIDAttribute: "entryUUID", UsernameAttribute: "uid", GroupAttribute: "memberOf", ConnectTimeout: time.Second, SearchTimeout: time.Second}
}

func TestEscapeAssertionRFC4515(t *testing.T) {
	got := EscapeAssertion("alice*)(uid=*)\\\x00")
	want := `alice\2a\29\28uid=\2a\29\5c\00`
	if got != want {
		t.Fatalf("EscapeAssertion() = %q, want %q", got, want)
	}
}

func TestValidateTransportSecurity(t *testing.T) {
	t.Run("plain LDAP rejected by default", func(t *testing.T) {
		cfg := validTestConfig()
		cfg.StartTLS = false
		if err := Validate(cfg); err == nil {
			t.Fatal("expected plaintext LDAP rejection")
		}
	})
	t.Run("explicit development plaintext accepted", func(t *testing.T) {
		cfg := validTestConfig()
		cfg.StartTLS = false
		cfg.AllowPlaintext = true
		if err := Validate(cfg); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
	t.Run("production plaintext rejected even when requested", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		cfg := validTestConfig()
		cfg.StartTLS = false
		cfg.AllowPlaintext = true
		if err := Validate(cfg); err == nil {
			t.Fatal("expected production plaintext rejection")
		}
	})
	t.Run("LDAPS accepted", func(t *testing.T) {
		cfg := validTestConfig()
		cfg.ServerURL = "ldaps://directory.example.test:636"
		cfg.StartTLS = false
		if err := Validate(cfg); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
	t.Run("invalid custom CA rejected", func(t *testing.T) {
		cfg := validTestConfig()
		cfg.CABundle = "not a certificate"
		if err := Validate(cfg); err == nil {
			t.Fatal("expected invalid CA rejection")
		}
	})
}

func TestValidateRequiresSingleEscapedPlaceholder(t *testing.T) {
	for _, filter := range []string{"(uid=alice)", "(|(uid={{username}})(mail={{username}}))"} {
		cfg := validTestConfig()
		cfg.UserFilter = filter
		if err := Validate(cfg); err == nil {
			t.Fatalf("expected filter %q rejection", filter)
		}
	}
}
