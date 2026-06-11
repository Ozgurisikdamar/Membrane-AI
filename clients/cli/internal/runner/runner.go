// Package runner walks a directory tree and runs the pkg/scan detectors over
// every scannable file — the offline core of the Code Sweeper CLI.
package runner

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

// DefaultMaxFileSize bounds a scanned file (larger files are skipped).
const DefaultMaxFileSize = 1 << 20 // 1 MiB

// skipDirs are never descended into: dependency trees and build output would
// drown real findings in third-party noise.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".venv": true, "venv": true,
	"dist": true, "build": true, "target": true, "__pycache__": true, ".task": true,
	".idea": true, ".vscode": true,
}

// languageByExt maps file extensions to the language tags pkg/scan understands.
var languageByExt = map[string]string{
	".go": "go", ".py": "python", ".js": "javascript", ".ts": "typescript",
	".jsx": "javascript", ".tsx": "typescript", ".java": "java", ".rb": "ruby",
	".php": "php", ".cs": "csharp", ".sql": "sql", ".sh": "shell", ".ps1": "powershell",
	".yaml": "yaml", ".yml": "yaml", ".tf": "terraform", ".env": "dotenv",
}

// Options configures a scan run.
type Options struct {
	Root        string
	MaxFileSize int64 // 0 = DefaultMaxFileSize
}

// FileFinding is one finding located in a file.
type FileFinding struct {
	File string `json:"file"`
	Line int    `json:"line"`

	Rule     string        `json:"rule"`
	Severity scan.Severity `json:"severity"`
	Message  string        `json:"message"`
}

// Result is the aggregate outcome of a scan run.
type Result struct {
	Root         string        `json:"root"`
	FilesScanned int           `json:"files_scanned"`
	FilesSkipped int           `json:"files_skipped"`
	Findings     []FileFinding `json:"findings"`
	Blocking     int           `json:"blocking"`
	Warnings     int           `json:"warnings"`
	Infos        int           `json:"infos"`
}

// Run walks opts.Root and scans every eligible file with the default detector
// chain. Findings are sorted by file then line for stable output.
func Run(ctx context.Context, opts Options) (Result, error) {
	maxSize := opts.MaxFileSize
	if maxSize <= 0 {
		maxSize = DefaultMaxFileSize
	}
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Result{}, fmt.Errorf("stat root: %w", err)
	}

	res := Result{Root: root}
	detectors := scan.Default()

	scanOne := func(path string) error {
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		content, rerr := os.ReadFile(path) //nolint:gosec // walking a user-chosen tree is the CLI's job
		if rerr != nil {
			res.FilesSkipped++
			return nil // unreadable file: skip, never abort the whole scan
		}
		if isBinary(content) {
			res.FilesSkipped++
			return nil
		}
		res.FilesScanned++

		language := languageByExt[strings.ToLower(filepath.Ext(path))]
		findings, _, dErr := scan.Run(ctx, string(content), language, detectors...)
		if dErr != nil {
			return dErr // context cancellation only
		}
		for _, f := range findings {
			res.Findings = append(res.Findings, FileFinding{
				File: rel, Line: f.Line, Rule: f.Rule, Severity: f.Severity, Message: f.Message,
			})
		}
		return nil
	}

	if !info.IsDir() {
		if err := scanOne(root); err != nil {
			return Result{}, err
		}
		res.Root = filepath.Dir(root)
	} else {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil // unreadable entry: skip
			}
			if d.IsDir() {
				if skipDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if fi, ferr := d.Info(); ferr != nil || fi.Size() > maxSize {
				res.FilesSkipped++
				return nil
			}
			return scanOne(path)
		})
		if err != nil {
			return Result{}, err
		}
	}

	sort.Slice(res.Findings, func(i, j int) bool {
		if res.Findings[i].File != res.Findings[j].File {
			return res.Findings[i].File < res.Findings[j].File
		}
		return res.Findings[i].Line < res.Findings[j].Line
	})
	for _, f := range res.Findings {
		switch f.Severity {
		case scan.SeverityBlocking:
			res.Blocking++
		case scan.SeverityWarning:
			res.Warnings++
		case scan.SeverityInfo:
			res.Infos++
		}
	}
	return res, nil
}

// isBinary treats content with a NUL byte in its head as non-text.
func isBinary(content []byte) bool {
	head := content
	if len(head) > 8192 {
		head = head[:8192]
	}
	return bytes.IndexByte(head, 0) >= 0
}
