package httpaccess

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestClientUsesKubernetesProxyAndFiltersHeaders(t *testing.T) {
	var received *http.Request
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		received = r.Clone(r.Context())
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{
			"Set-Cookie": {"upstream=secret"}, "X-Upstream": {"kept"},
		}, Body: io.NopCloser(strings.NewReader("redirect"))}, nil
	})

	client, err := NewClient(&rest.Config{Host: "https://kubernetes.invalid", WrapTransport: func(http.RoundTripper) http.RoundTripper { return transport }})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, "http://kubevision.local/access", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer browser-token")
	req.Header.Set("Cookie", "session=secret")
	req.Header.Set("X-Trace", "kept")
	req.Body = io.NopCloser(strings.NewReader("must not pass"))
	req.ContentLength = int64(len("must not pass"))
	response, err := client.RoundTrip(req, "pods", "team-a", "web-0", "http", "/health/ready", url.Values{"watch": {"false"}})
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/namespaces/team-a/pods/web-0:http/proxy/health/ready", received.URL.Path)
	assert.Equal(t, "watch=false", received.URL.RawQuery)
	assert.Empty(t, received.Header.Get("Authorization"))
	assert.Empty(t, received.Header.Get("Cookie"))
	assert.Nil(t, received.Body)
	assert.Zero(t, received.ContentLength)
	assert.Empty(t, received.TransferEncoding)
	assert.Equal(t, "kept", received.Header.Get("X-Trace"))
	assert.Equal(t, http.StatusTemporaryRedirect, response.StatusCode)
	assert.Equal(t, "redirect", string(body))
	filtered := FilterResponseHeaders(response.Header)
	assert.Empty(t, filtered.Get("Set-Cookie"))
	assert.Equal(t, "kept", filtered.Get("X-Upstream"))
}

func TestFilterHeadersRemovesConnectionNamedHeaders(t *testing.T) {
	headers := http.Header{
		"Connection":   {"X-Internal, keep-alive"},
		"X-Internal":   {"secret"},
		"Content-Type": {"text/plain"},
	}
	filtered := FilterRequestHeaders(headers)
	assert.Empty(t, filtered.Get("Connection"))
	assert.Empty(t, filtered.Get("X-Internal"))
	assert.Equal(t, "text/plain", filtered.Get("Content-Type"))
}

func TestStreamingBodyRemainsReadableAfterRoundTrip(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(strings.Repeat("chunk", 1024)))}, nil
	})
	client, err := NewClient(&rest.Config{Host: "https://kubernetes.invalid", WrapTransport: func(http.RoundTripper) http.RoundTripper { return transport }})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, "http://kubevision.local/", nil)
	require.NoError(t, err)
	response, err := client.RoundTrip(req, "services", "default", "web", "8080", "/", nil)
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Len(t, data, 5*1024)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
