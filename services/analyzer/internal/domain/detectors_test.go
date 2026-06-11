package domain_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

func input(t *testing.T, language, diff string) domain.Input {
	t.Helper()
	in, err := domain.NewInput("s", "o", "r", "f", language, diff)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func TestNewInput_Validation(t *testing.T) {
	_, err := domain.NewInput("s", "o", "r", "f", "go", "")
	if errs.KindOf(err) != errs.KindValidation {
		t.Fatalf("kind = %v, want validation", errs.KindOf(err))
	}
	in, err := domain.NewInput("s", "o", "r", "f", " Go ", "d")
	if err != nil {
		t.Fatal(err)
	}
	if in.Language != "go" {
		t.Fatalf("language not normalized: %q", in.Language)
	}
}

func TestSecretDetector_FindsWithLineNumbers(t *testing.T) {
	diff := "+func ok() {}\n+key := \"AKIAIOSFODNN7EXAMPLE\"\n+db_password = \"SuperSecret123!\""
	findings, err := domain.NewSecretDetector().Detect(context.Background(), input(t, "go", diff))
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
		if f.Severity != domain.SeverityBlocking {
			t.Fatalf("severity = %v, want blocking", f.Severity)
		}
	}
}

func TestSecretDetector_Mask(t *testing.T) {
	diff := `+key := "AKIAIOSFODNN7EXAMPLE"` + "\n+clean line"
	masked := domain.NewSecretDetector().Mask(diff)
	if strings.Contains(masked, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("secret survived masking: %s", masked)
	}
	if !strings.Contains(masked, "[MASKED:aws-access-key-id]") {
		t.Fatalf("placeholder missing: %s", masked)
	}
	if !strings.Contains(masked, "+clean line") {
		t.Fatal("masking must not alter clean lines")
	}
}

func TestRiskyPatternDetector(t *testing.T) {
	tests := []struct {
		name, language, diff, wantRule string
		wantCount                      int
	}{
		{"sql concat any language", "python",
			`+cur.execute("SELECT * FROM users WHERE id=" + uid)`, "sql-string-concat", 1},
		{"go exec concat", "go",
			`+cmd := exec.Command("sh", "-c", "echo "+userInput)`, "exec-command-concat", 1},
		{"go insecure tls", "go",
			`+cfg := &tls.Config{InsecureSkipVerify: true}`, "insecure-tls-skip-verify", 1},
		{"go rule skipped for python", "python",
			`+cfg := &tls.Config{InsecureSkipVerify: true}`, "", 0},
		{"clean", "go", "+x := 1", "", 0},
	}
	det := domain.NewRiskyPatternDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := det.Detect(context.Background(), input(t, tt.language, tt.diff))
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

func TestDetectors_HonorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	in := input(t, "go", "+x")
	if _, err := domain.NewSecretDetector().Detect(ctx, in); err == nil {
		t.Fatal("secret detector must honor cancellation")
	}
	if _, err := domain.NewRiskyPatternDetector().Detect(ctx, in); err == nil {
		t.Fatal("risky detector must honor cancellation")
	}
}
