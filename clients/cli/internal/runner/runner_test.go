package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestRun_ExcerptIsTheMaskedSourceLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "auth/creds.go", "package auth\n\nkey := \"AKIAIOSFODNN7EXAMPLE\"\n")
	// A risky-pattern line that also carries a credential: both findings must
	// quote the line with the credential already redacted.
	writeFile(t, root, "db/keys.go",
		"rows, err := db.Query(\"SELECT * FROM keys WHERE id = 'AKIAIOSFODNN7EXAMPLE' AND owner = \" + owner)\n")
	writeFile(t, root, "cfg.env", "\tDB_PASSWORD = \"SuperSecret123!\"\r\n")

	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"auth/creds.go:aws-access-key-id": `key := "[MASKED:aws-access-key-id]"`,
		"db/keys.go:aws-access-key-id":    `rows, err := db.Query("SELECT * FROM keys WHERE id = '[MASKED:aws-access-key-id]' AND owner = " + owner)`,
		"db/keys.go:sql-string-concat":    `rows, err := db.Query("SELECT * FROM keys WHERE id = '[MASKED:aws-access-key-id]' AND owner = " + owner)`,
		// Tab and CR neutralized and trimmed; the whole assignment is one match.
		"cfg.env:generic-assigned-secret": `DB_[MASKED:generic-assigned-secret]`,
	}
	if len(res.Findings) != len(want) {
		t.Fatalf("findings = %+v", res.Findings)
	}
	for _, f := range res.Findings {
		key := f.File + ":" + f.Rule
		if got, ok := want[key]; !ok || f.Excerpt != got {
			t.Errorf("%s excerpt = %q, want %q", key, f.Excerpt, got)
		}
		for _, raw := range []string{"AKIAIOSFODNN7EXAMPLE", "SuperSecret123!"} {
			if strings.Contains(f.Excerpt, raw) {
				t.Errorf("%s excerpt leaks %q", key, raw)
			}
		}
	}
}

func TestRun_ExcerptIsCappedOnRuneBoundary(t *testing.T) {
	root := t.TempDir()
	long := `db.Query("SELECT 1" + x) // ` + strings.Repeat("ğ", 300)
	writeFile(t, root, "min.js", long+"\n")

	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 1 {
		t.Fatalf("findings = %+v", res.Findings)
	}
	ex := res.Findings[0].Excerpt
	if n := utf8.RuneCountInString(ex); n != runner.MaxExcerptRunes+1 || !strings.HasSuffix(ex, "…") {
		t.Fatalf("excerpt runes = %d (want %d incl. ellipsis): %q", n, runner.MaxExcerptRunes+1, ex)
	}
	if !utf8.ValidString(ex) {
		t.Fatal("excerpt cut inside a multi-byte rune")
	}
}
