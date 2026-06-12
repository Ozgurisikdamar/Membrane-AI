// MEMBRANE.AI Code Sweeper VS Code extension: runs the offline `membrane` CLI
// over the workspace and surfaces its findings as editor diagnostics. The
// extension shells out to the same static binary the platform ships (D-027),
// so there is no second detector implementation to drift.

import { execFile } from "child_process";
import * as path from "path";
import * as vscode from "vscode";

interface Finding {
  file: string;
  line: number;
  rule: string;
  severity: string;
  message: string;
}

interface ScanResult {
  root: string;
  findings: Finding[] | null;
}

let diagnostics: vscode.DiagnosticCollection;

export function activate(context: vscode.ExtensionContext): void {
  diagnostics = vscode.languages.createDiagnosticCollection("membrane");
  context.subscriptions.push(diagnostics);

  context.subscriptions.push(
    vscode.commands.registerCommand("membrane.scanWorkspace", () => scanWorkspace())
  );

  if (vscode.workspace.getConfiguration("membrane").get<boolean>("scanOnSave")) {
    context.subscriptions.push(
      vscode.workspace.onDidSaveTextDocument(() => scanWorkspace())
    );
  }

  void scanWorkspace();
}

export function deactivate(): void {
  diagnostics?.dispose();
}

function scanWorkspace(): void {
  const folder = vscode.workspace.workspaceFolders?.[0];
  if (!folder) {
    return;
  }
  const root = folder.uri.fsPath;
  const bin = vscode.workspace.getConfiguration("membrane").get<string>("path") || "membrane";

  // --fail-on=never keeps a non-zero CLI exit (findings present) from being
  // treated as a hard error; we read the JSON regardless.
  execFile(
    bin,
    ["scan", root, "--json", "--fail-on=never"],
    { cwd: root, maxBuffer: 16 * 1024 * 1024 },
    (err, stdout) => {
      if (err && !stdout) {
        void vscode.window.showWarningMessage(`MEMBRANE.AI scan failed: ${err.message}`);
        return;
      }
      let result: ScanResult;
      try {
        result = JSON.parse(stdout) as ScanResult;
      } catch {
        void vscode.window.showWarningMessage("MEMBRANE.AI: could not parse scan output");
        return;
      }
      publish(result);
    }
  );
}

function publish(result: ScanResult): void {
  diagnostics.clear();
  const byFile = new Map<string, vscode.Diagnostic[]>();

  for (const f of result.findings ?? []) {
    const abs = path.isAbsolute(f.file) ? f.file : path.join(result.root, f.file);
    const lineIdx = Math.max(0, f.line - 1);
    const range = new vscode.Range(lineIdx, 0, lineIdx, Number.MAX_SAFE_INTEGER);
    const diag = new vscode.Diagnostic(range, `[${f.rule}] ${f.message}`, severityOf(f.severity));
    diag.source = "MEMBRANE.AI";
    diag.code = f.rule;

    const list = byFile.get(abs) ?? [];
    list.push(diag);
    byFile.set(abs, list);
  }

  for (const [file, diags] of byFile) {
    diagnostics.set(vscode.Uri.file(file), diags);
  }
}

function severityOf(s: string): vscode.DiagnosticSeverity {
  switch (s) {
    case "blocking":
      return vscode.DiagnosticSeverity.Error;
    case "warning":
      return vscode.DiagnosticSeverity.Warning;
    default:
      return vscode.DiagnosticSeverity.Information;
  }
}
