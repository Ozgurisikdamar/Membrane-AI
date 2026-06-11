// Package config loads 12-factor configuration from environment variables.
// A Loader accumulates missing-required keys and parse errors so startup can
// fail fast with a single, complete message (see ENGINEERING-STANDARDS §5).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Loader reads keys prefixed with "<PREFIX>_". Construct one, read all fields,
// then call Err() exactly once before using the values.
type Loader struct {
	prefix  string
	missing []string
	errs    []error
}

// NewLoader returns a Loader that prefixes every key with prefix + "_"
// (e.g. NewLoader("MEMBRANE_INGESTION").String("HTTP_ADDR", …) reads
// MEMBRANE_INGESTION_HTTP_ADDR).
func NewLoader(prefix string) *Loader {
	return &Loader{prefix: strings.TrimRight(prefix, "_")}
}

func (l *Loader) key(name string) string {
	if l.prefix == "" {
		return name
	}
	return l.prefix + "_" + name
}

func (l *Loader) lookup(name string) (string, bool) {
	v, ok := os.LookupEnv(l.key(name))
	if !ok {
		return "", false
	}
	return v, true
}

// String returns the value or def when unset/empty.
func (l *Loader) String(name, def string) string {
	if v, ok := l.lookup(name); ok && v != "" {
		return v
	}
	return def
}

// Required returns the value or records the key as missing (Err will report it).
func (l *Loader) Required(name string) string {
	if v, ok := l.lookup(name); ok && v != "" {
		return v
	}
	l.missing = append(l.missing, l.key(name))
	return ""
}

// Int parses an int or records a parse error; returns def when unset.
func (l *Loader) Int(name string, def int) int {
	v, ok := l.lookup(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: invalid int %q", l.key(name), v))
		return def
	}
	return n
}

// Duration parses a Go duration (e.g. "1200ms") or records a parse error.
func (l *Loader) Duration(name string, def time.Duration) time.Duration {
	v, ok := l.lookup(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: invalid duration %q", l.key(name), v))
		return def
	}
	return d
}

// Bool parses 1/t/true/yes/on (case-insensitive) as true; records a parse error
// on anything else; returns def when unset.
func (l *Loader) Bool(name string, def bool) bool {
	v, ok := l.lookup(name)
	if !ok || v == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "t", "true", "yes", "on":
		return true
	case "0", "f", "false", "no", "off":
		return false
	default:
		l.errs = append(l.errs, fmt.Errorf("%s: invalid bool %q", l.key(name), v))
		return def
	}
}

// Err returns a single aggregated error if any required key was missing or any
// value failed to parse, else nil.
func (l *Loader) Err() error {
	if len(l.missing) == 0 && len(l.errs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("config error:")
	if len(l.missing) > 0 {
		fmt.Fprintf(&b, " missing required [%s]", strings.Join(l.missing, ", "))
	}
	for _, e := range l.errs {
		fmt.Fprintf(&b, "; %v", e)
	}
	return errors.New(b.String())
}
