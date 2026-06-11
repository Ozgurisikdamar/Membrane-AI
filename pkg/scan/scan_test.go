package scan_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

func TestSecretDetector_FindsWithLineNumbers(t *testing.T) {
	content := "+func ok() {}\n+key := \"AKIAIOSFODNN7EXAMPLE\"\n+db_password = \"SuperSecret123!\""
	findings, err := scan.NewSecretDetector().Detect(context.Background(), content, "go")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %+v, want 2", findings)
	}
	if findings[0].Rule != "aws-access-key-id" || findings[0].Line != 2 {
		t.Fatalf("first = %+v", findings[0])
	}
	if findings[1].Rule != "generic-assigned-secret" || findings[1].Line != 3 {
		t.Fatalf("second = %+v", findings[1])
	}
	for _, f := range findings {
		if f.Severity != scan.SeverityBlocking {
			t.Fatalf("severity = %v, want blocking", f.Severity)
		}
	}
}

func TestSecretDetector_DetectsAllPatternKinds(t *testing.T) {
	tests := []struct {
		name, line, wantRule string
	}{
		{"private key", "-----BEGIN RSA PRIVATE KEY-----", "private-key-block"},
		{"aws", `key = "AKIAIOSFODNN7EXAMPLE"`, "aws-access-key-id"},
		{"github", `t := "ghp_abcdefghijklmnopqrstuvwxyz1234"`, "github-token"},
		{"slack", `t := "xoxb-1234567890-abcdef"`, "slack-token"},
		{"assigned", `api_key = "supersecretvalue1"`, "generic-assigned-secret"},
	}
	det := scan.NewSecretDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := det.Detect(context.Background(), tt.line, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) == 0 || findings[0].Rule != tt.wantRule {
				t.Fatalf("findings = %+v, want rule %s", findings, tt.wantRule)
			}
		})
	}
}

func TestSecretDetector_Mask(t *testing.T) {
	content := `key := "AKIAIOSFODNN7EXAMPLE"` + "\nclean line"
	masked := scan.NewSecretDetector().Mask(content)
	if strings.Contains(masked, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("secret survived masking: %s", masked)
	}
	if !strings.Contains(masked, "[MASKED:aws-access-key-id]") {
		t.Fatalf("placeholder missing: %s", masked)
	}
	if !strings.Contains(masked, "clean line") {
		t.Fatal("masking must not alter clean lines")
	}
}

func TestRiskyPatternDetector_LanguageScoping(t *testing.T) {
	tests := []struct {
		name, language, line, wantRule string
		wantCount                      int
	}{
		{"sql concat any language", "python", `cur.execute("SELECT 1 WHERE id=" + uid)`, "sql-string-concat", 1},
		{"go exec concat", "go", `exec.Command("sh", "-c", "echo "+x)`, "exec-command-concat", 1},
		{"go tls", "GO", `tls.Config{InsecureSkipVerify: true}`, "insecure-tls-skip-verify", 1},
		{"go rule skipped for python", "python", `tls.Config{InsecureSkipVerify: true}`, "", 0},
		{"clean", "go", "x := 1", "", 0},
	}
	det := scan.NewRiskyPatternDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := det.Detect(context.Background(), tt.line, tt.language)
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != tt.wantCount {
				t.Fatalf("findings = %+v, want %d", findings, tt.wantCount)
			}
			if tt.wantCount > 0 && findings[0].Rule != tt.wantRule {
				t.Fatalf("rule = %s, want %s", findings[0].Rule, tt.wantRule)
			}
		})
	}
}

func TestRun_CollectsAndMasks(t *testing.T) {
	content := "key := \"AKIAIOSFODNN7EXAMPLE\"\ntls.Config{InsecureSkipVerify: true}"
	findings, masked, err := scan.Run(context.Background(), content, "go", scan.Default()...)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %+v, want 2", findings)
	}
	if strings.Contains(masked, "AKIA") {
		t.Fatalf("masked content leaked secret: %s", masked)
	}
	if !strings.Contains(masked, "InsecureSkipVerify") {
		t.Fatal("risky patterns are flagged, not masked")
	}
}

func TestDetectors_HonorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scan.NewSecretDetector().Detect(ctx, "x", ""); err == nil {
		t.Fatal("secret detector must honor cancellation")
	}
	if _, err := scan.NewRiskyPatternDetector().Detect(ctx, "x", "go"); err == nil {
		t.Fatal("risky detector must honor cancellation")
	}
	if _, _, err := scan.Run(ctx, "x", "go", scan.Default()...); err == nil {
		t.Fatal("Run must propagate cancellation")
	}
}
