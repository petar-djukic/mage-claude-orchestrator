// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

// prd: prd006-vscode-extension R9
// uc: rel02.0-uc006-specification-browser

import * as vscode from "vscode";
import { SpecGraph } from "./specModel";

/** Matches lines like: // prd: prd006-vscode-extension R8.1 */
export const PRD_PATTERN = /\/\/\s*prd:\s+(prd\d{3}-[\w-]+)/;

/** Matches lines like: // uc: rel02.0-uc006-specification-browser */
export const UC_PATTERN = /\/\/\s*uc:\s+(rel[\w.-]+)/;

/**
 * CodeLensProvider for Go source files. Detects // prd: and // uc:
 * annotation comments and renders a "View Requirement" lens that opens
 * the referenced specification YAML.
 */
export class TraceabilityProvider implements vscode.CodeLensProvider {
  provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
    const lenses: vscode.CodeLens[] = [];

    for (let i = 0; i < document.lineCount; i++) {
      const line = document.lineAt(i).text;

      const prdMatch = line.match(PRD_PATTERN);
      if (prdMatch) {
        const range = new vscode.Range(i, 0, i, line.length);
        lenses.push(
          new vscode.CodeLens(range, {
            title: "View Requirement",
            command: "mageOrchestrator.viewRequirement",
            arguments: [prdMatch[1]],
          })
        );
      }

      const ucMatch = line.match(UC_PATTERN);
      if (ucMatch) {
        const range = new vscode.Range(i, 0, i, line.length);
        lenses.push(
          new vscode.CodeLens(range, {
            title: "View Requirement",
            command: "mageOrchestrator.viewRequirement",
            arguments: [ucMatch[1]],
          })
        );
      }
    }

    return lenses;
  }
}

/**
 * Command handler for mageOrchestrator.viewRequirement. Resolves a
 * prd-id or uc-id to its YAML file path via SpecGraph and opens it.
 * Shows a warning if the id is not found.
 */
export async function viewRequirement(
  graph: SpecGraph,
  id: string
): Promise<void> {
  await graph.ensureBuilt();

  const prd = graph.getPrd(id);
  if (prd) {
    const uri = vscode.Uri.file(prd.filePath);
    await vscode.window.showTextDocument(uri);
    return;
  }

  const uc = graph.getUseCase(id);
  if (uc) {
    const uri = vscode.Uri.file(uc.filePath);
    await vscode.window.showTextDocument(uri);
    return;
  }

  vscode.window.showWarningMessage(`Spec not found: ${id}`);
}
