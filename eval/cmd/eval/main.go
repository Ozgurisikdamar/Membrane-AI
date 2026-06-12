// Command eval scores the MEMBRANE.AI deterministic detectors against a labeled
// corpus and enforces an accuracy SLO (D-032 sibling: quality gate for pkg/scan).
//
//	eval [--corpus path] [--json] [--min-precision f] [--min-recall f] [--max-fp-rate f]
//
// Exit codes: 0 = all SLOs met · 1 = an SLO was breached · 2 = usage/load error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Ozgurisikdamar/Membrane-AI/eval/harness"
)

const (
	exitPass  = 0
	exitSLO   = 1
	exitUsage = 2
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	corpusPath := fs.String("corpus", "corpus/corpus.json", "path to the labeled corpus JSON")
	jsonOut := fs.Bool("json", false, "emit the metrics as JSON")
	minPrecision := fs.Float64("min-precision", 0.95, "fail below this precision")
	minRecall := fs.Float64("min-recall", 0.90, "fail below this recall")
	maxFPRate := fs.Float64("max-fp-rate", 0.05, "fail above this clean-case false-positive rate")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	corpus, err := harness.LoadCorpus(*corpusPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load corpus:", err)
		return exitUsage
	}
	metrics, err := harness.Evaluate(context.Background(), corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "evaluate:", err)
		return exitUsage
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(metrics); err != nil {
			fmt.Fprintln(os.Stderr, "encode:", err)
			return exitUsage
		}
	} else {
		render(metrics)
	}

	return decide(metrics, *minPrecision, *minRecall, *maxFPRate)
}

func render(m harness.Metrics) {
	fmt.Printf("MEMBRANE.AI detector accuracy — %d cases (%d clean)\n", m.Cases, m.CleanCases)
	fmt.Printf("  TP=%d  FP=%d  FN=%d\n", m.TruePositives, m.FalsePositives, m.FalseNegatives)
	fmt.Printf("  precision=%.3f  recall=%.3f  f1=%.3f  fp-rate=%.3f\n",
		m.Precision, m.Recall, m.F1, m.FalsePositiveRate)
	for _, r := range m.Results {
		if len(r.Missing) > 0 || len(r.Unexpected) > 0 {
			fmt.Printf("  ✗ %s  missing=%v unexpected=%v\n", r.Name, r.Missing, r.Unexpected)
		}
	}
}

// decide reports each breached SLO and returns the exit code.
func decide(m harness.Metrics, minP, minR, maxFP float64) int {
	breached := false
	if m.Precision < minP {
		fmt.Fprintf(os.Stderr, "SLO breach: precision %.3f < %.3f\n", m.Precision, minP)
		breached = true
	}
	if m.Recall < minR {
		fmt.Fprintf(os.Stderr, "SLO breach: recall %.3f < %.3f\n", m.Recall, minR)
		breached = true
	}
	if m.FalsePositiveRate > maxFP {
		fmt.Fprintf(os.Stderr, "SLO breach: fp-rate %.3f > %.3f\n", m.FalsePositiveRate, maxFP)
		breached = true
	}
	if breached {
		return exitSLO
	}
	return exitPass
}
