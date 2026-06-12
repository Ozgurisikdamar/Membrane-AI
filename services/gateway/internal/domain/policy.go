// Package domain holds the pure governance logic of the MCP gateway: it decides
// whether an agent's tool call is allowed, screens a package install against a
// firewall policy, and classifies AI-agent usage as sanctioned or shadow. No
// I/O lives here (ENGINEERING-STANDARDS §1/§8) — decisions are deterministic
// functions over a Policy, so they are exhaustively table-testable.
package domain

import (
	"strings"
)

// Decision is the gateway's verdict on a governed action.
type Decision string

const (
	// DecisionAllow lets the action proceed.
	DecisionAllow Decision = "allow"
	// DecisionDeny blocks the action outright.
	DecisionDeny Decision = "deny"
	// DecisionReview pauses for human approval (the enforcement gate's "ask").
	DecisionReview Decision = "review"
)

// Verdict is a decision plus the reasons behind it (auditable, explainable).
type Verdict struct {
	Decision Decision
	Reasons  []string
}

func allow(reason string) Verdict { return Verdict{Decision: DecisionAllow, Reasons: []string{reason}} }
func deny(reason string) Verdict  { return Verdict{Decision: DecisionDeny, Reasons: []string{reason}} }
func review(reason string) Verdict {
	return Verdict{Decision: DecisionReview, Reasons: []string{reason}}
}

// ToolCall is a proposed MCP tool invocation by an agent.
type ToolCall struct {
	Tool   string
	Args   map[string]string
	Server string
	Agent  string
}

// Package is a proposed dependency install.
type Package struct {
	Ecosystem string // npm | pypi | go | ...
	Name      string
	Version   string
}

// ShadowSignal is an observed AI-agent/tool usage event.
type ShadowSignal struct {
	Agent  string
	Source string // where it was observed (ide, ci, endpoint, …)
	User   string
}

// Policy is the governance configuration (loaded from config; default-deny when
// an allowlist is set, deny-list + heuristics otherwise). JSON-tagged so an
// operator can supply it as a file (MEMBRANE_GATEWAY_POLICY_FILE).
type Policy struct {
	DeniedTools          []string `json:"denied_tools"`
	AllowedTools         []string `json:"allowed_tools"` // non-empty ⇒ default-deny unknown tools
	DangerousArgPatterns []string `json:"dangerous_arg_patterns"`

	DeniedPackages  map[string][]string `json:"denied_packages"`  // ecosystem → deny
	AllowedPackages map[string][]string `json:"allowed_packages"` // ecosystem → allow
	PopularPackages map[string][]string `json:"popular_packages"` // ecosystem → typosquat baseline
	RiskyNameParts  []string            `json:"risky_name_parts"`

	ApprovedAgents []string `json:"approved_agents"` // sanctioned agents (others are shadow)
}

// DefaultPolicy is a sensible built-in baseline used when no policy file is
// configured: no tool allowlist (open except dangerous/denied), high-signal
// destructive-arg patterns, popular-package typosquat baselines, and the
// common sanctioned AI agents.
func DefaultPolicy() Policy {
	return Policy{
		DangerousArgPatterns: []string{
			"rm -rf", "| sh", "| bash", "curl | sh", "wget | sh",
			":(){:|:&};:", "mkfs", "dd if=", "/etc/shadow", "/etc/passwd",
		},
		PopularPackages: map[string][]string{
			"npm":  {"react", "lodash", "express", "axios", "chalk", "commander", "webpack", "next", "vue", "typescript"},
			"pypi": {"requests", "numpy", "pandas", "flask", "django", "boto3", "pytest", "fastapi", "pydantic", "urllib3"},
		},
		RiskyNameParts: []string{"crypto-miner", "keylogger", "stealer", "backdoor"},
		ApprovedAgents: []string{"claude-code", "cursor", "github-copilot"},
	}
}

// EvaluateToolCall governs an MCP tool call. Order: explicit deny → dangerous
// argument → allowlist miss → allow.
func EvaluateToolCall(p Policy, c ToolCall) Verdict {
	if contains(p.DeniedTools, c.Tool) {
		return deny("tool '" + c.Tool + "' is on the deny list")
	}
	for _, arg := range c.Args {
		low := strings.ToLower(arg)
		for _, pat := range p.DangerousArgPatterns {
			if pat != "" && strings.Contains(low, strings.ToLower(pat)) {
				return deny("argument contains a dangerous pattern: " + pat)
			}
		}
	}
	if len(p.AllowedTools) > 0 && !contains(p.AllowedTools, c.Tool) {
		return deny("tool '" + c.Tool + "' is not on the allow list (default-deny)")
	}
	return allow("tool call permitted by policy")
}

// ScreenPackage is the package-install firewall. Order: explicit deny →
// typosquat of a popular package → risky name → allowlist miss → allow.
func ScreenPackage(p Policy, pkg Package) Verdict {
	if contains(p.DeniedPackages[pkg.Ecosystem], pkg.Name) {
		return deny("package '" + pkg.Name + "' is on the deny list")
	}
	for _, popular := range p.PopularPackages[pkg.Ecosystem] {
		if pkg.Name != popular && nearMiss(pkg.Name, popular) {
			return deny("package '" + pkg.Name + "' looks like a typosquat of '" + popular + "'")
		}
	}
	low := strings.ToLower(pkg.Name)
	for _, part := range p.RiskyNameParts {
		if part != "" && strings.Contains(low, strings.ToLower(part)) {
			return review("package name contains a suspicious token: " + part)
		}
	}
	if allowed := p.AllowedPackages[pkg.Ecosystem]; len(allowed) > 0 && !contains(allowed, pkg.Name) {
		return review("package '" + pkg.Name + "' is not on the allow list")
	}
	return allow("package permitted by policy")
}

// ClassifyShadow flags unsanctioned AI usage. Approved agents pass; everything
// else is shadow AI and surfaces for governance review.
func ClassifyShadow(p Policy, s ShadowSignal) Verdict {
	if contains(p.ApprovedAgents, s.Agent) {
		return allow("agent '" + s.Agent + "' is sanctioned")
	}
	return review("shadow AI: '" + s.Agent + "' is not in the approved registry")
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

// nearMiss reports whether a and b differ by a single edit (insert/delete/
// substitute) — a cheap typosquat signal. Equal strings are not near-misses.
func nearMiss(a, b string) bool {
	if a == b {
		return false
	}
	return editDistanceWithin1(a, b)
}

// editDistanceWithin1 returns true iff the Levenshtein distance between a and b
// is exactly 1 (bounded check — O(n), no full DP matrix).
func editDistanceWithin1(a, b string) bool {
	la, lb := len(a), len(b)
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}
	if lb-la > 1 {
		return false
	}
	if la == lb {
		diffs := 0
		for i := range a {
			if a[i] != b[i] {
				diffs++
				if diffs > 1 {
					return false
				}
			}
		}
		return diffs == 1
	}
	// lengths differ by exactly 1: b has one extra char — try to align.
	for i := 0; i < la; i++ {
		if a[i] != b[i] {
			return a[i:] == b[i+1:]
		}
	}
	return true
}
