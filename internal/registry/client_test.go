package registry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type staticResolver map[string][]net.IPAddr

func (r staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	return append([]net.IPAddr(nil), r[host]...), nil
}

func localClient(t *testing.T, server *httptest.Server, authHosts ...string) (*Client, Reference) {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	dialer := &net.Dialer{Timeout: time.Second}
	c := NewClient(ClientConfig{AllowedRegistries: []string{u.Host}, AllowedAuthHosts: authHosts, AllowPrivate: true, AllowHTTP: true, DialContext: dialer.DialContext})
	return c, Reference{Registry: u.Host, Repository: "team/app"}
}

func TestClientPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/team/app/tags/list" || r.URL.Query().Get("n") != "2" || r.URL.Query().Get("last") != "old" {
			t.Errorf("unexpected request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"name":"team/app","tags":["one","two"]}`))
	}))
	defer server.Close()
	c, ref := localClient(t, server)
	tags, next, err := c.Tags(context.Background(), ref, 2, "old")
	if err != nil || len(tags) != 2 || next != "two" {
		t.Fatalf("tags=%v next=%q err=%v", tags, next, err)
	}
}

func TestClientBearerChallenge(t *testing.T) {
	var registry *httptest.Server
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"token":"secret-token"}`)) }))
	defer auth.Close()
	authURL, _ := url.Parse(auth.URL)
	registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s/token",service="registry.test",scope="repository:team/app:pull"`, auth.URL))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"tags":["latest"]}`))
	}))
	defer registry.Close()
	c, ref := localClient(t, registry, authURL.Host)
	tags, _, err := c.Tags(context.Background(), ref, 10, "")
	if err != nil || len(tags) != 1 {
		t.Fatalf("tags=%v err=%v", tags, err)
	}
}

func TestClientRejectsUnlistedRegistry(t *testing.T) {
	c := NewClient(ClientConfig{})
	_, _, err := c.Tags(context.Background(), Reference{Registry: "example.com", Repository: "app"}, 10, "")
	if !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestClientDialsOnlyResolvedValidatedIP(t *testing.T) {
	var dialed string
	c := NewClient(ClientConfig{
		AllowedRegistries: []string{"registry.example:5443"},
		Resolver:          staticResolver{"registry.example": {{IP: net.ParseIP("203.0.113.9")}}},
		DialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
			dialed = address
			return nil, errors.New("stop after observing dial target")
		},
	})
	_, _, _ = c.Tags(context.Background(), Reference{Registry: "registry.example:5443", Repository: "team/app"}, 10, "")
	if dialed != "203.0.113.9:5443" {
		t.Fatalf("dialed %q; expected the validated IP and original port", dialed)
	}
}

func TestClientPreservesTransportDefaultPortWhenDialingValidatedIP(t *testing.T) {
	for _, tt := range []struct{ scheme, port string }{{"http", "80"}, {"https", "443"}} {
		t.Run(tt.scheme, func(t *testing.T) {
			var dialed string
			c := NewClient(ClientConfig{
				Resolver: staticResolver{"registry.example": {{IP: net.ParseIP("203.0.113.10")}}},
				DialContext: func(_ context.Context, _ string, address string) (net.Conn, error) {
					dialed = address
					return nil, errors.New("stop after observing dial target")
				},
			})
			_, _ = c.request(context.Background(), &url.URL{Scheme: tt.scheme, Host: "registry.example", Path: "/v2/"}, "")
			want := net.JoinHostPort("203.0.113.10", tt.port)
			if dialed != want {
				t.Fatalf("dialed %q, want %q", dialed, want)
			}
		})
	}
}

func TestClientDisablesEnvironmentProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	c := NewClient(ClientConfig{})
	transport, ok := c.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatal("registry transport must not use environment proxies")
	}
}

func TestRedirectDoesNotForwardAuthorizationAcrossHosts(t *testing.T) {
	var mu sync.Mutex
	forwarded := ""
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		forwarded = r.Header.Get("Authorization")
		mu.Unlock()
		_, _ = w.Write([]byte(`{"tags":[]}`))
	}))
	defer target.Close()
	targetURL, _ := url.Parse(target.URL)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/redirected", http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	sourceURL, _ := url.Parse(source.URL)
	dialer := &net.Dialer{Timeout: time.Second}
	c := NewClient(ClientConfig{
		AllowedRegistries: []string{sourceURL.Host, targetURL.Host},
		AllowPrivate:      true, AllowHTTP: true, DialContext: dialer.DialContext,
	})
	resp, err := c.request(context.Background(), &url.URL{Scheme: "http", Host: sourceURL.Host, Path: "/start"}, "Bearer sensitive")
	if err != nil {
		t.Fatal(err)
	}
	closeResponse(resp)
	mu.Lock()
	got := forwarded
	mu.Unlock()
	if got != "" {
		t.Fatalf("authorization forwarded across hosts: %q", got)
	}
}

func TestUnsafeIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "::1", "ff02::1"}
	for _, raw := range blocked {
		if !unsafeIP(net.ParseIP(raw)) {
			t.Errorf("expected %s blocked", raw)
		}
	}
	if unsafeIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public address blocked")
	}
}

func TestChallengeParserDoesNotExposeTokenMaterial(t *testing.T) {
	params := parseChallenge(`realm="https://auth.example/token",service="registry",scope="repository:a/b:pull"`)
	if params["scope"] != "repository:a/b:pull" || strings.Contains(fmt.Sprint(params), "password") {
		t.Fatalf("unexpected parse: %v", params)
	}
}
