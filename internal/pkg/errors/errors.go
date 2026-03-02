package errors

import "fmt"

// Business error codes
const (
	CodeSuccess        = 0
	CodeParamMissing   = 40001
	CodeParamInvalid   = 40002
	CodeUnauthorized   = 40100
	CodeTokenExpired   = 40101
	Code2FARequired    = 40102
	CodeForbidden      = 40300
	CodeNotFound       = 40400
	CodeConflict       = 40900
	CodeValidation     = 42200
	CodeInternal       = 50000
	CodeK8sUnavailable = 50200
)

// default messages for each error code
var codeMessages = map[int]string{
	CodeSuccess:        "success",
	CodeParamMissing:   "missing required parameter",
	CodeParamInvalid:   "invalid parameter",
	CodeUnauthorized:   "unauthorized",
	CodeTokenExpired:   "token expired",
	Code2FARequired:    "two-factor authentication required",
	CodeForbidden:      "forbidden",
	CodeNotFound:       "resource not found",
	CodeConflict:       "resource conflict",
	CodeValidation:     "validation failed",
	CodeInternal:       "internal server error",
	CodeK8sUnavailable: "kubernetes cluster unavailable",
}

// BizError represents a business logic error with a code and message.
type BizError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *BizError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// New creates a new BizError with the given code and message.
func New(code int, message string) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}

// NewWithCode creates a new BizError with the given code and default message.
func NewWithCode(code int) *BizError {
	msg, ok := codeMessages[code]
	if !ok {
		msg = "unknown error"
	}
	return &BizError{
		Code:    code,
		Message: msg,
	}
}

// Wrap wraps an existing error with a business error code.
func Wrap(code int, err error) *BizError {
	if err == nil {
		return nil
	}
	return &BizError{
		Code:    code,
		Message: err.Error(),
	}
}

// Is checks if the given error is a BizError with the specified code.
func Is(err error, code int) bool {
	if bizErr, ok := err.(*BizError); ok {
		return bizErr.Code == code
	}
	return false
}

// Predefined common errors for convenience.
var (
	ErrParamMissing   = NewWithCode(CodeParamMissing)
	ErrParamInvalid   = NewWithCode(CodeParamInvalid)
	ErrUnauthorized   = NewWithCode(CodeUnauthorized)
	ErrTokenExpired   = NewWithCode(CodeTokenExpired)
	Err2FARequired    = NewWithCode(Code2FARequired)
	ErrForbidden      = NewWithCode(CodeForbidden)
	ErrNotFound       = NewWithCode(CodeNotFound)
	ErrConflict       = NewWithCode(CodeConflict)
	ErrValidation     = NewWithCode(CodeValidation)
	ErrInternal       = NewWithCode(CodeInternal)
	ErrK8sUnavailable = NewWithCode(CodeK8sUnavailable)
)
