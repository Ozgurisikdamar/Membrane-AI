# MEMBRANE.AI

**Real-time security and architectural guardrails for AI coding agents.**

![status](https://img.shields.io/badge/status-early%20development-orange)
![platform](https://img.shields.io/badge/cloud-AWS%20%2F%20EKS-232F3E)
![deploy](https://img.shields.io/badge/deploy-SaaS%20%7C%20VPC%20%7C%20air--gapped-0FB5A6)
![license](https://img.shields.io/badge/license-see%20LICENSE-blue)

MEMBRANE.AI is an autonomous **"architectural & security immune system"** for the
generative-AI era. As Cursor, Claude Code, Copilot and semi-autonomous agents write a
fast-growing share of production code, MEMBRANE.AI governs that code **at the point of
generation** and **in CI/CD** — so every change is secure *and* aligned with your
codebase's own gold-standard patterns, without slowing developers down.

---

## Why it exists

- Independent testing finds roughly **45% of AI-generated code fails security checks**
  and introduces OWASP Top 10 vulnerabilities — a rate that has stayed flat for two years.
- AI-written code carries about **2.7× more vulnerabilities** than human-written code.
- Adoption is outrunning governance: a growing majority of code is now AI-assisted, while
  only a small minority of organizations have a formal AI-code governance policy.
- Malware has begun **weaponizing the developer AI toolchain** (e.g. the Nx "s1ngularity"
  and Bitwarden CLI supply-chain attacks), turning the workstation and the agent's
  tool-call boundary into first-class attack vectors that post-commit AppSec cannot govern.

Traditional, signature-based SAST runs *after* commit and is blind to semantic
vulnerabilities, prompt injection in code, and architectural drift. MEMBRANE.AI is built
for the new failure modes.

## What it does

- **Dual-point governance** — a proactive prompt/MCP gateway intercepts the AI request and
  injects architectural context *before* the model answers, plus deterministic AST +
  semantic verification on every diff from the IDE and CI/CD.
- **Gold-codebase RAG** — a per-organization `pgvector` index of your own best
  implementations grounds every judgment in *your* standards, not a generic rulebook.
- **Cost-managed multi-model consensus** — tiered gating (cache → context pruning → local
  model triage → premium consensus) makes deep semantic review economically viable at
  enterprise push volumes.
- **MCP & agentic-threat governance** — MCP-gateway security, tool-call governance, a
  package firewall against hallucinated/malicious dependencies, and shadow-AI discovery.
- **Deployment optionality** — masked SaaS, private VPC, or fully **air-gapped** on-prem,
  so regulated industries (FinTech, MedTech, defense) can adopt it.

## How it works (high level)

```
IDE / Prompt Gateway  ──▶  Inbound (Route 53 · ALB · Envoy)  ──▶  Event bus (Kafka/Redpanda)
        │                                                                   │
        ▼                                                                   ▼
Source-control webhooks                              Orchestrator (Go, Saga state machine)
                                                       ├─ Static analyzer (AST + data masking)
                                                       ├─ Vector context (Aurora pgvector / HNSW)
                                                       └─ Semantic engine (local vLLM + cloud consensus)
                                                                   │
                                          Redis (Blake3 cache)  ·  Aurora PostgreSQL (state + gold index)
                                                                   │
                                                                   ▼
                              Report & remediation  ──▶  IDE fixes · PR status/comments · Slack/Jira/SIEM
```

## Tech stack

| Layer | Choice |
| --- | --- |
| Core services & orchestration | Go (Golang) |
| Semantic AI interface | Python 3.12+ / FastAPI |
| Edge gateway | Envoy Proxy |
| Event bus | Redpanda / Apache Kafka |
| Cache & state | Redis Enterprise · Amazon Aurora PostgreSQL (pgvector) |
| Local inference | vLLM on EC2 Inf2 / GPU |
| Infrastructure | AWS EKS, multi-AZ VPC, Terraform + Helm |

## Deployment models

| Model | Isolation | Summary |
| --- | --- | --- |
| Masked SaaS | Lowest | Secrets masked locally; only cleaned code leaves the perimeter; zero-data-retention LLM APIs. |
| Private VPC | High | Full stack runs in the customer's AWS account; cloud-LLM access via PrivateLink. |
| Air-gapped / on-prem | Maximum | No external API calls; 100% local fine-tuned open models. |

## Roadmap

Development is tracked on the project board:
**https://github.com/users/Ozgurisikdamar/projects/5**

## Status

Early development — architecture and specification defined; implementation in progress.

## License

See [LICENSE](./LICENSE).
