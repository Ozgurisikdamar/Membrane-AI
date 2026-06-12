# SCM integration (GitHub / GitLab)

MEMBRANE.AI gates merges two ways — pick one or run both.

## 1. CLI PR gate (zero backend, works today)

Copy `deploy/github-app/membrane-scan.yml` into a repo's `.github/workflows/`.
It installs the `membrane` CLI, runs `membrane scan . --fail-on=blocking` (the
same detectors as the platform, D-027), and fails the check on any blocking
finding — so branch protection on the `MEMBRANE.AI / code-sweeper` check blocks
the merge. The Technical-Debt Report is uploaded as an artifact. GitLab: run the
same `membrane scan` command in a CI job and key the gate on its exit code
(0 clean / 1 findings / 2 error).

## 2. Platform loop (streaming, the full product)

The backend already implements the outbound gating surface end-to-end (E2E
proven): a submission flows ingestion → orchestrator → reporter, and the reporter
posts a `membrane-ai/governance` **commit status** (approved→success,
rejected→failure, needs_review→neutral) plus a **PR comment** (D-025). The
`shadow`/`enforce` posture (D-035) controls whether a rejection blocks.

Wiring:

1. **Create the GitHub App** from `deploy/github-app/manifest.json` (Settings →
   Developer settings → GitHub Apps → *New from manifest*). Set the webhook URL
   to your ingestion host's `/webhook` and a webhook secret = the ingestion
   `MEMBRANE_INGESTION_WEBHOOK_SECRET` (HMAC-verified, D-026/ingestion).
   Permissions in the manifest: contents:read, statuses:write,
   pull_requests:write, checks:write.
2. **Reporter token**: set `MEMBRANE_REPORTER_GITHUB_TOKEN` to the App
   installation token so the reporter can post statuses/comments.
3. **Branch protection**: require the `membrane-ai/governance` status on the
   protected branch. In `enforce` mode a `rejected` verdict fails it.
4. **Submission source**: a thin push/PR handler extracts the diff (via the
   GitHub compare/PR-files API using the App token) and POSTs the membrane
   webhook payload (`organization_id`, `repository`, `commit_sha`, `pr_number`,
   `diff`, …) to ingestion. The CLI gate (option 1) covers this with no backend;
   a native GitHub-event ingestion adapter is the productionization step.

The two paths compose: the CLI gate gives instant coverage; the platform loop
adds caching, the semantic/premium tiers, RAG, audit, and central policy.
