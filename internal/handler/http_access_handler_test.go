package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

func TestNormalizedHTTPPath(t *testing.T) {
	tests := []struct {
		raw, want string
		invalid   bool
	}{
		{"", "/", false}, {"/health/ready", "/health/ready", false},
		{"/../secret", "", true}, {"/%2e%2e/secret", "", true},
		{"/%252e%252e/secret", "", true}, {"/a%2fb", "", true},
		{"/a%00b", "", true}, {"/%", "", true},
	}
	for _, test := range tests {
		got, err := normalizedHTTPPath(test.raw)
		if test.invalid {
			assert.Error(t, err, test.raw)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		}
	}
}

func TestHTTPAccessHandlerStreamsAndAudits(t *testing.T) {
	var upstreamRequest *http.Request
	transport := handlerRoundTripper(func(request *http.Request) (*http.Response, error) {
		upstreamRequest = request.Clone(request.Context())
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{
			"Content-Type": {"text/plain"}, "Set-Cookie": {"blocked=yes"}, "Location": {"/login"},
		}, Body: io.NopCloser(strings.NewReader("not followed"))}, nil
	})
	clients := &fakeHTTPClients{config: &rest.Config{Host: "https://kubernetes.invalid", WrapTransport: func(http.RoundTripper) http.RoundTripper { return transport }}}
	audit := &recordingAudit{}
	handler := NewHTTPAccessHandler(clients, newStubClusterRepo(&model.Cluster{ID: 7, Name: "dev"}), &httpAccessRoleRepo{role: &model.Role{Name: "viewer", Permissions: `["pods:get"]`}}, audit, nil)
	router := setupHTTPAccessRouter(handler)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/7/namespaces/default/http/pods/web-0/health?port=8080&verbose=true", nil)
	request.Header.Set("Authorization", "Bearer browser-secret")
	request.Header.Set("Cookie", "session=secret")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "not followed", recorder.Body.String())
	assert.Equal(t, "/login", recorder.Header().Get("Location"))
	assert.Empty(t, recorder.Header().Get("Set-Cookie"))
	require.NotNil(t, upstreamRequest)
	assert.Equal(t, "/api/v1/namespaces/default/pods/web-0:8080/proxy/health", upstreamRequest.URL.Path)
	assert.Equal(t, "verbose=true", upstreamRequest.URL.RawQuery)
	assert.Empty(t, upstreamRequest.Header.Get("Authorization"))
	assert.Empty(t, upstreamRequest.Header.Get("Cookie"))
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "http-access", audit.entries[0].Action)
	assert.Equal(t, "/health", audit.entries[0].Path)
	assert.Equal(t, "8080", audit.entries[0].Port)
}

func TestHTTPAccessHandlerAuthorizationAndValidation(t *testing.T) {
	tests := []struct {
		name, path, permissions string
		cluster                 bool
		wantCode                int
	}{
		{"kind", "/api/v1/clusters/7/namespaces/default/http/nodes/node-a?port=80", `["*:*"]`, true, 40002},
		{"namespace", "/api/v1/clusters/7/namespaces/Bad_NS/http/pods/web?port=80", `["pods:get"]`, true, 40002},
		{"port low", "/api/v1/clusters/7/namespaces/default/http/pods/web?port=0", `["pods:get"]`, true, 40002},
		{"port high", "/api/v1/clusters/7/namespaces/default/http/services/web?port=65536", `["services:get"]`, true, 40002},
		{"traversal", "/api/v1/clusters/7/namespaces/default/http/pods/web/%252e%252e/secret?port=80", `["pods:get"]`, true, 40002},
		{"denied concrete kind", "/api/v1/clusters/7/namespaces/default/http/services/web?port=80", `["pods:get"]`, true, 40300},
		{"missing cluster", "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", `["pods:get"]`, false, 40400},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusters := newStubClusterRepo()
			if test.cluster {
				clusters = newStubClusterRepo(&model.Cluster{ID: 7, Name: "dev"})
			}
			handler := NewHTTPAccessHandler(&fakeHTTPClients{config: &rest.Config{Host: "https://kubernetes.invalid"}}, clusters, &httpAccessRoleRepo{role: &model.Role{Name: "viewer", Permissions: test.permissions}}, &recordingAudit{}, nil)
			recorder := httptest.NewRecorder()
			setupHTTPAccessRouter(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			assert.Contains(t, recorder.Body.String(), `"code":`+strconv.Itoa(test.wantCode))
		})
	}
}

func TestHTTPAccessHandlerAcceptsDeclaredNamedPort(t *testing.T) {
	pod := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "web", "namespace": "default"},
		"spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"name": "app", "ports": []interface{}{map[string]interface{}{"name": "http", "containerPort": int64(8080)}}}}},
	}}
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{{Version: "v1", Resource: "pods"}: "PodList"}, pod)
	transport := handlerRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	clients := &fakeHTTPClients{dynamic: dynamicClient, config: &rest.Config{Host: "https://kubernetes.invalid", WrapTransport: func(http.RoundTripper) http.RoundTripper { return transport }}}
	handler := NewHTTPAccessHandler(clients, newStubClusterRepo(&model.Cluster{ID: 7, Name: "dev"}), &httpAccessRoleRepo{role: &model.Role{Name: "viewer", Permissions: `["pods:get"]`}}, &recordingAudit{}, nil)
	recorder := httptest.NewRecorder()
	setupHTTPAccessRouter(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=http", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ok", recorder.Body.String())
}

func TestHTTPAccessHandlerRejectsGETAndHEADBodies(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodHead}
	for _, method := range methods {
		t.Run(method+" content length", func(t *testing.T) {
			called := false
			handler := testHTTPAccessHandler(handlerRoundTripper(func(*http.Request) (*http.Response, error) {
				called = true
				return nil, errors.New("must not be called")
			}))
			request := httptest.NewRequest(method, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", strings.NewReader("body"))
			recorder := httptest.NewRecorder()
			setupHTTPAccessRouter(handler).ServeHTTP(recorder, request)
			assert.Contains(t, recorder.Body.String(), `"code":40002`)
			assert.False(t, called)
		})

		t.Run(method+" transfer encoding", func(t *testing.T) {
			called := false
			handler := testHTTPAccessHandler(handlerRoundTripper(func(*http.Request) (*http.Response, error) {
				called = true
				return nil, errors.New("must not be called")
			}))
			request := httptest.NewRequest(method, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", nil)
			request.TransferEncoding = []string{"chunked"}
			recorder := httptest.NewRecorder()
			setupHTTPAccessRouter(handler).ServeHTTP(recorder, request)
			assert.Contains(t, recorder.Body.String(), `"code":40002`)
			assert.False(t, called)
		})
	}
}

func TestHTTPAccessHandlerRejectsKnownOversizeResponseBeforeHeaders(t *testing.T) {
	handler := testHTTPAccessHandler(handlerRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, ContentLength: maxHTTPAccessResponseBytes + 1,
			Header: http.Header{"Content-Length": {strconv.FormatInt(maxHTTPAccessResponseBytes+1, 10)}, "X-Upstream": {"must-not-leak"}},
			Body:   io.NopCloser(strings.NewReader("unread")),
		}, nil
	}))
	recorder := httptest.NewRecorder()
	setupHTTPAccessRouter(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", nil))
	assert.Contains(t, recorder.Body.String(), `"code":50200`)
	assert.Empty(t, recorder.Header().Get("X-Upstream"))
	assert.NotEqual(t, strconv.FormatInt(maxHTTPAccessResponseBytes+1, 10), recorder.Header().Get("Content-Length"))
}

func TestHTTPAccessHandlerUnknownLengthTruncatesWithoutExtraByte(t *testing.T) {
	body := strings.Repeat("a", maxHTTPAccessResponseBytes) + "z"
	handler := testHTTPAccessHandler(handlerRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, ContentLength: -1,
			Header: http.Header{"Content-Length": {"999999999"}},
			Body:   io.NopCloser(strings.NewReader(body)),
		}, nil
	}))
	recorder := httptest.NewRecorder()
	setupHTTPAccessRouter(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Len(t, recorder.Body.Bytes(), maxHTTPAccessResponseBytes)
	assert.NotContains(t, recorder.Body.String(), "z")
	assert.Empty(t, recorder.Header().Get("Content-Length"))
	assert.Equal(t, "close", recorder.Header().Get("Connection"))
	assert.Equal(t, "true", recorder.Header().Get(responseTruncatedTrailer))
}

func TestHTTPAccessHandlerHEADPreservesUpstreamLengthWithoutReadingBody(t *testing.T) {
	body := &readTrackingBody{Reader: strings.NewReader("body")}
	handler := testHTTPAccessHandler(handlerRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, ContentLength: 4, Header: http.Header{"Content-Length": {"4"}}, Body: body}, nil
	}))
	recorder := httptest.NewRecorder()
	setupHTTPAccessRouter(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/api/v1/clusters/7/namespaces/default/http/pods/web?port=80", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "4", recorder.Header().Get("Content-Length"))
	assert.Empty(t, recorder.Body.String())
	assert.False(t, body.read)
}

func testHTTPAccessHandler(transport http.RoundTripper) *HTTPAccessHandler {
	clients := &fakeHTTPClients{config: &rest.Config{Host: "https://kubernetes.invalid", WrapTransport: func(http.RoundTripper) http.RoundTripper { return transport }}}
	return NewHTTPAccessHandler(clients, newStubClusterRepo(&model.Cluster{ID: 7, Name: "dev"}), &httpAccessRoleRepo{role: &model.Role{Name: "viewer", Permissions: `["pods:get"]`}}, &recordingAudit{}, nil)
}

type readTrackingBody struct {
	*strings.Reader
	read bool
}

func (body *readTrackingBody) Read(buffer []byte) (int, error) {
	body.read = true
	return body.Reader.Read(buffer)
}
func (body *readTrackingBody) Close() error { return nil }

func setupHTTPAccessRouter(handler *HTTPAccessHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(9))
		c.Set("username", "alice")
		c.Set("userRole", "viewer")
		c.Next()
	})
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/http/:kind/:name", handler.Serve)
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/http/:kind/:name/*path", handler.Serve)
	router.HEAD("/api/v1/clusters/:id/namespaces/:namespace/http/:kind/:name", handler.Serve)
	router.HEAD("/api/v1/clusters/:id/namespaces/:namespace/http/:kind/:name/*path", handler.Serve)
	return router
}

type fakeHTTPClients struct {
	config  *rest.Config
	dynamic dynamic.Interface
}

func (f *fakeHTTPClients) RESTConfig(string) (*rest.Config, error) {
	if f.config == nil {
		return nil, errors.New("missing")
	}
	return f.config, nil
}
func (f *fakeHTTPClients) DynamicClient(string) (dynamic.Interface, error) {
	if f.dynamic == nil {
		return nil, errors.New("missing")
	}
	return f.dynamic, nil
}

type recordingAudit struct{ entries []model.AuditLog }

func (a *recordingAudit) Record(entry model.AuditLog) { a.entries = append(a.entries, entry) }

type handlerRoundTripper func(*http.Request) (*http.Response, error)

func (fn handlerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type httpAccessRoleRepo struct{ role *model.Role }

func (r *httpAccessRoleRepo) Create(context.Context, *model.Role) error          { return nil }
func (r *httpAccessRoleRepo) GetByID(context.Context, uint) (*model.Role, error) { return r.role, nil }
func (r *httpAccessRoleRepo) GetByName(_ context.Context, name string) (*model.Role, error) {
	if r.role == nil || r.role.Name != name {
		return nil, errors.New("not found")
	}
	return r.role, nil
}
func (r *httpAccessRoleRepo) Update(context.Context, *model.Role) error  { return nil }
func (r *httpAccessRoleRepo) Delete(context.Context, uint) error         { return nil }
func (r *httpAccessRoleRepo) List(context.Context) ([]model.Role, error) { return nil, nil }

var _ repository.RoleRepo = (*httpAccessRoleRepo)(nil)
