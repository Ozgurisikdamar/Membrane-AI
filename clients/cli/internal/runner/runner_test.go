package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/runner"
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

func TestRun_FindsSortsAndCounts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "clean.go", "package main\n")
	writeFile(t, root, "auth/creds.go", "key := \"AKIAIOSFODNN7EXAMPLE\"\n")
	writeFile(t, root, "auth/tls.go", "cfg := tls.Config{InsecureSkipVerify: true}\n")

	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesScanned != 3 {
		t.Fatalf("scanned = %d, want 3", res.FilesScanned)
	}
	if len(res.Findings) != 2 || res.Blocking != 1 || res.Warnings != 1 {
		t.Fatalf("findings = %+v (blocking=%d warnings=%d)", res.Findings, res.Blocking, res.Warnings)
	}
	// Sorted by file path: auth/creds.go before auth/tls.go.
	if res.Findings[0].File != "auth/creds.go" || res.Findings[0].Rule != "aws-access-key-id" {
		t.Fatalf("first = %+v", res.Findings[0])
	}
	if res.Findings[1].File != "auth/tls.go" || res.Findings[1].Severity != "warning" {
		t.Fatalf("second = %+v", res.Findings[1])
	}
}

func TestRun_SkipsNoiseDirsBinariesAndBigFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "node_modules/dep.js", "api_key = \"leakedvalue123\"\n")
	writeFile(t, root, ".git/config", "password = \"leakedvalue123\"\n")
	writeFile(t, root, "vendor/lib.go", "key := \"AKIAIOSFODNN7EXAMPLE\"\n")
	writeFile(t, root, "bin.dat", "abc\x00def")
	writeFile(t, root, "ok.go", "x := 1\n")

	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("noise dirs must be skipped: %+v", res.Findings)
	}
	if res.FilesScanned != 1 { // only ok.go
		t.Fatalf("scanned = %d, want 1", res.FilesScanned)
	}
	if res.FilesSkipped != 1 { // bin.dat
		t.Fatalf("skipped = %d, want 1", res.FilesSkipped)
	}
}

func TestRun_SingleFileTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "creds.env", "DB_PASSWORD = \"SuperSecret123!\"\n")

	res, err := runner.Run(context.Background(), runner.Options{Root: filepath.Join(root, "creds.env")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != "generic-assigned-secret" {
		t.Fatalf("findings = %+v", res.Findings)
	}
	// rel is computed against the file's directory — the finding must carry
	// the file name, not "." (which used to label every single-file finding).
	if res.Findings[0].File != "creds.env" {
		t.Fatalf("file = %q, want creds.env", res.Findings[0].File)
	}
	if res.Root != root {
		t.Fatalf("root = %q, want the file's directory %q", res.Root, root)
	}
}

func TestRun_MissingRootErrors(t *testing.T) {
	if _, err := runner.Run(context.Background(), runner.Options{Root: filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Fatal("missing root must error")
	}
}

func TestRun_LanguageScopedRuleNeedsGoFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "tls.py", "cfg = Config(InsecureSkipVerify: true)\n")
	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("go-only rule fired on a python file: %+v", res.Findings)
	}
}
