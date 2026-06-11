package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

type flakyStage struct {
	name string
	out  ports.StageResult
	err  error
}

func (f flakyStage) Name() string { return f.name }
func (f flakyStage) Analyze(context.Context, domain.Submission) (ports.StageResult, error) {
	return f.out, f.err
}

func optSub(t *testing.T) domain.Submission {
	t.Helper()
	s, err := domain.NewSubmission("s", "o", "r", "f", "go", "+d", "ide", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOptional_PassesThroughOnSuccess(t *testing.T) {
	inner := flakyStage{name: "semantic", out: ports.StageResult{
		Findings: []domain.Finding{{Stage: "semantic", Rule: "r"}},
	}}
	stage := app.Optional(inner)

	out, err := stage.Analyze(context.Background(), optSub(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Findings) != 1 || out.Findings[0].Rule != "r" {
		t.Fatalf("out = %+v", out)
	}
	if stage.Name() != "semantic" {
		t.Fatalf("name = %s", stage.Name())
	}
}

func TestOptional_ErrorBecomesWarningFinding(t *testing.T) {
	stage := app.Optional(flakyStage{name: "semantic", err: errors.New("service down")})

	out, err := stage.Analyze(context.Background(), optSub(t))
	if err != nil {
		t.Fatal("optional stage must never return an error")
	}
	if len(out.Findings) != 1 {
		t.Fatalf("findings = %+v", out.Findings)
	}
	f := out.Findings[0]
	if f.Rule != "stage-unavailable" || f.Severity != domain.SeverityWarning || f.Stage != "semantic" {
		t.Fatalf("finding = %+v", f)
	}
}
