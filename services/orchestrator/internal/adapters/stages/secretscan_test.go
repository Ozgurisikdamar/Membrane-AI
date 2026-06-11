package stages_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/stages"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

func scanDiff(t *testing.T, diff string) []domain.Finding {
	t.Helper()
	sub, err := domain.NewSubmission("s", "o", "r", "f", "go", diff, "ide", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	got, err := stages.NewSecretScan().Analyze(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	return got.Findings
}

func TestSecretScan_Detects(t *testing.T) {
	tests := []struct {
		name, diff, wantRule string
	}{
		{"private key", "+-----BEGIN RSA PRIVATE KEY-----", "private-key-block"},
		{"aws key id", `+const key = "AKIAIOSFODNN7EXAMPLE"`, "aws-access-key-id"},
		{"github token", "+token := \"ghp_abcdefghijklmnopqrstuvwxyz1234\"", "github-token"},
		{"assigned password", `+db_password = "SuperSecret123!"`, "generic-assigned-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanDiff(t, tt.diff)
			if len(findings) == 0 {
				t.Fatalf("expected a finding for %q", tt.diff)
			}
			if findings[0].Rule != tt.wantRule || findings[0].Severity != domain.SeverityBlocking {
				t.Fatalf("finding = %+v, want rule %s (blocking)", findings[0], tt.wantRule)
			}
		})
	}
}

func TestSecretScan_CleanDiffHasNoFindings(t *testing.T) {
	clean := "+func Add(a, b int) int { return a + b }\n+// the password prompt is shown to the user"
	if findings := scanDiff(t, clean); len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestSecretScan_HonorsContextCancellation(t *testing.T) {
	sub, _ := domain.NewSubmission("s", "o", "r", "f", "go", "d", "ide", time.Now())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stages.NewSecretScan().Analyze(ctx, sub); err == nil {
		t.Fatal("expected context error")
	}
}
