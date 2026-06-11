package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func TestError_Error(t *testing.T) {
	cause := errors.New("boom")
	tests := []struct {
		name string
		err  *errs.Error
		want string
	}{
		{"op+cause", errs.Validation("svc.Do", "bad input", cause), "svc.Do: bad input: boom"},
		{"op only", errs.NotFound("svc.Get", "missing", nil), "svc.Get: missing"},
		{"cause only", errs.E(errs.KindInternal, "", "wrap", cause), "wrap: boom"},
		{"msg only", errs.E(errs.KindInternal, "", "plain", nil), "plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKindOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want errs.Kind
	}{
		{"nil", nil, errs.KindUnknown},
		{"plain", errors.New("x"), errs.KindInternal},
		{"typed", errs.Conflict("op", "dup", nil), errs.KindConflict},
		{"wrapped typed", fmt.Errorf("ctx: %w", errs.NotFound("op", "no", nil)), errs.KindNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errs.KindOf(tt.err); got != tt.want {
				t.Fatalf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root")
	err := errs.Internal("op", "wrap", cause)
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should find the wrapped cause")
	}
}

func TestKind_String(t *testing.T) {
	for k, want := range map[errs.Kind]string{
		errs.KindValidation:   "validation",
		errs.KindNotFound:     "not_found",
		errs.KindConflict:     "conflict",
		errs.KindUnauthorized: "unauthorized",
		errs.KindUnavailable:  "unavailable",
		errs.KindInternal:     "internal",
		errs.KindUnknown:      "unknown",
	} {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}
