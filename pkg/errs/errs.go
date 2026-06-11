// Package errs defines typed, transport-agnostic domain errors shared across
// MEMBRANE.AI services. Domain and application layers return these; adapters map
// them to gRPC/HTTP status codes at the boundary (see ENGINEERING-STANDARDS §5).
package errs

import (
	"errors"
	"fmt"
)

// Kind classifies an error so adapters can map it to a transport status without
// inspecting messages.
type Kind int

const (
	// KindUnknown is the zero value; treat as an internal error.
	KindUnknown Kind = iota
	// KindValidation means the input failed a business/contract rule.
	KindValidation
	// KindNotFound means a requested resource does not exist.
	KindNotFound
	// KindConflict means the request conflicts with current state.
	KindConflict
	// KindUnauthorized means authentication/authorization failed.
	KindUnauthorized
	// KindUnavailable means a dependency is temporarily unavailable (retryable).
	KindUnavailable
	// KindInternal means an unexpected server-side failure.
	KindInternal
)

// String renders the kind for logs and messages.
func (k Kind) String() string {
	switch k {
	case KindValidation:
		return "validation"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindUnauthorized:
		return "unauthorized"
	case KindUnavailable:
		return "unavailable"
	case KindInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// Error is the typed error used across the codebase. Op is a stable operation
// name (e.g. "ingestion.EnqueueSubmission") that builds a breadcrumb trail.
type Error struct {
	Kind Kind
	Op   string
	Msg  string
	Err  error
}

// Error implements the error interface, including the wrapped cause when present.
func (e *Error) Error() string {
	switch {
	case e.Op != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
	case e.Op != "":
		return fmt.Sprintf("%s: %s", e.Op, e.Msg)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	default:
		return e.Msg
	}
}

// Unwrap exposes the wrapped cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.Err }

// E builds a typed error. cause may be nil.
func E(kind Kind, op, msg string, cause error) *Error {
	return &Error{Kind: kind, Op: op, Msg: msg, Err: cause}
}

// Validation builds a KindValidation error.
func Validation(op, msg string, cause error) *Error { return E(KindValidation, op, msg, cause) }

// NotFound builds a KindNotFound error.
func NotFound(op, msg string, cause error) *Error { return E(KindNotFound, op, msg, cause) }

// Conflict builds a KindConflict error.
func Conflict(op, msg string, cause error) *Error { return E(KindConflict, op, msg, cause) }

// Unauthorized builds a KindUnauthorized error.
func Unauthorized(op, msg string, cause error) *Error { return E(KindUnauthorized, op, msg, cause) }

// Unavailable builds a KindUnavailable error.
func Unavailable(op, msg string, cause error) *Error { return E(KindUnavailable, op, msg, cause) }

// Internal builds a KindInternal error.
func Internal(op, msg string, cause error) *Error { return E(KindInternal, op, msg, cause) }

// KindOf walks the error chain and returns the first typed Kind it finds, or
// KindInternal for any non-nil untyped error, or KindUnknown for nil.
func KindOf(err error) Kind {
	if err == nil {
		return KindUnknown
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}
