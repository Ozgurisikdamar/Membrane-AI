package report_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/report"
	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/runner"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

var at = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

func sampleResult() runner.Result {
	return runner.Result{
		Root:         "/repo",
		FilesScanned: 40,
		FilesSkipped: 3,
		Blocking:     2,
		Warnings:     1,
		Infos:        1,
		Findings: []runner.FileFinding{
			{File: "auth/creds.go", Line: 3, Rule: "aws-access-key-id", Severity: scan.SeverityBlocking, Message: "credential"},
			{File: "auth/creds.go", Line: 9, Rule: "generic-assigned-secret", Severity: scan.SeverityBlocking, Message: "credential"},
			{File: "db/query.go", Line: 5, Rule: "sql-string-concat", Severity: scan.SeverityWarning, Message: "sql concat"},
			{File: "notes|weird.md", Line: 1, Rule: "todo-marker", Severity: scan.SeverityInfo, Message: "has `ticks`"},
		},
	}
}

func TestBuild_Aggregates(t *testing.T) {
	rep := report.Build(sampleResult(), at)

	if rep.Total != 4 || rep.AffectedFiles != 3 {
		t.Fatalf("total=%d affected=%d", rep.Total, rep.AffectedFiles)
	}
	// 2×5 + 1×2 + 1×1 = 13 weighted over 40 files → 32.5 per 100 → grade D.
	if rep.DebtScore != 32.5 || rep.Grade != "D" {
		t.Fatalf("score=%v grade=%s", rep.DebtScore, rep.Grade)
	}
	if len(rep.ByRule) != 4 {
		t.Fatalf("byRule = %+v", rep.ByRule)
	}
	// All counts tie at 1 → pure name tie-break, ascending.
	for i := 1; i < len(rep.ByRule); i++ {
		if rep.ByRule[i-1].Rule > rep.ByRule[i].Rule {
			t.Fatalf("byRule tie-break broken: %+v", rep.ByRule)
		}
	}
	// auth (weighted 10) must outrank db (weighted 2).
	if rep.ByDir[0].Dir != "auth" || rep.ByDir[0].Weighted != 10 {
		t.Fatalf("byDir = %+v", rep.ByDir)
	}
	if rep.TopFiles[0].File != "auth/creds.go" || rep.TopFiles[0].Blocking != 2 {
		t.Fatalf("topFiles = %+v", rep.TopFiles)
	}
	if rep.Truncated != 0 {
		t.Fatalf("truncated = %d", rep.Truncated)
	}
}

func TestBuild_GradeBoundaries(t *testing.T) {
	cases := []struct {
		blocking, scanned int
		grade             string
	}{
		{0, 10, "A"},  // zero findings
		{0, 0, "—"},   // nothing scanned: no grade, not a clean A
		{1, 200, "B"}, // 5/200×100 = 2.5
		{1, 50, "C"},  // 10
		{1, 20, "D"},  // 25
		{8, 10, "F"},  // 400
		// Exact thresholds — strict <, so the boundary belongs to the worse grade.
		{1, 100, "C"}, // exactly 5.0
		{3, 100, "D"}, // exactly 15.0
		{8, 100, "F"}, // exactly 40.0
	}
	for _, c := range cases {
		res := runner.Result{FilesScanned: c.scanned, Blocking: c.blocking}
		for i := 0; i < c.blocking; i++ {
			res.Findings = append(res.Findings, runner.FileFinding{
				File: "f.go", Line: i + 1, Rule: "r", Severity: scan.SeverityBlocking,
			})
		}
		if got := report.Build(res, at).Grade; got != c.grade {
			t.Errorf("blocking=%d scanned=%d: grade=%s want %s", c.blocking, c.scanned, got, c.grade)
		}
	}
}

func TestBuild_SmallScanGradeFloor(t *testing.T) {
	// One warning in a one-file scan: the honest density is 200, but the
	// grade uses the 10-file floor → 2×100/10 = 20 → D, not an automatic F.
	res := runner.Result{
		FilesScanned: 1,
		Findings: []runner.FileFinding{
			{File: "f.go", Line: 1, Rule: "r", Severity: scan.SeverityWarning},
		},
	}
	rep := report.Build(res, at)
	if rep.DebtScore != 200 || rep.Grade != "D" {
		t.Fatalf("score=%v grade=%s, want 200 / D", rep.DebtScore, rep.Grade)
	}
}

func TestBuild_RuleOrderingByCountThenName(t *testing.T) {
	res := runner.Result{FilesScanned: 10}
	add := func(rule string, n int) {
		for i := 0; i < n; i++ {
			res.Findings = append(res.Findings, runner.FileFinding{
				File: "f.go", Line: len(res.Findings) + 1, Rule: rule, Severity: scan.SeverityInfo,
			})
		}
	}
	add("zeta", 3)
	add("alpha", 1)
	add("beta", 1)

	got := report.Build(res, at).ByRule
	want := []string{"zeta", "alpha", "beta"} // count desc, then name asc
	for i, w := range want {
		if got[i].Rule != w {
			t.Fatalf("byRule order = %+v, want %v", got, want)
		}
	}
}

func TestBuild_DetailCapKeepsBlockingFirst(t *testing.T) {
	// 250 early-sorting info findings + 1 blocking in a late-sorting file:
	// the capped detail section must keep the blocking row, severity-first.
	res := runner.Result{FilesScanned: 500}
	for i := 0; i < 250; i++ {
		res.Findings = append(res.Findings, runner.FileFinding{
			File: fmt.Sprintf("aaa/f%03d.go", i), Line: 1, Rule: "todo", Severity: scan.SeverityInfo,
		})
	}
	res.Findings = append(res.Findings, runner.FileFinding{
		File: "zzz/creds.go", Line: 7, Rule: "aws-access-key-id", Severity: scan.SeverityBlocking,
	})

	rep := report.Build(res, at)
	if rep.Truncated != 51 {
		t.Fatalf("truncated = %d", rep.Truncated)
	}
	if rep.Findings[0].File != "zzz/creds.go" || rep.Findings[0].Severity != scan.SeverityBlocking {
		t.Fatalf("blocking finding not first: %+v", rep.Findings[0])
	}
}

func TestBuild_TotalsDerivedFromFindings(t *testing.T) {
	// Runner counters are deliberately wrong: the report must not trust them.
	res := runner.Result{
		FilesScanned: 10,
		Blocking:     99, Warnings: 99, Infos: 99,
		Findings: []runner.FileFinding{
			{File: "f.go", Line: 1, Rule: "r", Severity: scan.SeverityBlocking},
		},
	}
	rep := report.Build(res, at)
	if rep.Blocking != 1 || rep.Warnings != 0 || rep.Infos != 0 {
		t.Fatalf("totals = %d/%d/%d, want 1/0/0", rep.Blocking, rep.Warnings, rep.Infos)
	}
}

func TestBuild_CapsDetailAndDirs(t *testing.T) {
	res := runner.Result{FilesScanned: 1000}
	for i := 0; i < 250; i++ {
		res.Findings = append(res.Findings, runner.FileFinding{
			File: fmt.Sprintf("dir%03d/f.go", i), Line: 1, Rule: "r", Severity: scan.SeverityInfo,
		})
		res.Infos++
	}
	rep := report.Build(res, at)

	if len(rep.Findings) != 200 || rep.Truncated != 50 {
		t.Fatalf("detail=%d truncated=%d", len(rep.Findings), rep.Truncated)
	}
	// 15 dir rows + the aggregated remainder.
	if len(rep.ByDir) != 16 || rep.ByDir[15].Dir != "(other directories)" {
		t.Fatalf("byDir rows=%d last=%q", len(rep.ByDir), rep.ByDir[len(rep.ByDir)-1].Dir)
	}
	if rep.ByDir[15].Infos != 250-15 {
		t.Fatalf("other-dirs infos = %d", rep.ByDir[15].Infos)
	}
}

func TestRenderMarkdown_EscapesAndStructure(t *testing.T) {
	var b strings.Builder
	if err := report.RenderMarkdown(&b, report.Build(sampleResult(), at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()

	for _, want := range []string{
		"# Generative-AI Technical-Debt Report",
		"## Health grade: D",
		"| Files scanned | 40 |",
		"`aws-access-key-id`",
		"generated 2026-06-11 12:00 UTC",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	// Path pipes and message backticks must not break the tables.
	if !strings.Contains(out, "notes\\|weird.md") {
		t.Error("pipe in path not escaped")
	}
	if strings.Contains(out, "has `ticks`") {
		t.Error("backticks in message not neutralized")
	}
}

func TestRenderMarkdown_NeutralizesControlCharsAndBackslashes(t *testing.T) {
	res := runner.Result{
		FilesScanned: 10,
		Findings: []runner.FileFinding{
			// CR would split a GFM table row; ESC would smuggle ANSI codes;
			// a literal backslash before a pipe would re-arm the delimiter.
			{File: "x\rfake.go", Line: 1, Rule: "r", Severity: scan.SeverityInfo, Message: "esc \x1b[31m here"},
			{File: `dir\|tricky.go`, Line: 2, Rule: "r", Severity: scan.SeverityInfo, Message: "m"},
		},
	}
	var b strings.Builder
	if err := report.RenderMarkdown(&b, report.Build(res, at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, "\r") || strings.Contains(out, "\x1b") {
		t.Error("control characters leaked into the report")
	}
	if !strings.Contains(out, `dir\\\|tricky.go`) {
		t.Error("backslash not doubled before pipe escaping")
	}
}

func TestRenderMarkdown_NothingScanned(t *testing.T) {
	var b strings.Builder
	res := runner.Result{Root: "/repo", FilesSkipped: 4}
	if err := report.RenderMarkdown(&b, report.Build(res, at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "Health grade: —") || !strings.Contains(out, "Nothing was scanned") {
		t.Fatalf("zero-scan report wrong:\n%s", out)
	}
	if strings.Contains(out, "Clean ✓") {
		t.Error("zero-scan run must not claim a clean bill of health")
	}
}

func TestRenderMarkdown_CleanRun(t *testing.T) {
	var b strings.Builder
	res := runner.Result{Root: "/repo", FilesScanned: 5}
	if err := report.RenderMarkdown(&b, report.Build(res, at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "Health grade: A") || !strings.Contains(out, "Clean ✓") {
		t.Fatalf("clean report wrong:\n%s", out)
	}
	if strings.Contains(out, "## Detail") {
		t.Error("clean report must not render empty tables")
	}
}

func TestRenderHTML_EscapesAndStructure(t *testing.T) {
	res := sampleResult()
	res.Findings = append(res.Findings, runner.FileFinding{
		File: "x.go", Line: 1, Rule: "r", Severity: scan.SeverityInfo,
		Message: `<script>alert("x")</script>`,
	})
	res.Infos++

	var b strings.Builder
	if err := report.RenderHTML(&b, report.Build(res, at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()

	for _, want := range []string{
		"<!DOCTYPE html>",
		"Generative-AI Technical-Debt Report",
		`<span class="sev blocking">blocking</span>`,
		"aws-access-key-id",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "<script>alert") {
		t.Error("repo-derived text not HTML-escaped")
	}
}

// maskedScan runs the real runner (and therefore pkg/scan's real maskers) over
// a file holding a documented example credential, so the rendered reports are
// checked against genuine masker output rather than a hand-written string.
func maskedScan(t *testing.T) runner.Result {
	t.Helper()
	root := t.TempDir()
	src := "package cfg\n\nvar awsKey = \"AKIAIOSFODNN7EXAMPLE\" // <b>fallback</b> | dev\n"
	if err := os.WriteFile(filepath.Join(root, "cfg.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := runner.Run(context.Background(), runner.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Excerpt == "" {
		t.Fatalf("findings = %+v", res.Findings)
	}
	return res
}

func TestRenderHTML_ExcerptIsMaskedHighlightedAndEscaped(t *testing.T) {
	var b strings.Builder
	if err := report.RenderHTML(&b, report.Build(maskedScan(t), at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatal("raw credential reached the HTML report")
	}
	for _, want := range []string{
		`<mark>[MASKED:aws-access-key-id]</mark>`, // placeholder regex matches the real masker
		`&lt;b&gt;fallback&lt;/b&gt;`,             // repo text around it stays escaped
		`<span class="ln">3</span>`,
		`class="grade grade-`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestRenderMarkdown_ExcerptColumnIsMaskedAndEscaped(t *testing.T) {
	var b strings.Builder
	if err := report.RenderMarkdown(&b, report.Build(maskedScan(t), at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatal("raw credential reached the Markdown report")
	}
	// The pipe inside the quoted line must not split the table row.
	if !strings.Contains(out, "`var awsKey = \"[MASKED:aws-access-key-id]\" // <b>fallback</b> \\| dev` |") {
		t.Fatalf("masked source column missing or unescaped:\n%s", out)
	}
}

func TestRenderHTML_RuleBarsScaleToTheLargestRule(t *testing.T) {
	var b strings.Builder
	if err := report.RenderHTML(&b, report.Build(sampleResult(), at)); err != nil {
		t.Fatal(err)
	}
	// Every sample rule has count 1, so every bar is full width.
	if got := strings.Count(b.String(), `style="width: 100%"`); got != 4 {
		t.Fatalf("full-width bars = %d, want 4", got)
	}

	res := runner.Result{FilesScanned: 10}
	for i, rule := range []string{"zeta", "zeta", "zeta", "alpha"} {
		res.Findings = append(res.Findings, runner.FileFinding{
			File: "f.go", Line: i + 1, Rule: rule, Severity: scan.SeverityWarning,
		})
	}
	b.Reset()
	if err := report.RenderHTML(&b, report.Build(res, at)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `style="width: 33.3%"`) {
		t.Fatal("a 1-of-3 rule must render a one-third bar")
	}
}

func TestRenderHTML_NothingScannedHasNeutralGrade(t *testing.T) {
	var b strings.Builder
	if err := report.RenderHTML(&b, report.Build(runner.Result{Root: "/repo", FilesSkipped: 4}, at)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, `class="grade grade-none"`) || !strings.Contains(out, "Nothing was scanned") {
		t.Fatalf("zero-scan HTML report wrong:\n%s", out)
	}
	if strings.Contains(out, "Clean ✓") {
		t.Error("zero-scan run must not claim a clean bill of health")
	}
}
