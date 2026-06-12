# MEMBRANE.AI Code Sweeper — VS Code extension

Surfaces the offline `membrane` CLI's findings (leaked secrets, risky patterns)
as native editor diagnostics. It shells out to the same static binary the
platform runs in CI (`pkg/scan`, D-027), so there is no second detector to drift.

## Requirements

The `membrane` CLI on your `PATH` (or set `membrane.path`). Build it with
`task build:cli`, or install a release (see `docs/RELEASING.md`).

## Settings

| Setting | Default | Description |
| --- | --- | --- |
| `membrane.path` | `membrane` | Path to the CLI binary. |
| `membrane.scanOnSave` | `true` | Re-scan the workspace on every save. |

Command: **MEMBRANE.AI: Scan Workspace** (`membrane.scanWorkspace`).

## Develop

```sh
npm install
npm run compile      # tsc → out/extension.js
# F5 in VS Code launches an Extension Development Host
npm run package      # vsce package → .vsix
```
