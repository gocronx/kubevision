package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

// uuidV4Regex matches a standard UUID v4 format (8-4-4-4-12 hex groups).
var uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestRequestID_AddsHeaderToResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	rid := w.Header().Get("X-Request-ID")
	if rid == "" {
		t.Fatal("expected X-Request-ID header in response, got empty")
	}
}

func TestRequestID_GeneratedIDIsValidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	rid := w.Header().Get("X-Request-ID")
	if !uuidV4Regex.MatchString(rid) {
		t.Errorf("X-Request-ID %q is not a valid UUID format", rid)
	}
}

func TestRequestID_EachRequestGetsUniqueID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	const numRequests = 50
	ids := make(map[string]bool, numRequests)

	for i := 0; i < numRequests; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)

		rid := w.Header().Get("X-Request-ID")
		if rid == "" {
			t.Fatalf("request %d: X-Request-ID header is empty", i)
		}
		if ids[rid] {
			t.Fatalf("request %d: duplicate X-Request-ID %q", i, rid)
		}
		ids[rid] = true
	}

	if len(ids) != numRequests {
		t.Errorf("expected %d unique IDs, got %d", numRequests, len(ids))
	}
}

func TestRequestID_ReusesClientProvidedID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	clientID := "client-provided-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", clientID)
	router.ServeHTTP(w, req)

	rid := w.Header().Get("X-Request-ID")
	if rid != clientID {
		t.Errorf("expected X-Request-ID to be client-provided %q, got %q", clientID, rid)
	}
}

func TestRequestID_SetsContextValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	var contextRID string
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		v, exists := c.Get("requestID")
		if !exists {
			t.Error("requestID not found in context")
			return
		}
		contextRID = v.(string)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	headerRID := w.Header().Get("X-Request-ID")
	if contextRID != headerRID {
		t.Errorf("context requestID %q does not match header X-Request-ID %q", contextRID, headerRID)
	}
}
