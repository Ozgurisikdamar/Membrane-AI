// Command membrane is the MEMBRANE.AI Code Sweeper CLI: an offline,
// dependency-free scan of a repository or file using the same deterministic
// detectors the platform runs (pkg/scan). CI-friendly: exit code 1 when the
// scan finds findings at or above --fail-on.
//
// Usage:
//
//	membrane scan [path] [--json] [--report md|html] [--out file] [--fail-on=blocking|warning|never]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/report"
	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/runner"
)

// sanitize neutralizes control characters (incl. ANSI/OSC escapes) in
// repo-derived text before it reaches the terminal — an adversarial repo
// must not be able to rewrite or hide scan output.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}

// Exit codes (stable contract for CI pipelines).
const (
	exitClean = 0
	exitFail  = 1
	exitUsage = 2
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 || args[0] != "scan" {
		usage()
		return exitUsage
	}

	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON")
	reportFmt := fs.String("report", "", "emit a technical-debt report instead of plain output: md|html")
	outPath := fs.String("out", "", "write the report to this file instead of stdout (requires --report)")
	failOn := fs.String("fail-on", "blocking", "exit non-zero on findings at/above this severity: blocking|warning|never")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	// Two-pass parse so flags work both before AND after the path argument
	// (stdlib flag stops at the first positional).
	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
		if err := fs.Parse(fs.Args()[1:]); err != nil {
			return exitUsage
		}
		// A leftover positional would silently drop that path AND every flag
		// after it — in CI that is a false-clean. Refuse loudly instead.
		if fs.NArg() > 0 {
			fmt.Fprintln(os.Stderr, "unexpected argument:", fs.Arg(0), "(scan takes a single path)")
			return exitUsage
		}
	}
	if *failOn != "blocking" && *failOn != "warning" && *failOn != "never" {
		fmt.Fprintln(os.Stderr, "invalid --fail-on:", *failOn)
		return exitUsage
	}
	if *reportFmt != "" && *reportFmt != "md" && *reportFmt != "html" {
		fmt.Fprintln(os.Stderr, "invalid --report:", *reportFmt)
		return exitUsage
	}
	if *jsonOut && *reportFmt != "" {
		fmt.Fprintln(os.Stderr, "--json and --report are mutually exclusive")
		return exitUsage
	}
	if *outPath != "" && *reportFmt == "" {
		fmt.Fprintln(os.Stderr, "--out requires --report")
		return exitUsage
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	res, err := runner.Run(ctx, runner.Options{Root: root})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return exitUsage
	}

	switch {
	case *jsonOut:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(os.Stderr, "encode failed:", err)
			return exitUsage
		}
	case *reportFmt != "":
		if err := writeReport(res, *reportFmt, *outPath); err != nil {
			fmt.Fprintln(os.Stderr, "report failed:", err)
			return exitUsage
		}
	default:
		render(res)
	}
	return decide(*failOn, res)
}

// writeReport renders the technical-debt report to outPath (or stdout when
// empty). Rendering goes to memory first so a render error never leaves a
// truncated file behind, and the write (including close) is checked before
// the confirmation line claims success.
func writeReport(res runner.Result, format, outPath string) error {
	rep := report.Build(res, time.Now().UTC())

	var buf bytes.Buffer
	var err error
	if format == "md" {
		err = report.RenderMarkdown(&buf, rep)
	} else {
		err = report.RenderHTML(&buf, rep)
	}
	if err != nil {
		return err
	}

	if outPath == "" {
		_, err = os.Stdout.Write(buf.Bytes())
		return err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o600); err != nil {
		return err
	}
	fmt.Printf("technical-debt report written to %s (%d finding(s), grade %s)\n",
		outPath, rep.Total, rep.Grade)
	return nil
}

// decide maps the scan outcome to an exit code per the --fail-on policy.
func decide(failOn string, res runner.Result) int {
	switch failOn {
	case "warning":
		if res.Blocking+res.Warnings > 0 {
			return exitFail
		}
	case "blocking":
		if res.Blocking > 0 {
			return exitFail
		}
	case "never":
	}
	return exitClean
}

func render(res runner.Result) {
	for _, f := range res.Findings {
		fmt.Printf("%s:%d  [%s] %s — %s\n",
			sanitize(f.File), f.Line, f.Severity, f.Rule, sanitize(f.Message))
	}
	if len(res.Findings) > 0 {
		fmt.Println()
	}
	fmt.Printf("MEMBRANE.AI Code Sweeper — scanned %d file(s), skipped %d\n", res.FilesScanned, res.FilesSkipped)
	fmt.Printf("findings: %d blocking, %d warning, %d info\n", res.Blocking, res.Warnings, res.Infos)
	if res.Blocking == 0 && res.Warnings == 0 && res.Infos == 0 {
		fmt.Println("clean ✓")
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `MEMBRANE.AI Code Sweeper

usage:
  membrane scan [path] [--json] [--report md|html] [--out file] [--fail-on=blocking|warning|never]

  --report md|html   emit the Generative-AI Technical-Debt Report
  --out file         write the report to a file (default stdout)

exit codes: 0 clean (per --fail-on) · 1 findings · 2 usage/error`)
}
