package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

type fakeDetector struct {
	name     string
	findings []domain.Finding
	err      error
}

func (f fakeDetector) Name() string { return f.name }
func (f fakeDetector) Detect(context.Context, domain.Input) ([]domain.Finding, error) {
	return f.findings, f.err
}

func mustInput(t *testing.T, diff string) domain.Input {
	t.Helper()
	in, err := domain.NewInput("s", "o", "r", "f", "go", diff)
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func TestHandle_CollectsAcrossDetectors(t *testing.T) {
	uc := app.NewAnalyzeDiff(
		fakeDetector{name: "a", findings: []domain.Finding{{Rule: "r1"}}},
		fakeDetector{name: "b", findings: []domain.Finding{{Rule: "r2"}, {Rule: "r3"}}},
	)
	res, err := uc.Handle(context.Background(), mustInput(t, "+x"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 3 {
		t.Fatalf("findings = %+v", res.Findings)
	}
	if res.MaskedDiff != "+x" {
		t.Fatalf("masked diff changed without a masker: %q", res.MaskedDiff)
	}
}

func TestHandle_DetectorErrorAborts(t *testing.T) {
	uc := app.NewAnalyzeDiff(fakeDetector{name: "bad", err: errors.New("boom")})
	_, err := uc.Handle(context.Background(), mustInput(t, "+x"))
	if errs.KindOf(err) != errs.KindInternal {
		t.Fatalf("kind = %v, want internal", errs.KindOf(err))
	}
}

func TestHandle_RealDetectorsEndToEnd(t *testing.T) {
	uc := app.NewAnalyzeDiff(domain.NewSecretDetector(), domain.NewRiskyPatternDetector())
	diff := "+key := \"AKIAIOSFODNN7EXAMPLE\"\n+cfg := &tls.Config{InsecureSkipVerify: true}"
	res, err := uc.Handle(context.Background(), mustInput(t, diff))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 2 {
		t.Fatalf("findings = %+v, want 2", res.Findings)
	}
	if strings.Contains(res.MaskedDiff, "AKIA") {
		t.Fatalf("secret survived masking: %s", res.MaskedDiff)
	}
	if !strings.Contains(res.MaskedDiff, "InsecureSkipVerify") {
		t.Fatal("risky pattern detector must not mask (only secrets are masked)")
	}
}
