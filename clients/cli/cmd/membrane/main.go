// Command membrane is the MEMBRANE.AI Code Sweeper CLI: an offline,
// dependency-free scan of a repository or file using the same deterministic
// detectors the platform runs (pkg/scan). CI-friendly: exit code 1 when the
// scan finds findings at or above --fail-on.
//
// Usage:
//
//	membrane scan [path] [--json] [--fail-on=blocking|warning|never]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ozgurisikdamar/Membrane-AI/clients/cli/internal/runner"
)

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
	}
	if *failOn != "blocking" && *failOn != "warning" && *failOn != "never" {
		fmt.Fprintln(os.Stderr, "invalid --fail-on:", *failOn)
		return exitUsage
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	res, err := runner.Run(ctx, runner.Options{Root: root})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return exitUsage
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(os.Stderr, "encode failed:", err)
			return exitUsage
		}
	} else {
		render(res)
	}
	return decide(*failOn, res)
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
		fmt.Printf("%s:%d  [%s] %s — %s\n", f.File, f.Line, f.Severity, f.Rule, f.Message)
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
  membrane scan [path] [--json] [--fail-on=blocking|warning|never]

exit codes: 0 clean (per --fail-on) · 1 findings · 2 usage/error`)
}
