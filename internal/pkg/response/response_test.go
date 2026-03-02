package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// parseResponseBody is a helper that unmarshals JSON response body into a Response struct.
func parseResponseBody(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

// parseResponseBodyWithMeta unmarshals the full JSON including the meta field as a raw map.
type rawResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    json.RawMessage  `json:"data,omitempty"`
	Meta    *json.RawMessage `json:"meta,omitempty"`
}

func TestSuccess(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantCode int
		wantMsg  string
	}{
		{
			name:     "StringData",
			data:     "hello",
			wantCode: 0,
			wantMsg:  "success",
		},
		{
			name:     "MapData",
			data:     map[string]string{"key": "value"},
			wantCode: 0,
			wantMsg:  "success",
		},
		{
			name:     "NilData",
			data:     nil,
			wantCode: 0,
			wantMsg:  "success",
		},
		{
			name:     "NumericData",
			data:     42,
			wantCode: 0,
			wantMsg:  "success",
		},
		{
			name:     "SliceData",
			data:     []int{1, 2, 3},
			wantCode: 0,
			wantMsg:  "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Success(c, tt.data)

			if w.Code != http.StatusOK {
				t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
			}

			resp := parseResponseBody(t, w)
			if resp.Code != tt.wantCode {
				t.Errorf("Response.Code = %d, want %d", resp.Code, tt.wantCode)
			}
			if resp.Message != tt.wantMsg {
				t.Errorf("Response.Message = %q, want %q", resp.Message, tt.wantMsg)
			}
		})
	}
}

func TestSuccessDataContent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{
		"name": "test",
		"id":   float64(123),
	}
	Success(c, data)

	// Parse the full raw body to check data content.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal(raw["data"], &resultData); err != nil {
		t.Fatalf("failed to unmarshal data field: %v", err)
	}

	if resultData["name"] != "test" {
		t.Errorf("data.name = %v, want %q", resultData["name"], "test")
	}
	if resultData["id"] != float64(123) {
		t.Errorf("data.id = %v, want %v", resultData["id"], float64(123))
	}
}

func TestSuccessNilDataOmitsDataField(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	if _, exists := raw["data"]; exists {
		t.Error("data field should be omitted when nil")
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		wantCode int
		wantMsg  string
	}{
		{
			name:     "Unauthorized",
			code:     bizerr.CodeUnauthorized,
			message:  "unauthorized",
			wantCode: bizerr.CodeUnauthorized,
			wantMsg:  "unauthorized",
		},
		{
			name:     "NotFound",
			code:     bizerr.CodeNotFound,
			message:  "user not found",
			wantCode: bizerr.CodeNotFound,
			wantMsg:  "user not found",
		},
		{
			name:     "Internal",
			code:     bizerr.CodeInternal,
			message:  "something went wrong",
			wantCode: bizerr.CodeInternal,
			wantMsg:  "something went wrong",
		},
		{
			name:     "ParamMissing",
			code:     bizerr.CodeParamMissing,
			message:  "name is required",
			wantCode: bizerr.CodeParamMissing,
			wantMsg:  "name is required",
		},
		{
			name:     "Validation",
			code:     bizerr.CodeValidation,
			message:  "email format is invalid",
			wantCode: bizerr.CodeValidation,
			wantMsg:  "email format is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Error(c, tt.code, tt.message)

			if w.Code != http.StatusOK {
				t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
			}

			resp := parseResponseBody(t, w)
			if resp.Code != tt.wantCode {
				t.Errorf("Response.Code = %d, want %d", resp.Code, tt.wantCode)
			}
			if resp.Message != tt.wantMsg {
				t.Errorf("Response.Message = %q, want %q", resp.Message, tt.wantMsg)
			}
		})
	}
}

func TestErrorHasNoDataField(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, bizerr.CodeInternal, "error")

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	if _, exists := raw["data"]; exists {
		t.Error("error response should not contain a data field")
	}
	if _, exists := raw["meta"]; exists {
		t.Error("error response should not contain a meta field")
	}
}

func TestErrorWithBizErr(t *testing.T) {
	tests := []struct {
		name     string
		bizErr   *bizerr.BizError
		wantCode int
		wantMsg  string
	}{
		{
			name:     "PredefinedUnauthorized",
			bizErr:   bizerr.ErrUnauthorized,
			wantCode: bizerr.CodeUnauthorized,
			wantMsg:  "unauthorized",
		},
		{
			name:     "PredefinedNotFound",
			bizErr:   bizerr.ErrNotFound,
			wantCode: bizerr.CodeNotFound,
			wantMsg:  "resource not found",
		},
		{
			name:     "PredefinedForbidden",
			bizErr:   bizerr.ErrForbidden,
			wantCode: bizerr.CodeForbidden,
			wantMsg:  "forbidden",
		},
		{
			name:     "CustomBizError",
			bizErr:   bizerr.New(bizerr.CodeConflict, "username already taken"),
			wantCode: bizerr.CodeConflict,
			wantMsg:  "username already taken",
		},
		{
			name:     "TokenExpired",
			bizErr:   bizerr.ErrTokenExpired,
			wantCode: bizerr.CodeTokenExpired,
			wantMsg:  "token expired",
		},
		{
			name:     "K8sUnavailable",
			bizErr:   bizerr.ErrK8sUnavailable,
			wantCode: bizerr.CodeK8sUnavailable,
			wantMsg:  "kubernetes cluster unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ErrorWithBizErr(c, tt.bizErr)

			if w.Code != http.StatusOK {
				t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
			}

			resp := parseResponseBody(t, w)
			if resp.Code != tt.wantCode {
				t.Errorf("Response.Code = %d, want %d", resp.Code, tt.wantCode)
			}
			if resp.Message != tt.wantMsg {
				t.Errorf("Response.Message = %q, want %q", resp.Message, tt.wantMsg)
			}
		})
	}
}

func TestSuccessWithMeta(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"item1", "item2"}
	meta := &Meta{
		Source:    "cache",
		Stale:     true,
		Total:     100,
		RequestID: "req-123",
	}

	SuccessWithMeta(c, data, meta)

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse the full response manually to inspect meta.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	// Check code and message.
	var code int
	if err := json.Unmarshal(raw["code"], &code); err != nil {
		t.Fatalf("failed to unmarshal code: %v", err)
	}
	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}

	// Check meta is present.
	if _, exists := raw["meta"]; !exists {
		t.Fatal("meta field should be present")
	}

	var gotMeta Meta
	if err := json.Unmarshal(raw["meta"], &gotMeta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if gotMeta.Source != "cache" {
		t.Errorf("meta.Source = %q, want %q", gotMeta.Source, "cache")
	}
	if !gotMeta.Stale {
		t.Error("meta.Stale = false, want true")
	}
	if gotMeta.Total != 100 {
		t.Errorf("meta.Total = %d, want 100", gotMeta.Total)
	}
	if gotMeta.RequestID != "req-123" {
		t.Errorf("meta.RequestID = %q, want %q", gotMeta.RequestID, "req-123")
	}
}

func TestPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"a", "b", "c"}
	Paginated(c, data, 42, "req-456")

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	// Check code.
	var code int
	if err := json.Unmarshal(raw["code"], &code); err != nil {
		t.Fatalf("failed to unmarshal code: %v", err)
	}
	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}

	// Check meta.
	if _, exists := raw["meta"]; !exists {
		t.Fatal("meta field should be present for paginated response")
	}

	var gotMeta Meta
	if err := json.Unmarshal(raw["meta"], &gotMeta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if gotMeta.Total != 42 {
		t.Errorf("meta.Total = %d, want 42", gotMeta.Total)
	}
	if gotMeta.RequestID != "req-456" {
		t.Errorf("meta.RequestID = %q, want %q", gotMeta.RequestID, "req-456")
	}
}

func TestResponseContentType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, "test")

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json; charset=utf-8")
	}
}
