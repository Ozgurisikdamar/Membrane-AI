# MEMBRANE.AI API

This document summarizes the public service contracts exposed by the current
monorepo. Internal package APIs are intentionally omitted.

## Base Addresses

Local development uses the following default ports:

| Service | Protocol | Default address |
| --- | --- | --- |
| Ingestion | gRPC | `localhost:9001` |
| Ingestion health | HTTP | `http://localhost:8101` |
| Orchestrator health | HTTP | `http://localhost:8102` |
| Analyzer | gRPC | `localhost:9003` |
| Analyzer health | HTTP | `http://localhost:8103` |
| Resolver | gRPC | `localhost:9004` |
| Resolver health | HTTP | `http://localhost:8104` |
| Semantic | HTTP + JSON | `http://localhost:8005` |
| Gateway | HTTP + JSON | `http://localhost:8006` |

Every health server exposes:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/livez` | Liveness check. |
| `GET` | `/readyz` | Readiness check. |

## gRPC Contracts

The source of truth for gRPC messages and services is `proto/membrane/*/v1`.
Regenerate stubs with `task proto`.

### `membrane.ingestion.v1.IngestionService`

Accepts code submissions from IDEs, agents, and webhook bridges, then queues
them for analysis.

| RPC | Request | Response | Notes |
| --- | --- | --- | --- |
| `SubmitDiff` | `SubmitDiffRequest` | `SubmitDiffResponse` | One-shot submission for webhook bridges and simple clients. |
| `StreamCodeDiff` | stream `StreamCodeDiffRequest` | stream `StreamCodeDiffResponse` | Duplex stream for IDE or agent integrations. |

`CodeSubmission` fields:

| Field | Type | Description |
| --- | --- | --- |
| `organization_id` | `string` | Tenant UUID and event partition key. |
| `repository` | `string` | Repository name or slug. |
| `file_path` | `string` | Changed file path. |
| `language` | `string` | Source language. |
| `diff` | `string` | Unified diff under review. |
| `origin` | `Origin` | `ORIGIN_IDE` or `ORIGIN_WEBHOOK`. |
| `commit_sha` | `string` | Optional commit under review. |
| `pr_number` | `int32` | Optional pull request number. |

### `membrane.analyzer.v1.AnalyzerService`

Runs deterministic static analysis and masks secrets before semantic review.

| RPC | Request | Response |
| --- | --- | --- |
| `Analyze` | `AnalyzeRequest` | `AnalyzeResponse` |

`AnalyzeRequest` includes `submission_id`, `organization_id`, `repository`,
`file_path`, `language`, and `diff`.

`AnalyzeResponse` returns:

| Field | Type | Description |
| --- | --- | --- |
| `findings` | repeated `Finding` | Static findings with rule, severity, message, and diff line. |
| `masked_diff` | `string` | Input diff with detected secret values replaced by `[MASKED:<rule>]`. |

### `membrane.resolver.v1.ResolverService`

Retrieves gold-codebase context for the semantic stage.

| RPC | Request | Response |
| --- | --- | --- |
| `ResolveContext` | `ResolveContextRequest` | `ResolveContextResponse` |

`ResolveContextRequest` includes `organization_id`, `language`, `diff`, and
`limit` (default 3, max 10). Responses contain `GoldMatch` records with
`file_path`, `raw_code_content`, `architectural_context`, and
`cosine_distance`.

## Semantic HTTP API

Base URL: `http://localhost:8005`

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/livez` | Liveness check. |
| `GET` | `/readyz` | Readiness check. |
| `POST` | `/v1/semantic/evaluate` | Evaluate a masked diff with optional resolver context. |

### `POST /v1/semantic/evaluate`

Request:

```json
{
  "submission_id": "sub_123",
  "organization_id": "org_123",
  "language": "go",
  "masked_diff": "--- a/main.go\n+++ b/main.go\n@@ ...",
  "gold_context": [
    {
      "file_path": "services/example.go",
      "code": "package example",
      "architectural_context": "Preferred repository pattern"
    }
  ]
}
```

Response:

```json
{
  "findings": [
    {
      "rule": "semantic-risk",
      "severity": "warning",
      "message": "Review the data access boundary."
    }
  ],
  "tier": "local",
  "escalated": false
}
```

Errors:

| Status | Meaning |
| --- | --- |
| `400` | Invalid request payload or domain validation failure. |
| `503` | Premium consensus is enabled but unavailable or misconfigured. |

## Gateway HTTP API

Base URL: `http://localhost:8006`

All gateway endpoints accept JSON and return:

```json
{
  "decision": "allow",
  "reasons": ["policy matched"]
}
```

| Method | Path | Required fields | Description |
| --- | --- | --- | --- |
| `POST` | `/v1/gateway/tool-call` | `tool` | Evaluate an agent or MCP tool call before execution. |
| `POST` | `/v1/gateway/package` | `ecosystem`, `name` | Screen a package dependency request. |
| `POST` | `/v1/gateway/shadow-ai` | `agent` | Record or govern a shadow-AI usage signal. |

### Tool Call Request

```json
{
  "tool": "list_dir",
  "args": {
    "path": "."
  },
  "server": "desktop",
  "agent": "codex"
}
```

### Package Request

```json
{
  "ecosystem": "pypi",
  "name": "fastapi",
  "version": "0.115.0"
}
```

### Shadow AI Request

```json
{
  "agent": "browser-extension",
  "source": "ide",
  "user": "user@example.com"
}
```

Gateway errors:

| Status | Meaning |
| --- | --- |
| `400` | Invalid JSON or missing required fields. |
| `503` | Governance evaluation failed. |

## Versioning

Public HTTP APIs are versioned under `/v1`. gRPC APIs use protobuf packages of
the form `membrane.<service>.v1`; breaking changes are expected to be gated by
`buf breaking`.
