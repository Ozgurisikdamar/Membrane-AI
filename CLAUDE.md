# CLAUDE.md — MEMBRANE.AI Session Bootstrap

> **You are working on MEMBRANE.AI** — real-time security and architectural guardrails for AI coding
> agents: an "architectural & security immune system" that governs AI-generated code at the point of
> generation (prompt/MCP gateway) and again in CI/CD (AST + semantic consensus), grounded in each
> organization's own "gold codebase". Private repo: `github.com/Ozgurisikdamar/Membrane-AI`.
> Local working copy: **`C:\Dev\Membrane-AI`** (this directory).

---

## ⚡ THE "DEVAM ET" PROTOCOL (read this first)

When the user says **"devam et"** (or "continue", or gives no specific instruction), do exactly this:

1. **Read `docs/HANDOVER.md`.** It holds the current state and a precise **Next up** section.
2. **Execute "Next up" immediately** — do not ask what to do, do not list options, do not re-plan what
   is already planned. Consult `docs/ROADMAP.md` (what), `docs/ARCHITECTURE.md` (how it fits),
   `docs/ENGINEERING-STANDARDS.md` (how well), `docs/SKILLS.md` (commands) as needed.
3. **Never re-ask a settled question.** All decisions live in `docs/DECISIONS.md`. If your question is
   answered there, it is final. Only ask the user when something is genuinely new and blocking.
4. **At every milestone** (feature complete, or session ending):
   - Run quality gates: `task lint` + `task test` (+ `task build`) — must be clean.
   - Commit locally (rules below).
   - **Update `docs/HANDOVER.md`**: what was done, exact current state, the new "Next up" (specific
     enough that a cold-start agent needs zero questions), any blockers.
   - Tick `docs/ROADMAP.md` checkboxes for anything that changed state.
5. **Push ONLY when the user says "pushla".** Never push, open PRs, or merge without that word.

> A session that ends without updating `docs/HANDOVER.md` is a failed session — that file is the only
> memory the next session has.

## 🔒 Hard rules (non-negotiable, every session, any model)

| Rule | Detail |
| --- | --- |
| **Reply language** | Always answer the user in **Turkish**. Code, docs, comments, commit messages: English. |
| **Commit identity** | Author/committer `Özgür Işık Damar <74007174+Ozgurisikdamar@users.noreply.github.com>` (already global — don't override). |
| **No AI trailers** | NEVER add `Co-Authored-By: Claude`, `Generated with…`, or any AI signature to commits/PRs. |
| **Commit style** | Short, human, English, one line ≤70 chars (e.g. `Add ingestion gRPC stream adapter`). Commit at milestones, not every micro-step. |
| **Push policy** | Local commits anytime; `git push` only on explicit **"pushla"**. |
| **History** | Never rewrite published history or touch other contributors' commits. |
| **Quality gates** | `task lint` + `task test` must pass before any commit. No commented-out code, no `TODO` without a ROADMAP/issue reference, no skipped tests. |
| **Standards** | `docs/ENGINEERING-STANDARDS.md` is binding: hexagonal services, SOLID, DI, typed errors, transactional outbox, table-driven tests, ≥80% domain/app coverage. |
| **Secrets** | Never commit tokens/keys; `.env*` is gitignored. If a GitHub op fails (401/403/404-on-private), ask the user for a **classic PAT with `repo`+`project` scopes** (fine-grained PATs were rejected), `gh auth login --with-token`, and remind them to revoke it afterwards. |

## 📚 Document map

| File | Purpose |
| --- | --- |
| `docs/HANDOVER.md` | **Living session state — start here on "devam et".** |
| `docs/ROADMAP.md` | Phases P0–P3 over the 20 backlog items; checkbox status. Source of truth; the GitHub board mirrors it. |
| `docs/ARCHITECTURE.md` | Components, event flow, data design, monorepo layout, naming/port/topic conventions, cross-platform build rules. |
| `docs/ENGINEERING-STANDARDS.md` | The binding quality bar: SOLID, hexagonal, design patterns, ACID/outbox, testing, lint, API/versioning. |
| `docs/SKILLS.md` | Engineering playbook: toolchain install, every `task` command, local dev, codegen, "add a service" checklist. |
| `docs/DECISIONS.md` | ADR-lite log of settled decisions. Do not relitigate. |
| `docs/report/report.md` | Full product/business/technical master report (markdown + figures). |
| `docs/blueprint/blueprint.md` | Visual system blueprint: story flow, architecture, sequence, data, deployment. |
| `docs/*.docx` | Human-facing Word versions of the two documents above. |

## 🛠 Environment facts (this machine)

- **Host:** Windows 11, PowerShell 7 (`pwsh`). Use PowerShell syntax for shell; the Bash tool also exists.
- **Repo path:** `C:\Dev\Membrane-AI` — kept OUTSIDE OneDrive on purpose (git/node speed). Do not move it.
- **Go:** 1.25 installed. Tools live in `%USERPROFILE%\go\bin` (on PATH): `task`, `buf`, `protoc-gen-go`,
  `protoc-gen-go-grpc`, `golangci-lint`. If any is missing, `go install` it (see `docs/SKILLS.md`).
- **Python:** 3.11 at `C:\Users\isiko\AppData\Local\Programs\Python\Python311\python.exe` (scripts); the
  semantic service targets Python 3.12+ in containers.
- **Docker:** installed. Compose dev stack in `deploy/compose/`. If Docker isn't running, author files
  anyway and let CI validate — never block on Docker.
- **Word + pandoc + PyMuPDF** are available if docx work is ever needed again (see `docs/DECISIONS.md` D-014).

## 🐙 GitHub facts

- Repo `https://github.com/Ozgurisikdamar/Membrane-AI` (private, default branch `main`).
- Project board **#5** — `https://github.com/users/Ozgurisikdamar/projects/5`
  - Project node `PVT_kwHOBGlChs4BaHL_`; Status field `PVTSSF_lAHOBGlChs4BaHL_zhVBDnk`
  - Options: Todo `f75ad846` · In Progress `47fc9ee4` · Done `98236657`
  - `gh project item-edit --id <item> --project-id PVT_kwHOBGlChs4BaHL_ --field-id PVTSSF_lAHOBGlChs4BaHL_zhVBDnk --single-select-option-id <opt>`
- **`docs/ROADMAP.md` is the source of truth**; mirror to the board when `gh` is authed, else skip and note in HANDOVER.

## 🧭 Working style

- The user is a hands-on founder; keep answers concise, concrete, in Turkish; lead with what changed.
- Prefer doing over asking — the docs answer almost everything.
- When you learn a fact future sessions need, write it into the right doc (HANDOVER = state,
  DECISIONS = choices, SKILLS = commands). Never leave it only in chat.
