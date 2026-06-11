// Package report aggregates a scan result into the "Generative-AI
// Technical-Debt Report": summary stats, per-rule / per-directory / per-file
// breakdowns and a transparent health grade, rendered as Markdown or a
// self-contained HTML page.
package report

import (
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/runner"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

// Severity weights for the debt score. Deliberately simple and disclosed in
// the rendered report — the score must be explainable, not a black box.
const (
	weightBlocking = 5
	weightWarning  = 2
	weightInfo     = 1
)

// Grade thresholds (strict <). Rendered verbatim via ScoringLegend, so the
// explanation can never drift from the buckets.
const (
	gradeB = 5
	gradeC = 15
	gradeD = 40
)

// minGradeSample floors the grade's denominator so tiny scans don't grade F
// on a single finding; the raw DebtScore stays the honest density.
const minGradeSample = 10

// maxDetailFindings caps the detail section; aggregates always cover
// everything and the report states how many rows were omitted.
const maxDetailFindings = 200

// maxDirRows caps the directory table; the remainder is aggregated into a
// single labelled row, never dropped silently.
const maxDirRows = 15

// maxTopFiles bounds the "top offenders" table.
const maxTopFiles = 10

// RuleStat is the aggregate for one detector rule.
type RuleStat struct {
	Rule     string
	Severity string
	Count    int
}

// DirStat is the aggregate for one directory.
type DirStat struct {
	Dir      string
	Blocking int
	Warnings int
	Infos    int
	Weighted int
}

// FileStat is the aggregate for one file.
type FileStat struct {
	File     string
	Blocking int
	Warnings int
	Infos    int
	Weighted int
}

// Report is the fully aggregated technical-debt view of one scan run.
type Report struct {
	Root        string
	GeneratedAt time.Time

	FilesScanned  int
	FilesSkipped  int
	Total         int
	Blocking      int
	Warnings      int
	Infos         int
	AffectedFiles int

	// DebtScore is severity-weighted findings per 100 scanned files
	// (blocking×5 + warning×2 + info×1). Grade buckets it A–F; scans smaller
	// than minGradeSample files are graded against that floor, and a scan
	// that inspected zero files grades "—". ScoringLegend is the rendered
	// explanation, built from the same constants.
	DebtScore     float64
	Grade         string
	ScoringLegend string

	ByRule   []RuleStat
	ByDir    []DirStat
	TopFiles []FileStat

	// Findings is the detail section, capped at maxDetailFindings;
	// Truncated counts the omitted rows (0 = complete).
	Findings  []runner.FileFinding
	Truncated int
}

// Build aggregates a runner result. The caller supplies the timestamp so
// rendering stays deterministic under test. Totals are tallied from the
// findings themselves — one source of truth shared with every table — rather
// than trusted from the runner's counters.
func Build(res runner.Result, generatedAt time.Time) Report {
	rep := Report{
		Root:         res.Root,
		GeneratedAt:  generatedAt,
		FilesScanned: res.FilesScanned,
		FilesSkipped: res.FilesSkipped,
		Total:        len(res.Findings),
	}

	rules := map[string]*RuleStat{}
	dirs := map[string]*DirStat{}
	files := map[string]*FileStat{}
	for _, f := range res.Findings {
		r, ok := rules[f.Rule]
		if !ok {
			r = &RuleStat{Rule: f.Rule, Severity: string(f.Severity)}
			rules[f.Rule] = r
		}
		r.Count++

		dir := path.Dir(f.File)
		d, ok := dirs[dir]
		if !ok {
			d = &DirStat{Dir: dir}
			dirs[dir] = d
		}
		fl, ok := files[f.File]
		if !ok {
			fl = &FileStat{File: f.File}
			files[f.File] = fl
		}
		w := weightOf(f.Severity)
		d.Weighted += w
		fl.Weighted += w
		switch f.Severity {
		case scan.SeverityBlocking:
			rep.Blocking++
			d.Blocking++
			fl.Blocking++
		case scan.SeverityWarning:
			rep.Warnings++
			d.Warnings++
			fl.Warnings++
		default:
			rep.Infos++
			d.Infos++
			fl.Infos++
		}
	}

	rep.AffectedFiles = len(files)
	rep.ByRule = sortRules(rules)
	rep.ByDir = sortDirs(dirs)
	rep.TopFiles = sortFiles(files)

	weighted := weightBlocking*rep.Blocking + weightWarning*rep.Warnings + weightInfo*rep.Infos
	if res.FilesScanned > 0 {
		rep.DebtScore = 100 * float64(weighted) / float64(res.FilesScanned)
	}
	rep.Grade = gradeOf(weighted, res.FilesScanned)
	rep.ScoringLegend = scoringLegend()

	// The detail section is capped, so order it severity-first: a truncated
	// report must never cut a blocking finding while keeping info rows.
	rep.Findings = append([]runner.FileFinding(nil), res.Findings...)
	sort.SliceStable(rep.Findings, func(i, j int) bool {
		ri, rj := severityRank(rep.Findings[i].Severity), severityRank(rep.Findings[j].Severity)
		if ri != rj {
			return ri > rj
		}
		if rep.Findings[i].File != rep.Findings[j].File {
			return rep.Findings[i].File < rep.Findings[j].File
		}
		return rep.Findings[i].Line < rep.Findings[j].Line
	})
	if len(rep.Findings) > maxDetailFindings {
		rep.Truncated = len(rep.Findings) - maxDetailFindings
		rep.Findings = rep.Findings[:maxDetailFindings]
	}
	return rep
}

// weightOf and severityRank are the single severity mapping. An unknown
// severity deliberately weighs like a warning — in a security report the
// unclassified case must never land in the cheapest bucket.
func weightOf(severity scan.Severity) int {
	switch severity {
	case scan.SeverityBlocking:
		return weightBlocking
	case scan.SeverityInfo:
		return weightInfo
	default:
		return weightWarning
	}
}

func severityRank(severity scan.Severity) int {
	switch severity {
	case scan.SeverityBlocking:
		return 3
	case scan.SeverityInfo:
		return 1
	default:
		return 2
	}
}

// gradeOf buckets the severity-weighted density. Thresholds are part of the
// report's public story (rendered via ScoringLegend), so change them
// deliberately. Small scans grade against the minGradeSample floor; a scan
// that inspected zero files cannot claim a grade at all.
func gradeOf(weighted, filesScanned int) string {
	if filesScanned == 0 {
		if weighted == 0 {
			return "—"
		}
		return "F" // findings without scanned files: inconsistent input, fail loud
	}
	sample := filesScanned
	if sample < minGradeSample {
		sample = minGradeSample
	}
	score := 100 * float64(weighted) / float64(sample)
	switch {
	case score == 0:
		return "A"
	case score < gradeB:
		return "B"
	case score < gradeC:
		return "C"
	case score < gradeD:
		return "D"
	default:
		return "F"
	}
}

// scoringLegend renders the formula from the same constants gradeOf uses, so
// the printed explanation can never drift from the actual buckets.
func scoringLegend() string {
	return fmt.Sprintf(
		"Weights: blocking ×%d · warning ×%d · info ×%d. Grades: A = 0 · B < %d · C < %d · D < %d · F ≥ %d. Scans under %d files are graded against a %d-file floor.",
		weightBlocking, weightWarning, weightInfo,
		gradeB, gradeC, gradeD, gradeD, minGradeSample, minGradeSample)
}

func sortRules(m map[string]*RuleStat) []RuleStat {
	out := make([]RuleStat, 0, len(m))
	for _, r := range m {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

func sortDirs(m map[string]*DirStat) []DirStat {
	out := make([]DirStat, 0, len(m))
	for _, d := range m {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weighted != out[j].Weighted {
			return out[i].Weighted > out[j].Weighted
		}
		return out[i].Dir < out[j].Dir
	})
	if len(out) > maxDirRows {
		other := DirStat{Dir: "(other directories)"}
		for _, d := range out[maxDirRows:] {
			other.Blocking += d.Blocking
			other.Warnings += d.Warnings
			other.Infos += d.Infos
			other.Weighted += d.Weighted
		}
		out = append(out[:maxDirRows], other)
	}
	return out
}

func sortFiles(m map[string]*FileStat) []FileStat {
	out := make([]FileStat, 0, len(m))
	for _, f := range m {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weighted != out[j].Weighted {
			return out[i].Weighted > out[j].Weighted
		}
		return out[i].File < out[j].File
	})
	if len(out) > maxTopFiles {
		out = out[:maxTopFiles]
	}
	return out
}
