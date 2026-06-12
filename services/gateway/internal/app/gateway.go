// Package app holds the gateway use-cases: it applies the governance policy
// (domain) to each request and records the verdict to the audit sink. The
// policy is injected (no globals); the sink is a port (no I/O coupling).
package app

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/ports"
)

// Gateway governs MCP tool calls, package installs and AI-agent usage.
type Gateway struct {
	policy domain.Policy
	audit  ports.AuditSink
}

// NewGateway wires the policy and audit sink.
func NewGateway(policy domain.Policy, audit ports.AuditSink) *Gateway {
	return &Gateway{policy: policy, audit: audit}
}

// GovernToolCall decides on an MCP tool call and audits the verdict.
func (g *Gateway) GovernToolCall(ctx context.Context, c domain.ToolCall) (domain.Verdict, error) {
	v := domain.EvaluateToolCall(g.policy, c)
	if err := g.audit.Record(ctx, "tool-call", c.Tool, v); err != nil {
		return domain.Verdict{}, err
	}
	return v, nil
}

// ScreenPackage decides on a package install and audits the verdict.
func (g *Gateway) ScreenPackage(ctx context.Context, p domain.Package) (domain.Verdict, error) {
	v := domain.ScreenPackage(g.policy, p)
	if err := g.audit.Record(ctx, "package", p.Ecosystem+":"+p.Name, v); err != nil {
		return domain.Verdict{}, err
	}
	return v, nil
}

// RecordShadow classifies an AI-agent usage signal and audits it.
func (g *Gateway) RecordShadow(ctx context.Context, s domain.ShadowSignal) (domain.Verdict, error) {
	v := domain.ClassifyShadow(g.policy, s)
	if err := g.audit.Record(ctx, "shadow-ai", s.Agent, v); err != nil {
		return domain.Verdict{}, err
	}
	return v, nil
}
