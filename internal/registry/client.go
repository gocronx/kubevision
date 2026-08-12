package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotAllowed     = errors.New("registry is not allowed")
	ErrUnsafeAddress  = errors.New("registry resolved to a blocked address")
	ErrAuthentication = errors.New("registry authentication failed")
)

// ClientConfig defines administrator-controlled outbound network boundaries.
type ClientConfig struct {
	AllowedRegistries []string
	AllowedAuthHosts  []string
	AllowPrivate      bool
	AllowHTTP         bool
	ConnectTimeout    time.Duration
	HeaderTimeout     time.Duration
	TotalTimeout      time.Duration
	MaxResponseBytes  int64
	Resolver          interface {
		LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
	}
	DialContext func(context.Context, string, string) (net.Conn, error)
}

type Client struct {
	httpClient *http.Client
	config     ClientConfig
	registries map[string]struct{}
	authHosts  map[string]struct{}
}

func NewClient(cfg ClientConfig) *Client {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 3 * time.Second
	}
	if cfg.HeaderTimeout <= 0 {
		cfg.HeaderTimeout = 5 * time.Second
	}
	if cfg.TotalTimeout <= 0 {
		cfg.TotalTimeout = 10 * time.Second
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = 2 << 20
	}
	if cfg.Resolver == nil {
		cfg.Resolver = net.DefaultResolver
	}
	registries := hostSet(append([]string{"docker.io", "registry-1.docker.io"}, cfg.AllowedRegistries...))
	authHosts := hostSet(append([]string{"auth.docker.io"}, cfg.AllowedAuthHosts...))
	c := &Client{config: cfg, registries: registries, authHosts: authHosts}
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout, KeepAlive: 30 * time.Second}
	dial := cfg.DialContext
	if dial == nil {
		dial = dialer.DialContext
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return c.dialValidated(ctx, network, address, dial)
		},
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		ResponseHeaderTimeout: cfg.HeaderTimeout,
		MaxIdleConns:          32,
		IdleConnTimeout:       30 * time.Second,
	}
	c.httpClient = &http.Client{Transport: transport, Timeout: cfg.TotalTimeout}
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if err := c.validateURL(req.Context(), req.URL, "redirect"); err != nil {
			return err
		}
		if len(via) > 0 && !sameAuthority(via[len(via)-1].URL, req.URL) {
			req.Header.Del("Authorization")
		}
		return nil
	}
	return c
}

type tagResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (c *Client) Tags(ctx context.Context, ref Reference, limit int, last string) ([]string, string, error) {
	if _, ok := c.registries[normalizeAuthority(ref.Registry)]; !ok {
		return nil, "", ErrNotAllowed
	}
	host := ref.Registry
	if host == "docker.io" {
		host = "registry-1.docker.io"
	}
	scheme := "https"
	if c.config.AllowHTTP {
		scheme = "http"
	}
	u := &url.URL{Scheme: scheme, Host: host, Path: "/v2/" + escapeRepository(ref.Repository) + "/tags/list"}
	q := u.Query()
	q.Set("n", strconv.Itoa(limit))
	if last != "" {
		q.Set("last", last)
	}
	u.RawQuery = q.Encode()
	if err := c.validateURL(ctx, u, "registry"); err != nil {
		return nil, "", err
	}

	resp, err := c.request(ctx, u, "")
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		challenge := resp.Header.Get("WWW-Authenticate")
		closeResponse(resp)
		token, tokenErr := c.bearerToken(ctx, challenge)
		if tokenErr != nil {
			return nil, "", tokenErr
		}
		resp, err = c.request(ctx, u, "Bearer "+token)
		if err != nil {
			return nil, "", err
		}
	}
	defer closeResponse(resp)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, "", ErrAuthentication
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}
	var payload tagResponse
	if err := decodeLimited(resp.Body, c.config.MaxResponseBytes, &payload); err != nil {
		return nil, "", err
	}
	next := ""
	if len(payload.Tags) == limit {
		next = payload.Tags[len(payload.Tags)-1]
	}
	return payload.Tags, next, nil
}

func (c *Client) request(ctx context.Context, u *url.URL, authorization string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.httpClient.Do(req)
}

func (c *Client) bearerToken(ctx context.Context, header string) (string, error) {
	scheme, raw, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrAuthentication
	}
	params := parseChallenge(raw)
	realm, err := url.Parse(params["realm"])
	if err != nil || realm.Host == "" {
		return "", ErrAuthentication
	}
	if err := c.validateURL(ctx, realm, "auth"); err != nil {
		return "", err
	}
	q := realm.Query()
	for _, key := range []string{"service", "scope"} {
		if params[key] != "" {
			q.Set(key, params[key])
		}
	}
	realm.RawQuery = q.Encode()
	resp, err := c.request(ctx, realm, "")
	if err != nil {
		return "", err
	}
	defer closeResponse(resp)
	if resp.StatusCode != http.StatusOK {
		return "", ErrAuthentication
	}
	var payload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := decodeLimited(resp.Body, c.config.MaxResponseBytes, &payload); err != nil {
		return "", err
	}
	if payload.Token == "" {
		payload.Token = payload.AccessToken
	}
	if payload.Token == "" {
		return "", ErrAuthentication
	}
	return payload.Token, nil
}

func (c *Client) validateURL(ctx context.Context, u *url.URL, kind string) error {
	if u.User != nil || u.Host == "" {
		return ErrNotAllowed
	}
	if u.Scheme != "https" && !(c.config.AllowHTTP && u.Scheme == "http") {
		return ErrNotAllowed
	}
	host := normalizeAuthority(u.Host)
	allowed := c.registries
	if kind == "auth" {
		allowed = c.authHosts
	} else if kind == "redirect" {
		if _, ok := c.registries[host]; !ok {
			if _, ok := c.authHosts[host]; !ok {
				return ErrNotAllowed
			}
		}
		return c.validateAddress(ctx, u.Host)
	}
	if _, ok := allowed[host]; !ok {
		return ErrNotAllowed
	}
	return c.validateAddress(ctx, u.Host)
}

func (c *Client) validateAddress(ctx context.Context, address string) error {
	host := addressHost(address)
	addresses, err := c.config.Resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	if len(addresses) == 0 {
		return ErrUnsafeAddress
	}
	if c.config.AllowPrivate {
		return nil
	}
	for _, address := range addresses {
		if unsafeIP(address.IP) {
			return ErrUnsafeAddress
		}
	}
	return nil
}

func (c *Client) dialValidated(
	ctx context.Context,
	network, address string,
	dial func(context.Context, string, string) (net.Conn, error),
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("registry dial address must include a port: %w", err)
	}
	addresses, err := c.config.Resolver.LookupIPAddr(ctx, strings.Trim(host, "[]"))
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, ErrUnsafeAddress
	}
	for _, candidate := range addresses {
		if !c.config.AllowPrivate && unsafeIP(candidate.IP) {
			return nil, ErrUnsafeAddress
		}
	}

	var dialErr error
	for _, candidate := range addresses {
		conn, err := dial(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	return nil, dialErr
}

func unsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

func addressHost(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(address, "[]")
}

func sameAuthority(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func hostSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, v := range values {
		if h := normalizeAuthority(v); h != "" {
			result[h] = struct{}{}
		}
	}
	return result
}
func normalizeAuthority(value string) string {
	if u, err := url.Parse("//" + strings.TrimSpace(value)); err == nil {
		return strings.ToLower(u.Host)
	}
	return ""
}
func escapeRepository(repo string) string {
	parts := strings.Split(repo, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}
}
func decodeLimited(reader io.Reader, max int64, target any) error {
	limited := &io.LimitedReader{R: reader, N: max + 1}
	dec := json.NewDecoder(limited)
	if err := dec.Decode(target); err != nil {
		return err
	}
	if limited.N <= 0 {
		return errors.New("registry response exceeds size limit")
	}
	return nil
}

func parseChallenge(raw string) map[string]string {
	result := map[string]string{}
	for len(raw) > 0 {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, ","))
		i := strings.IndexByte(raw, '=')
		if i < 1 {
			break
		}
		key := strings.ToLower(strings.TrimSpace(raw[:i]))
		raw = strings.TrimSpace(raw[i+1:])
		if !strings.HasPrefix(raw, `"`) {
			break
		}
		raw = raw[1:]
		j := strings.IndexByte(raw, '"')
		if j < 0 {
			break
		}
		result[key], raw = raw[:j], raw[j+1:]
	}
	return result
}
