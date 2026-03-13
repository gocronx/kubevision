package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

// Response is the unified JSON response structure.
// HTTP status code is always 200; business status is indicated by Code.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta carries optional metadata for paginated or enriched responses.
type Meta struct {
	Source    string `json:"source,omitempty"`
	Stale     bool   `json:"stale,omitempty"`
	Total     int64  `json:"total,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// Success sends a successful response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    bizerr.CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMeta sends a successful response with data and metadata.
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Code:    bizerr.CodeSuccess,
		Message: "success",
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error response with the given business code and message.
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithBizErr sends an error response derived from a BizError.
func ErrorWithBizErr(c *gin.Context, err *bizerr.BizError) {
	c.JSON(http.StatusOK, Response{
		Code:    err.Code,
		Message: err.Message,
	})
}

// Paginated sends a successful paginated response.
func Paginated(c *gin.Context, data interface{}, total int64, requestID string) {
	c.JSON(http.StatusOK, Response{
		Code:    bizerr.CodeSuccess,
		Message: "success",
		Data:    data,
		Meta: &Meta{
			Total:     total,
			RequestID: requestID,
		},
	})
}
