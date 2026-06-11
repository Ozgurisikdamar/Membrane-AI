package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRun_FlagValidation(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no subcommand", nil},
		{"bad fail-on", []string{"scan", ".", "--fail-on=nope"}},
		{"bad report", []string{"scan", ".", "--report=pdf"}},
		{"json and report", []string{"scan", ".", "--json", "--report=md"}},
		{"out without report", []string{"scan", ".", "--out=x.md"}},
		// A second positional would otherwise silently drop that path AND any
		// flags after it — must be a usage error, not a false-clean scan.
		{"extra positional", []string{"scan", "a", "b"}},
		{"flags after extra positional", []string{"scan", "a", "b", "--report=md"}},
	}
	for _, c := range cases {
		if got := run(c.args); got != exitUsage {
			t.Errorf("%s: exit=%d want %d", c.name, got, exitUsage)
		}
	}
}

func TestRun_ReportToFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "creds.go", "key := \"AKIAIOSFODNN7EXAMPLE\"\n")
	out := filepath.Join(t.TempDir(), "debt.html")

	// Findings exist → exit 1 under the default --fail-on=blocking, but the
	// report must still be written.
	if got := run([]string{"scan", root, "--report=html", "--out", out}); got != exitFail {
		t.Fatalf("exit=%d want %d", got, exitFail)
	}
	html, err := os.ReadFile(out) //nolint:gosec // test-owned path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "aws-access-key-id") {
		t.Fatal("report missing the finding")
	}

	// --fail-on=never keeps the exit code clean for lead-magnet runs.
	if got := run([]string{"scan", root, "--report=md", "--out", out, "--fail-on=never"}); got != exitClean {
		t.Fatalf("exit=%d want %d", got, exitClean)
	}
	md, err := os.ReadFile(out) //nolint:gosec // test-owned path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(md), "# Generative-AI Technical-Debt Report") {
		t.Fatal("markdown report missing title")
	}
}
