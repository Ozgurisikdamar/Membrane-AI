# Compliance posture & roadmap

MEMBRANE.AI is built to clear the bar for regulated buyers. This maps product
controls already in the codebase to the frameworks customers ask about, and
states what remains for formal certification (an organizational audit process,
not code).

## Implemented technical controls

| Control area | How MEMBRANE.AI implements it | Where |
| --- | --- | --- |
| **Audit logging** | Every governance decision is recorded (gateway `AuditSink`); every verdict writes a `verdict_audit` row in one ACID tx with the outbox (D-013). | `services/gateway`, `services/orchestrator` + `deploy/migrations` |
| **Secret handling** | Detected secrets are masked (`[MASKED:<rule>]`) before any LLM tier sees the diff; the privacy invariant is enforced structurally end-to-end (D-024) and never logged (ENGINEERING-STANDARDS §5). | `pkg/scan`, `services/analyzer`, `services/semantic` |
| **Least privilege (runtime)** | All service pods run non-root, read-only rootfs, all caps dropped, no service-account token, seccomp RuntimeDefault (D-030). | `deploy/helm/membrane` |
| **Encryption in transit** | Tracing exports default to TLS (secure-by-default, D-032); service-to-service can run under a mesh; the package firewall + tool-call governance constrain egress (D-034). | `pkg/observability`, `services/gateway` |
| **Data residency / isolation** | Three deployment postures — masked SaaS, private VPC, air-gapped — so customer code never leaves their boundary when required (D-011). | `deploy/terraform`, `deploy/helm` |
| **Change governance** | Shadow→enforce merge gate (D-035) + commit-status/PR gating give an auditable, reversible policy rollout (D-025). | `services/reporter` |
| **Supply-chain** | Package-install firewall (deny-list + typosquat detection) and shadow-AI discovery (D-034); reproducible, checksum-signed releases (D-033). | `services/gateway`, `clients/cli/.goreleaser.yaml` |
| **Quality assurance** | Accuracy SLO gate (precision/recall/FP-rate) in CI; ≥80% domain/app coverage bar; strict lint/typecheck. | `eval/`, `docs/ENGINEERING-STANDARDS.md` |

## Framework mapping (selected)

- **SOC 2 (Security, Availability, Confidentiality)** — audit logging, least
  privilege, encryption-in-transit, change governance, and the outbox's
  at-least-once durability map to CC6 (logical access), CC7 (operations), and
  the availability/confidentiality criteria.
- **ISO/IEC 27001** — Annex A controls for access control (A.9 / 8.x),
  cryptography (A.10), operations security (A.12), and supplier/supply-chain
  (A.15) are addressed by the controls above.
- **ISO/IEC 42001 (AI management)** — model governance (three-tier gate, D-008),
  AI-usage inventory (shadow-AI discovery), and human oversight (needs-review +
  shadow mode) map to the AI lifecycle and risk controls.

## Remaining for certification (process, not code)

Policies & procedures (access reviews, incident response, vendor management),
a risk assessment, evidence collection over an audit window, penetration
testing, and an accredited auditor engagement. The technical controls above are
the evidence base; certification is the organizational program on top.
