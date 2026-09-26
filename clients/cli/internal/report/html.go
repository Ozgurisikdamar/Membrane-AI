package report

import (
	"html/template"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// RenderHTML writes the report as a single self-contained HTML page (inline
// CSS and SVG, no external assets) so it can be attached to a PR, mailed or
// hosted as-is. html/template escapes all repo-derived text.
func RenderHTML(w io.Writer, rep Report) error {
	return htmlTmpl.Execute(w, rep)
}

// htmlFuncs are presentation-only helpers; every number they touch is
// already computed by Build.
var htmlFuncs = template.FuncMap{
	"gradeClass": gradeClass,
	"pct":        pct,
	"segments":   segments,
	"sentences":  sentences,
}

// gradeClass maps a grade to its CSS modifier ("—" and anything unexpected
// render neutral).
func gradeClass(grade string) string {
	switch grade {
	case "A", "B", "C", "D", "F":
		return strings.ToLower(grade)
	default:
		return "none"
	}
}

// pct renders n as a percentage of limit for the rule bars, clamped to
// [0, 100].
func pct(n, limit int) string {
	if n <= 0 || limit <= 0 {
		return "0"
	}
	if n >= limit {
		return "100"
	}
	return strconv.FormatFloat(100*float64(n)/float64(limit), 'f', 1, 64)
}

// sentences splits prose on ". " so the scoring legend renders one rule per
// line instead of breaking inside "C < 15".
func sentences(s string) []string {
	parts := strings.SplitAfter(s, ". ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// maskPlaceholder matches the [MASKED:<rule>] marker pkg/scan's maskers write
// in place of a detected secret, so the page can make every redaction
// visible (report_test.go pins it against the real masker output).
var maskPlaceholder = regexp.MustCompile(`\[MASKED:[A-Za-z0-9_-]+\]`)

// segment is a run of excerpt text; Masked runs are redaction placeholders.
type segment struct {
	Text   string
	Masked bool
}

// segments splits an excerpt around its mask placeholders. Text stays
// verbatim — html/template escapes it on output.
func segments(excerpt string) []segment {
	var out []segment
	last := 0
	for _, loc := range maskPlaceholder.FindAllStringIndex(excerpt, -1) {
		if loc[0] > last {
			out = append(out, segment{Text: excerpt[last:loc[0]]})
		}
		out = append(out, segment{Text: excerpt[loc[0]:loc[1]], Masked: true})
		last = loc[1]
	}
	if last < len(excerpt) {
		out = append(out, segment{Text: excerpt[last:]})
	}
	return out
}

var htmlTmpl = template.Must(template.New("report").Funcs(htmlFuncs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Generative-AI Technical-Debt Report — MEMBRANE.AI</title>
<style>
  :root {
    --ink: #0f172a; --ink-2: #334155; --muted: #64748b; --faint: #94a3b8;
    --line: #e2e8f0; --line-soft: #eef2f6; --bg: #f4f6f8; --card: #ffffff; --head: #f8fafc;
    --accent: #0f766e; --accent-ink: #115e59; --accent-soft: #ccfbf1; --accent-ring: #99f6e4;
    --blocking: #dc2626; --blocking-bg: #fef2f2; --blocking-ink: #b91c1c; --blocking-ring: #fecaca;
    --warning: #d97706;  --warning-bg: #fffbeb;  --warning-ink: #b45309;  --warning-ring: #fde68a;
    --info: #2563eb;     --info-bg: #eff6ff;     --info-ink: #1d4ed8;     --info-ring: #bfdbfe;
    --sans: Inter, ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    --mono: ui-monospace, "SF Mono", "JetBrains Mono", "Cascadia Code", Menlo, Consolas, monospace;
  }
  * { box-sizing: border-box; }
  body { margin: 0; background: var(--bg); color: var(--ink);
         font: 15px/1.6 var(--sans); -webkit-font-smoothing: antialiased; }
  .wrap { max-width: 1240px; margin: 0 auto; padding: 0 32px; }
  main.wrap { padding-top: 38px; padding-bottom: 8px; }
  code { font-family: var(--mono); font-size: 13px; font-variant-ligatures: none; }

  .top { background: var(--card); border-bottom: 1px solid var(--line); }
  .top .wrap { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 62px; }
  .brand { display: flex; align-items: center; gap: 10px; font-weight: 700; font-size: 15px; letter-spacing: 0.04em; }
  .brand svg { width: 28px; height: 28px; flex: none; }
  .brand .ai { color: var(--accent); }
  .brand .tool { font-weight: 500; letter-spacing: 0; color: var(--muted); padding-left: 12px; margin-left: 2px;
                 border-left: 1px solid var(--line); }
  .top .meta { color: var(--muted); font-size: 13px; }

  h1 { font-size: 30px; line-height: 1.2; letter-spacing: -0.022em; margin: 0 0 10px; }
  h2 { font-size: 17px; line-height: 1.3; letter-spacing: -0.01em; margin: 0 0 12px; }
  .sub { color: var(--muted); margin: 0 0 26px; display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; }
  .sub code { color: var(--ink-2); background: var(--card); border: 1px solid var(--line); border-radius: 7px; padding: 2px 8px; }
  .sub .dot { color: var(--faint); }
  section { margin-bottom: 38px; }
  .lead { color: var(--muted); font-size: 14px; margin: -4px 0 14px; }
  .lead code { font-size: 12.5px; color: var(--accent-ink); background: var(--accent-soft); border-radius: 5px; padding: 1px 6px; }
  .hint { color: var(--faint); font-size: 12.5px; margin-left: 6px; }

  .summary { display: grid; grid-template-columns: minmax(0, 7fr) minmax(0, 6fr); gap: 16px; margin-bottom: 38px; }
  .grade { display: flex; align-items: center; gap: 22px; background: var(--card); border: 1px solid var(--line);
           border-radius: 14px; padding: 22px 24px; }
  .grade .letter { flex: none; width: 88px; height: 88px; border-radius: 22px; display: grid; place-items: center;
                   font-size: 48px; font-weight: 800; line-height: 1; color: #fff; background: var(--faint); }
  .grade-a .letter { background: #15803d; }
  .grade-b .letter { background: var(--accent); }
  .grade-c .letter { background: #ca8a04; }
  .grade-d .letter { background: #ea580c; }
  .grade-f .letter { background: var(--blocking); }
  .grade .kicker { font-size: 12px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--muted); }
  .grade .body { flex: 1; min-width: 0; }
  .grade .score { margin: 2px 0 0; color: var(--ink-2); }
  .grade .score strong { font-size: 24px; color: var(--ink); letter-spacing: -0.01em; margin-left: 4px; }
  .grade .per { color: var(--ink-2); font-size: 14px; margin-bottom: 8px; }
  .grade .expl { color: var(--muted); font-size: 12.5px; line-height: 1.6; }
  .grade .expl span { display: block; }
  .cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
  .card { background: var(--card); border: 1px solid var(--line); border-radius: 12px; padding: 14px 18px; }
  .card .num { font-size: 28px; font-weight: 700; line-height: 1.25; letter-spacing: -0.02em; font-variant-numeric: tabular-nums; }
  .card .lbl { color: var(--muted); font-size: 13px; display: flex; align-items: center; gap: 7px; }
  .card .lbl i { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
  .card.blocking .num { color: var(--blocking-ink); } .card.blocking .lbl i { background: var(--blocking); }
  .card.warning .num { color: var(--warning-ink); }   .card.warning .lbl i { background: var(--warning); }
  .card.info .num { color: var(--info-ink); }         .card.info .lbl i { background: var(--info); }

  .split { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 20px; align-items: start; }
  table { width: 100%; border-collapse: separate; border-spacing: 0; background: var(--card);
          border: 1px solid var(--line); border-radius: 12px; overflow: hidden; }
  th, td { text-align: left; padding: 10px 14px; border-top: 1px solid var(--line-soft); vertical-align: middle; }
  thead th { background: var(--head); border-top: none; border-bottom: 1px solid var(--line); font-size: 11.5px;
             font-weight: 600; color: var(--muted); text-transform: uppercase; letter-spacing: 0.06em; }
  tbody tr:first-child td { border-top: none; }
  td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
  td.num.zero { color: #cbd5e1; }
  td.num.strong { font-weight: 700; }
  td code { color: var(--ink-2); white-space: nowrap; }
  .bar { display: inline-block; width: 96px; height: 8px; border-radius: 999px; background: #f1f5f9;
         overflow: hidden; vertical-align: middle; margin-right: 12px; }
  .bar i { display: block; height: 100%; border-radius: 999px; }
  .bar i.blocking { background: var(--blocking); } .bar i.warning { background: var(--warning); } .bar i.info { background: var(--info); }
  .sev { display: inline-block; border-radius: 999px; padding: 1px 10px; font-size: 12px; font-weight: 600; line-height: 1.6; white-space: nowrap; }
  .sev.blocking { background: var(--blocking-bg); color: var(--blocking-ink); box-shadow: inset 0 0 0 1px var(--blocking-ring); }
  .sev.warning { background: var(--warning-bg); color: var(--warning-ink); box-shadow: inset 0 0 0 1px var(--warning-ring); }
  .sev.info { background: var(--info-bg); color: var(--info-ink); box-shadow: inset 0 0 0 1px var(--info-ring); }

  table.detail td { vertical-align: baseline; padding-top: 9px; }
  table.detail td.msg { font-size: 13.5px; color: var(--ink-2); }
  table.detail tr.has-excerpt td { padding-bottom: 5px; }
  table.detail tr.excerpt-row td { border-top: none; padding-top: 0; padding-bottom: 10px; }
  .excerpt { display: flex; align-items: baseline; gap: 14px; background: var(--head); border: 1px solid var(--line);
             border-left: 3px solid var(--faint); border-radius: 8px; padding: 4px 14px; }
  .excerpt.blocking { border-left-color: var(--blocking); }
  .excerpt.warning { border-left-color: var(--warning); }
  .excerpt.info { border-left-color: var(--info); }
  .excerpt .ln { flex: none; min-width: 2.5ch; text-align: right; font: 12px/1.7 var(--mono); color: var(--faint); user-select: none; }
  .excerpt code { font-size: 13px; line-height: 1.7; color: var(--ink-2); white-space: pre-wrap; overflow-wrap: anywhere; }
  .excerpt mark { background: var(--accent-soft); color: var(--accent-ink); font-weight: 600; border-radius: 5px;
                  padding: 1px 4px; box-shadow: inset 0 0 0 1px var(--accent-ring); }

  .clean { background: #f0fdf4; color: #15803d; border: 1px solid #bbf7d0; border-radius: 12px; padding: 18px 22px; font-weight: 600; }
  .empty { background: var(--warning-bg); color: var(--warning-ink); border: 1px solid var(--warning-ring); border-radius: 12px; padding: 18px 22px; font-weight: 600; }
  .note { color: var(--muted); font-size: 13px; margin: 10px 0 0; }
  footer p { color: var(--muted); font-size: 13px; border-top: 1px solid var(--line); margin: 0; padding: 18px 0 40px; }

  @media (max-width: 900px) {
    .summary, .split { grid-template-columns: minmax(0, 1fr); }
    .top .meta { display: none; }
    td code { white-space: normal; overflow-wrap: anywhere; }
  }
  @media print {
    body { background: #fff; }
    .card, .grade, tr { break-inside: avoid; }
  }
</style>
</head>
<body>
<header class="top">
  <div class="wrap">
    <div class="brand">
      <svg viewBox="0 0 32 32" aria-hidden="true"><rect width="32" height="32" rx="8" fill="#0f766e"/><path d="M16 6.5l8.5 3.6v5.6c0 5.3-3.6 9.3-8.5 10.8-4.9-1.5-8.5-5.5-8.5-10.8v-5.6z" fill="none" stroke="#ccfbf1" stroke-width="2" stroke-linejoin="round"/><path d="M12.4 16.3l2.5 2.5 4.9-5.1" fill="none" stroke="#ccfbf1" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
      <span>MEMBRANE<span class="ai">.AI</span></span>
      <span class="tool">Code Sweeper</span>
    </div>
    <div class="meta">generated {{.GeneratedAt.UTC.Format "2006-01-02 15:04 UTC"}}</div>
  </div>
</header>
<main class="wrap">
  <h1>Generative-AI Technical-Debt Report</h1>
  <p class="sub"><span>MEMBRANE.AI Code Sweeper</span><span class="dot">·</span><code>{{.Root}}</code><span class="dot">·</span><span>{{.FilesScanned}} files scanned, {{.FilesSkipped}} skipped</span></p>

  <div class="summary">
    <div class="grade grade-{{gradeClass .Grade}}">
      <div class="letter">{{.Grade}}</div>
      <div class="body">
        <div class="kicker">Health grade</div>
        <div class="score">Debt score <strong>{{printf "%.1f" .DebtScore}}</strong></div>
        <div class="per">Severity-weighted findings per 100 scanned files.</div>
        <div class="expl">{{range sentences .ScoringLegend}}<span>{{.}}</span>{{end}}</div>
      </div>
    </div>
    <div class="cards">
      <div class="card"><div class="num">{{.FilesScanned}}</div><div class="lbl">files scanned</div></div>
      <div class="card"><div class="num">{{.Total}}</div><div class="lbl">findings</div></div>
      <div class="card"><div class="num">{{.AffectedFiles}}</div><div class="lbl">affected files</div></div>
      <div class="card blocking"><div class="num">{{.Blocking}}</div><div class="lbl"><i></i>blocking</div></div>
      <div class="card warning"><div class="num">{{.Warnings}}</div><div class="lbl"><i></i>warning</div></div>
      <div class="card info"><div class="num">{{.Infos}}</div><div class="lbl"><i></i>info</div></div>
    </div>
  </div>

{{if eq .Total 0}}
  <section>
  <h2>Result</h2>
  {{if eq .FilesScanned 0}}
  <div class="empty">Nothing was scanned — every file was skipped (binary, oversized or unreadable); this is not a clean bill of health.</div>
  {{else}}
  <div class="clean">Clean ✓ — no findings from the deterministic detector set.</div>
  {{end}}
  </section>
{{else}}
  {{$max := (index .ByRule 0).Count}}
  <div class="split">
  <section id="rules">
  <h2>Findings by rule</h2>
  <table>
    <thead><tr><th>Rule</th><th>Severity</th><th class="num">Count</th></tr></thead>
    <tbody>
    {{range .ByRule}}<tr><td><code>{{.Rule}}</code></td><td><span class="sev {{.Severity}}">{{.Severity}}</span></td><td class="num"><span class="bar"><i class="{{.Severity}}" style="width: {{pct .Count $max}}%"></i></span>{{.Count}}</td></tr>
    {{end}}</tbody>
  </table>
  </section>

  <section id="directories">
  <h2>Findings by directory</h2>
  <table>
    <thead><tr><th>Directory</th><th class="num">Blocking</th><th class="num">Warning</th><th class="num">Info</th><th class="num">Weighted</th></tr></thead>
    <tbody>
    {{range .ByDir}}<tr><td><code>{{.Dir}}</code>{{if eq .Dir "."}}<span class="hint">repository root</span>{{end}}</td><td class="num{{if eq .Blocking 0}} zero{{end}}">{{.Blocking}}</td><td class="num{{if eq .Warnings 0}} zero{{end}}">{{.Warnings}}</td><td class="num{{if eq .Infos 0}} zero{{end}}">{{.Infos}}</td><td class="num strong">{{.Weighted}}</td></tr>
    {{end}}</tbody>
  </table>
  </section>
  </div>

  <section id="files">
  <h2>Top files</h2>
  <table>
    <thead><tr><th>File</th><th class="num">Blocking</th><th class="num">Warning</th><th class="num">Info</th><th class="num">Weighted</th></tr></thead>
    <tbody>
    {{range .TopFiles}}<tr><td><code>{{.File}}</code></td><td class="num{{if eq .Blocking 0}} zero{{end}}">{{.Blocking}}</td><td class="num{{if eq .Warnings 0}} zero{{end}}">{{.Warnings}}</td><td class="num{{if eq .Infos 0}} zero{{end}}">{{.Infos}}</td><td class="num strong">{{.Weighted}}</td></tr>
    {{end}}</tbody>
  </table>
  </section>

  <section id="detail">
  <h2>Detail</h2>
  <p class="lead">Source lines are quoted after masking — a secret the detectors recognize appears only as its <code>[MASKED:rule]</code> placeholder, never as its value.</p>
  <table class="detail">
    <thead><tr><th>File</th><th class="num">Line</th><th>Severity</th><th>Rule</th><th>Message</th></tr></thead>
    <tbody>
    {{range .Findings}}<tr{{if .Excerpt}} class="has-excerpt"{{end}}><td><code>{{.File}}</code></td><td class="num">{{.Line}}</td><td><span class="sev {{.Severity}}">{{.Severity}}</span></td><td><code>{{.Rule}}</code></td><td class="msg">{{.Message}}</td></tr>
    {{if .Excerpt}}<tr class="excerpt-row"><td colspan="5"><div class="excerpt {{.Severity}}"><span class="ln">{{.Line}}</span><code>{{range segments .Excerpt}}{{if .Masked}}<mark>{{.Text}}</mark>{{else}}{{.Text}}{{end}}{{end}}</code></div></td></tr>
    {{end}}{{end}}</tbody>
  </table>
  {{if gt .Truncated 0}}<p class="note">{{.Truncated}} more finding(s) omitted from this table; the aggregates above cover all of them.</p>{{end}}
  </section>
{{end}}
</main>
<footer class="wrap"><p>Generated offline by the MEMBRANE.AI Code Sweeper — the same deterministic detectors the platform runs in CI.</p></footer>
</body>
</html>
`))
