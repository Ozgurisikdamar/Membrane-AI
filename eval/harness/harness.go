// Package harness evaluates the deterministic pkg/scan detectors against a
// labeled corpus and computes precision / recall / F1 / false-positive-rate —
// the accuracy SLO gate for the analysis pipeline (report §Accuracy & eval).
//
// Scoring is rule-level: a case lists the rule IDs it SHOULD trigger; matched
// rules are true positives, expected-but-missing are false negatives, and
// found-but-unexpected are false positives. A case with no expected rules is a
// "clean" negative; the share of clean cases that produce any finding is the
// case-level false-positive rate.
package harness

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

// Case is one labeled corpus sample.
type Case struct {
	Name     string   `json:"name"`
	Language string   `json:"language"`
	Content  string   `json:"content"`
	Expect   []string `json:"expect"` // expected rule IDs; empty ⇒ clean negative
}

// Corpus is the full labeled set.
type Corpus struct {
	Cases []Case `json:"cases"`
}

// LoadCorpus reads a JSON corpus file.
func LoadCorpus(path string) (Corpus, error) {
	data, err := os.ReadFile(path) //nolint:gosec // operator-supplied corpus path is the tool's job
	if err != nil {
		return Corpus{}, err
	}
	var c Corpus
	if err := json.Unmarshal(data, &c); err != nil {
		return Corpus{}, err
	}
	return c, nil
}

// CaseResult is the per-case scoring detail.
type CaseResult struct {
	Name       string   `json:"name"`
	Expected   []string `json:"expected"`
	Found      []string `json:"found"`
	Missing    []string `json:"missing"`    // false negatives
	Unexpected []string `json:"unexpected"` // false positives
}

// Metrics is the aggregate accuracy report.
type Metrics struct {
	Cases             int          `json:"cases"`
	CleanCases        int          `json:"clean_cases"`
	TruePositives     int          `json:"true_positives"`
	FalsePositives    int          `json:"false_positives"`
	FalseNegatives    int          `json:"false_negatives"`
	Precision         float64      `json:"precision"`
	Recall            float64      `json:"recall"`
	F1                float64      `json:"f1"`
	FalsePositiveRate float64      `json:"false_positive_rate"`
	Results           []CaseResult `json:"results"`
}

// Evaluate scores the corpus with the given detectors (defaults to scan.Default).
func Evaluate(ctx context.Context, corpus Corpus, detectors ...scan.Detector) (Metrics, error) {
	if len(detectors) == 0 {
		detectors = scan.Default()
	}
	m := Metrics{Cases: len(corpus.Cases)}
	cleanWithFinding := 0

	for _, c := range corpus.Cases {
		findings, _, err := scan.Run(ctx, c.Content, c.Language, detectors...)
		if err != nil {
			return Metrics{}, err
		}
		found := uniqueRules(findings)
		foundSet := toSet(found)
		expected := toSet(c.Expect)

		var missing, unexpected []string
		for r := range expected {
			if _, ok := foundSet[r]; ok {
				m.TruePositives++
			} else {
				m.FalseNegatives++
				missing = append(missing, r)
			}
		}
		for r := range foundSet {
			if _, ok := expected[r]; !ok {
				m.FalsePositives++
				unexpected = append(unexpected, r)
			}
		}
		sort.Strings(missing)
		sort.Strings(unexpected)

		if len(c.Expect) == 0 {
			m.CleanCases++
			if len(found) > 0 {
				cleanWithFinding++
			}
		}
		m.Results = append(m.Results, CaseResult{
			Name:       c.Name,
			Expected:   sortedKeys(expected),
			Found:      found,
			Missing:    missing,
			Unexpected: unexpected,
		})
	}

	m.Precision = ratio(m.TruePositives, m.TruePositives+m.FalsePositives)
	m.Recall = ratio(m.TruePositives, m.TruePositives+m.FalseNegatives)
	if m.Precision+m.Recall > 0 {
		m.F1 = 2 * m.Precision * m.Recall / (m.Precision + m.Recall)
	}
	if m.CleanCases > 0 {
		m.FalsePositiveRate = float64(cleanWithFinding) / float64(m.CleanCases)
	}
	return m, nil
}

func uniqueRules(findings []scan.Finding) []string {
	set := map[string]struct{}{}
	for _, f := range findings {
		set[f.Rule] = struct{}{}
	}
	return sortedKeys(set)
}

func toSet(ss []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		set[s] = struct{}{}
	}
	return set
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
