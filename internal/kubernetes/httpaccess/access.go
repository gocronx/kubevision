package httpaccess

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"k8s.io/client-go/rest"
)

const defaultTimeout = 30 * time.Second

var blockedHeaders = map[string]struct{}{
	"Authorization": {}, "Cookie": {}, "Proxy-Authorization": {},
	"Connection": {}, "Keep-Alive": {}, "Proxy-Authenticate": {},
	"Te": {}, "Trailer": {}, "Transfer-Encoding": {}, "Upgrade": {},
	"Forwarded": {}, "X-Forwarded-For": {}, "X-Forwarded-Host": {},
	"X-Forwarded-Proto": {}, "X-Real-Ip": {}, "X-Api-Key": {},
}

// Client sends requests only through a selected Kubernetes API server.
type Client struct {
	host      *url.URL
	transport http.RoundTripper
}

func NewClient(config *rest.Config) (*Client, error) {
	host, err := url.Parse(config.Host)
	if err != nil || host.Scheme == "" || host.Host == "" {
		return nil, fmt.Errorf("invalid kubernetes API server")
	}
	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes transport: %w", err)
	}
	return &Client{host: host, transport: transport}, nil
}

// RoundTrip builds a Kubernetes proxy-subresource request from structured
// fields. No caller-controlled host or scheme is accepted.
func (c *Client) RoundTrip(req *http.Request, kind, namespace, name, port, relativePath string, query url.Values) (*http.Response, error) {
	resource := kind
	target := name + ":" + port
	segments := []string{"api", "v1", "namespaces", namespace, resource, target, "proxy"}
	upstream := *c.host
	upstream.Path = strings.TrimSuffix(c.host.Path, "/") + "/" + path.Join(segments...)
	if relativePath != "/" {
		upstream.Path += relativePath
	} else {
		upstream.Path += "/"
	}
	upstream.RawQuery = query.Encode()

	ctx := req.Context()
	if _, ok := ctx.Deadline(); !ok {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		forward := req.Clone(ctx)
		forward.URL = &upstream
		forward.RequestURI = ""
		forward.Host = upstream.Host
		forward.Header = FilterRequestHeaders(req.Header)
		removeRequestBody(forward)
		response, err := c.transport.RoundTrip(forward)
		if err != nil {
			cancel()
			return nil, err
		}
		response.Body = &cancelOnClose{ReadCloser: response.Body, cancel: cancel}
		return response, nil
	}
	forward := req.Clone(ctx)
	forward.URL = &upstream
	forward.RequestURI = ""
	forward.Host = upstream.Host
	forward.Header = FilterRequestHeaders(req.Header)
	removeRequestBody(forward)
	return c.transport.RoundTrip(forward)
}

func removeRequestBody(request *http.Request) {
	request.Body = nil
	request.GetBody = nil
	request.ContentLength = 0
	request.TransferEncoding = nil
	request.Header.Del("Content-Length")
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (body *cancelOnClose) Close() error {
	err := body.ReadCloser.Close()
	body.cancel()
	return err
}

// FilterRequestHeaders removes credentials, cookies, forwarding metadata, and
// hop-by-hop headers before a request enters the Kubernetes API transport.
func FilterRequestHeaders(source http.Header) http.Header {
	return filterHeaders(source, true)
}

// FilterResponseHeaders removes cookies, forwarding metadata, and hop-by-hop
// headers before returning an application response.
func FilterResponseHeaders(source http.Header) http.Header {
	return filterHeaders(source, false)
}

func filterHeaders(source http.Header, request bool) http.Header {
	out := make(http.Header)
	connectionTokens := map[string]struct{}{}
	for _, token := range strings.Split(source.Get("Connection"), ",") {
		connectionTokens[http.CanonicalHeaderKey(strings.TrimSpace(token))] = struct{}{}
	}
	for key, values := range source {
		canonical := http.CanonicalHeaderKey(key)
		_, blocked := blockedHeaders[canonical]
		_, connected := connectionTokens[canonical]
		if blocked || connected || (!request && canonical == "Set-Cookie") {
			continue
		}
		out[canonical] = append([]string(nil), values...)
	}
	return out
}
