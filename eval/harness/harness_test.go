package harness_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/eval/harness"
)

func TestEvaluate_ScoresCorpus(t *testing.T) {
	corpus := harness.Corpus{Cases: []harness.Case{
		{Name: "tp", Language: "go", Content: `k := "AKIAIOSFODNN7EXAMPLE"`, Expect: []string{"aws-access-key-id"}},
		{Name: "fn", Language: "go", Content: "x := 1", Expect: []string{"sql-string-concat"}},
		{Name: "clean-ok", Language: "go", Content: "y := 2", Expect: nil},
		{Name: "clean-fp", Language: "go", Content: `p := "AKIAIOSFODNN7EXAMPLE"`, Expect: nil},
	}}

	m, err := harness.Evaluate(context.Background(), corpus)
	if err != nil {
		t.Fatal(err)
	}
	// tp: 1 TP. fn: 1 FN (expected sql, found none). clean-fp: 1 FP (aws found, none expected).
	if m.TruePositives != 1 || m.FalseNegatives != 1 || m.FalsePositives != 1 {
		t.Fatalf("TP=%d FP=%d FN=%d", m.TruePositives, m.FalsePositives, m.FalseNegatives)
	}
	if m.Precision != 0.5 || m.Recall != 0.5 {
		t.Fatalf("precision=%v recall=%v, want 0.5/0.5", m.Precision, m.Recall)
	}
	if m.CleanCases != 2 || m.FalsePositiveRate != 0.5 {
		t.Fatalf("clean=%d fp-rate=%v, want 2/0.5", m.CleanCases, m.FalsePositiveRate)
	}
}

func TestEvaluate_LanguageScopingIsRespected(t *testing.T) {
	// InsecureSkipVerify is a Go-only rule; in python it must NOT fire (clean).
	corpus := harness.Corpus{Cases: []harness.Case{
		{Name: "go", Language: "go", Content: "tls.Config{InsecureSkipVerify: true}", Expect: []string{"insecure-tls-skip-verify"}},
		{Name: "py", Language: "python", Content: "Config(InsecureSkipVerify: true)", Expect: nil},
	}}
	m, err := harness.Evaluate(context.Background(), corpus)
	if err != nil {
		t.Fatal(err)
	}
	if m.TruePositives != 1 || m.FalsePositives != 0 || m.FalseNegatives != 0 {
		t.Fatalf("TP=%d FP=%d FN=%d", m.TruePositives, m.FalsePositives, m.FalseNegatives)
	}
}

// TestGoldenCorpus_MeetsSLO is the real gate: the shipped corpus must score
// perfectly against the current detectors (precision/recall 1.0, no clean FPs).
func TestGoldenCorpus_MeetsSLO(t *testing.T) {
	corpus, err := harness.LoadCorpus(filepath.Join("..", "corpus", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := harness.Evaluate(context.Background(), corpus)
	if err != nil {
		t.Fatal(err)
	}
	if m.Precision != 1.0 {
		t.Errorf("precision=%.3f, want 1.0 — false positives: %+v", m.Precision, falsePositives(m))
	}
	if m.Recall != 1.0 {
		t.Errorf("recall=%.3f, want 1.0 — false negatives: %+v", m.Recall, falseNegatives(m))
	}
	if m.FalsePositiveRate != 0 {
		t.Errorf("fp-rate=%.3f, want 0", m.FalsePositiveRate)
	}
}

func falsePositives(m harness.Metrics) map[string][]string {
	out := map[string][]string{}
	for _, r := range m.Results {
		if len(r.Unexpected) > 0 {
			out[r.Name] = r.Unexpected
		}
	}
	return out
}

func falseNegatives(m harness.Metrics) map[string][]string {
	out := map[string][]string{}
	for _, r := range m.Results {
		if len(r.Missing) > 0 {
			out[r.Name] = r.Missing
		}
	}
	return out
}
