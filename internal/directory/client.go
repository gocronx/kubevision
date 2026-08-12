package directory

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	ldap "github.com/go-ldap/ldap/v3"
)

var (
	ErrInvalidConfig = errors.New("invalid directory configuration")
	ErrUnavailable   = errors.New("directory unavailable")
	ErrNoMatch       = errors.New("directory identity not found")
	ErrAmbiguous     = errors.New("directory identity is ambiguous")
	ErrCredentials   = errors.New("directory credentials rejected")
)

type Config struct {
	ServerURL, CABundle, BindDN, BindPassword, UserBaseDN, UserFilter string
	StableIDAttribute, UsernameAttribute, DisplayAttribute            string
	EmailAttribute, GroupAttribute                                    string
	StartTLS, AllowPlaintext                                          bool
	ConnectTimeout, SearchTimeout                                     time.Duration
}

type Identity struct {
	StableID, Username, DisplayName, Email string
	Groups                                 []string
}

type Client interface {
	Authenticate(ctx context.Context, cfg Config, identifier, password string) (*Identity, error)
	Lookup(ctx context.Context, cfg Config, identifier string) (*Identity, error)
	Ping(ctx context.Context, cfg Config) error
}

type LDAPClient struct{}

func NewLDAPClient() *LDAPClient { return &LDAPClient{} }

// EscapeAssertion delegates RFC 4515 assertion escaping to the protocol library.
func EscapeAssertion(value string) string { return ldap.EscapeFilter(value) }

func (c *LDAPClient) Authenticate(ctx context.Context, cfg Config, identifier, password string) (*Identity, error) {
	if password == "" {
		return nil, ErrCredentials
	}
	conn, err := c.connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := bindService(conn, cfg); err != nil {
		return nil, ErrUnavailable
	}
	entry, id, err := searchOne(conn, cfg, identifier)
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, ErrCredentials
	}
	return id, nil
}

func (c *LDAPClient) Lookup(ctx context.Context, cfg Config, identifier string) (*Identity, error) {
	conn, err := c.connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := bindService(conn, cfg); err != nil {
		return nil, ErrUnavailable
	}
	_, id, err := searchOne(conn, cfg, identifier)
	return id, err
}

func (c *LDAPClient) Ping(ctx context.Context, cfg Config) error {
	conn, err := c.connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := bindService(conn, cfg); err != nil {
		return ErrCredentials
	}
	return nil
}

func (c *LDAPClient) connect(ctx context.Context, cfg Config) (*ldap.Conn, error) {
	u, tlsCfg, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout}
	conn, err := ldap.DialURL(cfg.ServerURL, ldap.DialWithDialer(dialer), ldap.DialWithTLSConfig(tlsCfg))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	conn.SetTimeout(cfg.SearchTimeout)
	if u.Scheme == "ldap" && cfg.StartTLS {
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close()
			return nil, ErrUnavailable
		}
	}
	return conn, nil
}

func validateConfig(cfg Config) (*url.URL, *tls.Config, error) {
	u, err := url.Parse(cfg.ServerURL)
	if err != nil || u.Host == "" || (u.Scheme != "ldap" && u.Scheme != "ldaps") || u.User != nil || (u.Path != "" && u.Path != "/") {
		return nil, nil, ErrInvalidConfig
	}
	if u.Scheme == "ldap" && !cfg.StartTLS && !cfg.AllowPlaintext {
		return nil, nil, ErrInvalidConfig
	}
	if u.Scheme == "ldap" && !cfg.StartTLS && cfg.AllowPlaintext && strings.EqualFold(os.Getenv("GIN_MODE"), "release") {
		return nil, nil, ErrInvalidConfig
	}
	if cfg.UserBaseDN == "" || strings.Count(cfg.UserFilter, "{{username}}") != 1 {
		return nil, nil, ErrInvalidConfig
	}
	for _, attr := range []string{cfg.StableIDAttribute, cfg.UsernameAttribute, cfg.DisplayAttribute, cfg.EmailAttribute, cfg.GroupAttribute} {
		if attr != "" && !validAttribute(attr) {
			return nil, nil, ErrInvalidConfig
		}
	}
	host := u.Hostname()
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	if cfg.CABundle != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(cfg.CABundle)) {
			return nil, nil, ErrInvalidConfig
		}
		tlsCfg.RootCAs = pool
	}
	return u, tlsCfg, nil
}

func validAttribute(v string) bool {
	for _, r := range v {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '.' {
			return false
		}
	}
	return v != ""
}

func bindService(conn *ldap.Conn, cfg Config) error {
	if cfg.BindDN == "" {
		return conn.UnauthenticatedBind("")
	}
	return conn.Bind(cfg.BindDN, cfg.BindPassword)
}

func searchOne(conn *ldap.Conn, cfg Config, identifier string) (*ldap.Entry, *Identity, error) {
	filter := strings.Replace(cfg.UserFilter, "{{username}}", EscapeAssertion(identifier), 1)
	attrs := unique([]string{cfg.StableIDAttribute, cfg.UsernameAttribute, cfg.DisplayAttribute, cfg.EmailAttribute, cfg.GroupAttribute})
	result, err := conn.Search(ldap.NewSearchRequest(cfg.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, int(cfg.SearchTimeout.Seconds()), false, filter, attrs, nil))
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	if len(result.Entries) == 0 {
		return nil, nil, ErrNoMatch
	}
	if len(result.Entries) != 1 {
		return nil, nil, ErrAmbiguous
	}
	e := result.Entries[0]
	rawID := e.GetRawAttributeValue(cfg.StableIDAttribute)
	if len(rawID) == 0 {
		return nil, nil, ErrNoMatch
	}
	sum := sha256.Sum256([]byte(strings.ToLower(cfg.ServerURL) + "\x00" + string(rawID)))
	username := strings.TrimSpace(e.GetAttributeValue(cfg.UsernameAttribute))
	if username == "" {
		return nil, nil, ErrNoMatch
	}
	return e, &Identity{
		StableID: hex.EncodeToString(sum[:]), Username: username,
		DisplayName: strings.TrimSpace(e.GetAttributeValue(cfg.DisplayAttribute)),
		Email:       strings.TrimSpace(e.GetAttributeValue(cfg.EmailAttribute)),
		Groups:      unique(e.GetAttributeValues(cfg.GroupAttribute)),
	}, nil
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func Classify(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalidConfig):
		return "configuration"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return "timeout"
	case errors.Is(err, ErrCredentials):
		return "bind"
	default:
		return "connection"
	}
}

func Validate(cfg Config) error {
	_, _, err := validateConfig(cfg)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
