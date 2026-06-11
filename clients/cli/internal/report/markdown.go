package report

import (
	"fmt"
	"io"
	"strings"
)

// RenderMarkdown writes the report as GitHub-flavored Markdown.
func RenderMarkdown(w io.Writer, rep Report) error {
	var b strings.Builder

	b.WriteString("# Generative-AI Technical-Debt Report\n\n")
	fmt.Fprintf(&b, "**MEMBRANE.AI Code Sweeper** · `%s` · generated %s\n\n",
		mdEscape(rep.Root), rep.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))

	fmt.Fprintf(&b, "## Health grade: %s\n\n", rep.Grade)
	fmt.Fprintf(&b, "Debt score **%.1f** — severity-weighted findings per 100 scanned files. %s\n\n",
		rep.DebtScore, rep.ScoringLegend)

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| Files scanned | %d |\n", rep.FilesScanned)
	fmt.Fprintf(&b, "| Files skipped | %d |\n", rep.FilesSkipped)
	fmt.Fprintf(&b, "| Findings | %d |\n", rep.Total)
	fmt.Fprintf(&b, "| — blocking | %d |\n", rep.Blocking)
	fmt.Fprintf(&b, "| — warning | %d |\n", rep.Warnings)
	fmt.Fprintf(&b, "| — info | %d |\n", rep.Infos)
	fmt.Fprintf(&b, "| Affected files | %d |\n\n", rep.AffectedFiles)

	if rep.Total == 0 {
		if rep.FilesScanned == 0 {
			b.WriteString("**Nothing was scanned** — every file was skipped (binary, oversized or unreadable); this is not a clean bill of health.\n")
		} else {
			b.WriteString("**Clean ✓** — no findings from the deterministic detector set.\n")
		}
		_, err := io.WriteString(w, b.String())
		return err
	}

	b.WriteString("## Findings by rule\n\n")
	b.WriteString("| Rule | Severity | Count |\n| --- | --- | ---: |\n")
	for _, r := range rep.ByRule {
		fmt.Fprintf(&b, "| `%s` | %s | %d |\n", mdEscape(r.Rule), r.Severity, r.Count)
	}
	b.WriteString("\n")

	b.WriteString("## Findings by directory\n\n")
	b.WriteString("| Directory | Blocking | Warning | Info | Weighted |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, d := range rep.ByDir {
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d |\n",
			mdEscape(d.Dir), d.Blocking, d.Warnings, d.Infos, d.Weighted)
	}
	b.WriteString("\n")

	b.WriteString("## Top files\n\n")
	b.WriteString("| File | Blocking | Warning | Info | Weighted |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, f := range rep.TopFiles {
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d |\n",
			mdEscape(f.File), f.Blocking, f.Warnings, f.Infos, f.Weighted)
	}
	b.WriteString("\n")

	b.WriteString("## Detail\n\n")
	b.WriteString("| File | Line | Severity | Rule | Message |\n| --- | ---: | --- | --- | --- |\n")
	for _, f := range rep.Findings {
		fmt.Fprintf(&b, "| `%s` | %d | %s | `%s` | %s |\n",
			mdEscape(f.File), f.Line, f.Severity, mdEscape(f.Rule), mdEscape(f.Message))
	}
	if rep.Truncated > 0 {
		fmt.Fprintf(&b, "\n_%d more finding(s) omitted from this table; the aggregates above cover all of them._\n", rep.Truncated)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// mdEscape keeps repo-derived text (paths, messages) from breaking table
// syntax or smuggling terminal escapes: control characters (incl. \r\n and
// ESC) become spaces, backslashes are doubled BEFORE pipes are escaped so a
// literal `\` can never re-arm a following `|`, and backticks would
// terminate the inline-code spans the cells use.
func mdEscape(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(s, "`", "'")
}
