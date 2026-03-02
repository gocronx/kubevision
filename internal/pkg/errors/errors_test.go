package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestBizErrorImplementsErrorInterface(t *testing.T) {
	var _ error = (*BizError)(nil)

	bizErr := New(CodeInternal, "something went wrong")
	var err error = bizErr
	if err == nil {
		t.Fatal("BizError should implement error interface")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
	}{
		{"CustomError", 99999, "custom error message"},
		{"UnauthorizedCustom", CodeUnauthorized, "you are not logged in"},
		{"EmptyMessage", CodeInternal, ""},
		{"ZeroCode", 0, "no error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message)
			if err.Code != tt.code {
				t.Errorf("Code = %d, want %d", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}
		})
	}
}

func TestNewWithCode(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		wantMessage string
	}{
		{"Success", CodeSuccess, "success"},
		{"ParamMissing", CodeParamMissing, "missing required parameter"},
		{"ParamInvalid", CodeParamInvalid, "invalid parameter"},
		{"Unauthorized", CodeUnauthorized, "unauthorized"},
		{"TokenExpired", CodeTokenExpired, "token expired"},
		{"2FARequired", Code2FARequired, "two-factor authentication required"},
		{"Forbidden", CodeForbidden, "forbidden"},
		{"NotFound", CodeNotFound, "resource not found"},
		{"Conflict", CodeConflict, "resource conflict"},
		{"Validation", CodeValidation, "validation failed"},
		{"Internal", CodeInternal, "internal server error"},
		{"K8sUnavailable", CodeK8sUnavailable, "kubernetes cluster unavailable"},
		{"UnknownCode", 12345, "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWithCode(tt.code)
			if err.Code != tt.code {
				t.Errorf("Code = %d, want %d", err.Code, tt.code)
			}
			if err.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", err.Message, tt.wantMessage)
			}
		})
	}
}

func TestBizErrorErrorMethod(t *testing.T) {
	tests := []struct {
		name string
		err  *BizError
		want string
	}{
		{
			name: "Unauthorized",
			err:  New(CodeUnauthorized, "unauthorized"),
			want: "[40100] unauthorized",
		},
		{
			name: "NotFound",
			err:  New(CodeNotFound, "resource not found"),
			want: "[40400] resource not found",
		},
		{
			name: "CustomCode",
			err:  New(12345, "custom message"),
			want: "[12345] custom message",
		},
		{
			name: "ZeroCode",
			err:  New(0, "success"),
			want: "[0] success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         *BizError
		wantCode    int
		wantMessage string
	}{
		{"ErrParamMissing", ErrParamMissing, CodeParamMissing, "missing required parameter"},
		{"ErrParamInvalid", ErrParamInvalid, CodeParamInvalid, "invalid parameter"},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized, "unauthorized"},
		{"ErrTokenExpired", ErrTokenExpired, CodeTokenExpired, "token expired"},
		{"Err2FARequired", Err2FARequired, Code2FARequired, "two-factor authentication required"},
		{"ErrForbidden", ErrForbidden, CodeForbidden, "forbidden"},
		{"ErrNotFound", ErrNotFound, CodeNotFound, "resource not found"},
		{"ErrConflict", ErrConflict, CodeConflict, "resource conflict"},
		{"ErrValidation", ErrValidation, CodeValidation, "validation failed"},
		{"ErrInternal", ErrInternal, CodeInternal, "internal server error"},
		{"ErrK8sUnavailable", ErrK8sUnavailable, CodeK8sUnavailable, "kubernetes cluster unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", tt.err.Code, tt.wantCode)
			}
			if tt.err.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.wantMessage)
			}
		})
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		want bool
	}{
		{
			name: "MatchingBizError",
			err:  New(CodeUnauthorized, "unauthorized"),
			code: CodeUnauthorized,
			want: true,
		},
		{
			name: "NonMatchingBizError",
			err:  New(CodeUnauthorized, "unauthorized"),
			code: CodeNotFound,
			want: false,
		},
		{
			name: "PredefinedErrorMatches",
			err:  ErrNotFound,
			code: CodeNotFound,
			want: true,
		},
		{
			name: "PredefinedErrorNoMatch",
			err:  ErrNotFound,
			code: CodeInternal,
			want: false,
		},
		{
			name: "StandardErrorReturnsFalse",
			err:  fmt.Errorf("standard error"),
			code: CodeInternal,
			want: false,
		},
		{
			name: "CustomMessageSameCode",
			err:  New(CodeNotFound, "user not found"),
			code: CodeNotFound,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is(tt.err, tt.code)
			if got != tt.want {
				t.Errorf("Is(%v, %d) = %v, want %v", tt.err, tt.code, got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Run("WrapNilError", func(t *testing.T) {
		result := Wrap(CodeInternal, nil)
		if result != nil {
			t.Errorf("Wrap(nil) = %v, want nil", result)
		}
	})

	t.Run("WrapStandardError", func(t *testing.T) {
		stdErr := errors.New("connection refused")
		result := Wrap(CodeK8sUnavailable, stdErr)
		if result == nil {
			t.Fatal("Wrap should return non-nil BizError for non-nil error")
		}
		if result.Code != CodeK8sUnavailable {
			t.Errorf("Code = %d, want %d", result.Code, CodeK8sUnavailable)
		}
		if result.Message != "connection refused" {
			t.Errorf("Message = %q, want %q", result.Message, "connection refused")
		}
	})

	t.Run("WrapBizError", func(t *testing.T) {
		original := New(CodeNotFound, "not found")
		result := Wrap(CodeInternal, original)
		if result.Code != CodeInternal {
			t.Errorf("Code = %d, want %d", result.Code, CodeInternal)
		}
		// The message should be the Error() output of the original.
		if result.Message != "[40400] not found" {
			t.Errorf("Message = %q, want %q", result.Message, "[40400] not found")
		}
	})
}

func TestErrorCodes(t *testing.T) {
	// Verify exact numeric values of all error codes.
	tests := []struct {
		name string
		code int
		want int
	}{
		{"CodeSuccess", CodeSuccess, 0},
		{"CodeParamMissing", CodeParamMissing, 40001},
		{"CodeParamInvalid", CodeParamInvalid, 40002},
		{"CodeUnauthorized", CodeUnauthorized, 40100},
		{"CodeTokenExpired", CodeTokenExpired, 40101},
		{"Code2FARequired", Code2FARequired, 40102},
		{"CodeForbidden", CodeForbidden, 40300},
		{"CodeNotFound", CodeNotFound, 40400},
		{"CodeConflict", CodeConflict, 40900},
		{"CodeValidation", CodeValidation, 42200},
		{"CodeInternal", CodeInternal, 50000},
		{"CodeK8sUnavailable", CodeK8sUnavailable, 50200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}
