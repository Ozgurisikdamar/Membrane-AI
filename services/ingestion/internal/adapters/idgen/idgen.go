// Package idgen provides the production IDGen and Clock adapters.
package idgen

import (
	"time"

	"github.com/google/uuid"
)

// UUID generates RFC-4122 v4 identifiers.
type UUID struct{}

// NewID returns a new random UUID string.
func (UUID) NewID() string { return uuid.NewString() }

// SystemClock returns the real wall-clock time in UTC.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
