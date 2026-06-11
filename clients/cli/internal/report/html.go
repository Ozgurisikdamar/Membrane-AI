package report

import (
	"html/template"
	"io"
)

// RenderHTML writes the report as a single self-contained HTML page (inline
// CSS, no external assets) so it can be attached to a PR, mailed or hosted
// as-is. html/template escapes all repo-derived text.
func RenderHTML(w io.Writer, rep Report) error {
	return htmlTmpl.Execute(w, rep)
}

var htmlTmpl = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Generative-AI Technical-Debt Report — MEMBRANE.AI</title>
<style>
  :root {
    --ink: #1f2937; --muted: #6b7280; --line: #e5e7eb; --bg: #f8fafc;
    --card: #ffffff; --accent: #0f766e;
    --blocking-bg: #fee2e2; --blocking-ink: #b91c1c;
    --warning-bg: #fef3c7;  --warning-ink: #b45309;
    --info-bg: #dbeafe;     --info-ink: #1d4ed8;
  }
  * { box-sizing: border-box; }
  body { margin: 0; padding: 40px 24px; background: var(--bg); color: var(--ink);
         font: 15px/1.6 -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
  main { max-width: 960px; margin: 0 auto; }
  h1 { font-size: 26px; margin: 0 0 4px; }
  h2 { font-size: 18px; margin: 36px 0 12px; }
  .sub { color: var(--muted); margin-bottom: 28px; }
  .sub code { background: var(--card); border: 1px solid var(--line); border-radius: 6px; padding: 1px 6px; }
  .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 14px; }
  .card { background: var(--card); border: 1px solid var(--line); border-radius: 12px; padding: 16px 18px; }
  .card .num { font-size: 26px; font-weight: 700; }
  .card .lbl { color: var(--muted); font-size: 13px; }
  .grade { display: flex; align-items: center; gap: 18px; background: var(--card);
           border: 1px solid var(--line); border-radius: 12px; padding: 18px 22px; margin-bottom: 18px; }
  .grade .letter { font-size: 44px; font-weight: 800; color: var(--accent); }
  .grade .expl { color: var(--muted); font-size: 13px; }
  table { width: 100%; border-collapse: collapse; background: var(--card);
          border: 1px solid var(--line); border-radius: 12px; overflow: hidden; }
  th, td { text-align: left; padding: 9px 14px; border-top: 1px solid var(--line); }
  thead th { background: #f1f5f9; border-top: none; font-size: 13px; color: var(--muted);
             text-transform: uppercase; letter-spacing: 0.04em; }
  td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; }
  .sev { display: inline-block; border-radius: 999px; padding: 1px 10px; font-size: 12px; font-weight: 600; }
  .sev.blocking { background: var(--blocking-bg); color: var(--blocking-ink); }
  .sev.warning { background: var(--warning-bg); color: var(--warning-ink); }
  .sev.info { background: var(--info-bg); color: var(--info-ink); }
  .clean { background: #dcfce7; color: #15803d; border-radius: 12px; padding: 18px 22px; font-weight: 600; }
  .empty { background: #fef3c7; color: #b45309; border-radius: 12px; padding: 18px 22px; font-weight: 600; }
  .note { color: var(--muted); font-size: 13px; margin-top: 10px; }
  footer { color: var(--muted); font-size: 13px; margin-top: 40px; }
  code { font: 13px/1.5 ui-monospace, "Cascadia Code", Consolas, monospace; }
</style>
</head>
<body>
<main>
  <h1>Generative-AI Technical-Debt Report</h1>
  <p class="sub">MEMBRANE.AI Code Sweeper · <code>{{.Root}}</code> · generated {{.GeneratedAt.UTC.Format "2006-01-02 15:04 UTC"}}</p>

  <div class="grade">
    <div class="letter">{{.Grade}}</div>
    <div>
      <div>Debt score <strong>{{printf "%.1f" .DebtScore}}</strong> — severity-weighted findings per 100 scanned files.</div>
      <div class="expl">{{.ScoringLegend}}</div>
    </div>
  </div>

  <div class="cards">
    <div class="card"><div class="num">{{.FilesScanned}}</div><div class="lbl">files scanned</div></div>
    <div class="card"><div class="num">{{.Total}}</div><div class="lbl">findings</div></div>
    <div class="card"><div class="num">{{.Blocking}}</div><div class="lbl">blocking</div></div>
    <div class="card"><div class="num">{{.Warnings}}</div><div class="lbl">warning</div></div>
    <div class="card"><div class="num">{{.Infos}}</div><div class="lbl">info</div></div>
    <div class="card"><div class="num">{{.AffectedFiles}}</div><div class="lbl">affected files</div></div>
  </div>

{{if eq .Total 0}}
  <h2>Result</h2>
  {{if eq .FilesScanned 0}}
  <div class="empty">Nothing was scanned — every file was skipped (binary, oversized or unreadable); this is not a clean bill of health.</div>
  {{else}}
  <div class="clean">Clean ✓ — no findings from the deterministic detector set.</div>
  {{end}}
{{else}}
  <h2>Findings by rule</h2>
  <table>
    <thead><tr><th>Rule</th><th>Severity</th><th class="num">Count</th></tr></thead>
    <tbody>
    {{range .ByRule}}<tr><td><code>{{.Rule}}</code></td><td><span class="sev {{.Severity}}">{{.Severity}}</span></td><td class="num">{{.Count}}</td></tr>
    {{end}}</tbody>
  </table>

  <h2>Findings by directory</h2>
  <table>
    <thead><tr><th>Directory</th><th class="num">Blocking</th><th class="num">Warning</th><th class="num">Info</th><th class="num">Weighted</th></tr></thead>
    <tbody>
    {{range .ByDir}}<tr><td><code>{{.Dir}}</code></td><td class="num">{{.Blocking}}</td><td class="num">{{.Warnings}}</td><td class="num">{{.Infos}}</td><td class="num">{{.Weighted}}</td></tr>
    {{end}}</tbody>
  </table>

  <h2>Top files</h2>
  <table>
    <thead><tr><th>File</th><th class="num">Blocking</th><th class="num">Warning</th><th class="num">Info</th><th class="num">Weighted</th></tr></thead>
    <tbody>
    {{range .TopFiles}}<tr><td><code>{{.File}}</code></td><td class="num">{{.Blocking}}</td><td class="num">{{.Warnings}}</td><td class="num">{{.Infos}}</td><td class="num">{{.Weighted}}</td></tr>
    {{end}}</tbody>
  </table>

  <h2>Detail</h2>
  <table>
    <thead><tr><th>File</th><th class="num">Line</th><th>Severity</th><th>Rule</th><th>Message</th></tr></thead>
    <tbody>
    {{range .Findings}}<tr><td><code>{{.File}}</code></td><td class="num">{{.Line}}</td><td><span class="sev {{.Severity}}">{{.Severity}}</span></td><td><code>{{.Rule}}</code></td><td>{{.Message}}</td></tr>
    {{end}}</tbody>
  </table>
  {{if gt .Truncated 0}}<p class="note">{{.Truncated}} more finding(s) omitted from this table; the aggregates above cover all of them.</p>{{end}}
{{end}}

  <footer>Generated offline by the MEMBRANE.AI Code Sweeper — the same deterministic detectors the platform runs in CI.</footer>
</main>
</body>
</html>
`))
