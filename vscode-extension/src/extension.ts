// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

import * as vscode from "vscode";

export function activate(context: vscode.ExtensionContext): void {
  // TODO: Register tree data providers for status and issues views.
  // TODO: Watch beads directory for changes and refresh views.
  // TODO: Parse .beads/issues.jsonl for issue status.

  const showDashboard = vscode.commands.registerCommand(
    "mageOrchestrator.showDashboard",
    () => {
      vscode.window.showInformationMessage(
        "Mage Orchestrator dashboard — not yet implemented."
      );
    }
  );

  context.subscriptions.push(showDashboard);
}

export function deactivate(): void {
  // Cleanup resources if needed.
}
