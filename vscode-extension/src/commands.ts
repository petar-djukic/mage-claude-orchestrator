// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

// prd: prd006-vscode-extension R1, R3
// uc: rel02.0-uc001-lifecycle-commands
// uc: rel02.0-uc003-branch-comparison

import * as vscode from "vscode";
import { execSync } from "child_process";
import {
  ComparisonBrowserProvider,
  listVersionTags,
  computeDiff,
  resolveGenerationRef,
  FileNode,
} from "./comparisonBrowser";

/** Default branch name prefix for generation branches. */
const GENERATION_PREFIX = "generation-";

/** Shared terminal name for mage commands. */
const TERMINAL_NAME = "Mage Orchestrator";

/** Returns the workspace root folder path, or undefined if none is open. */
function workspaceRoot(): string | undefined {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

/**
 * Finds or creates an integrated terminal for mage commands.
 * Reuses an existing terminal with the same name if one is open.
 */
function getTerminal(): vscode.Terminal {
  const existing = vscode.window.terminals.find(
    (t) => t.name === TERMINAL_NAME
  );
  if (existing) {
    return existing;
  }
  return vscode.window.createTerminal({
    name: TERMINAL_NAME,
    cwd: workspaceRoot(),
  });
}

/**
 * Runs a mage target in the integrated terminal.
 * Shows the terminal and sends the command text.
 */
function runMageTarget(target: string): void {
  const terminal = getTerminal();
  terminal.show();
  terminal.sendText(`mage ${target}`);
}

/**
 * Shows a confirmation dialog and returns true if the user confirmed.
 * Used for destructive commands (start, stop, reset).
 */
async function confirmDestructive(action: string): Promise<boolean> {
  const result = await vscode.window.showWarningMessage(
    `Are you sure you want to ${action}? This action modifies generation state.`,
    { modal: true },
    "Yes"
  );
  return result === "Yes";
}

/** Runs mage generator:start after confirmation. */
export async function generatorStart(
  output: vscode.OutputChannel
): Promise<void> {
  if (!(await confirmDestructive("start a new generation"))) {
    return;
  }
  try {
    runMageTarget("generator:start");
  } catch (err) {
    output.appendLine(`generator:start error: ${err}`);
  }
}

/** Runs mage generator:run in the integrated terminal. */
export function generatorRun(output: vscode.OutputChannel): void {
  try {
    runMageTarget("generator:run");
  } catch (err) {
    output.appendLine(`generator:run error: ${err}`);
  }
}

/** Runs mage generator:resume in the integrated terminal. */
export function generatorResume(output: vscode.OutputChannel): void {
  try {
    runMageTarget("generator:resume");
  } catch (err) {
    output.appendLine(`generator:resume error: ${err}`);
  }
}

/** Runs mage generator:stop after confirmation. */
export async function generatorStop(
  output: vscode.OutputChannel
): Promise<void> {
  if (!(await confirmDestructive("stop the current generation"))) {
    return;
  }
  try {
    runMageTarget("generator:stop");
  } catch (err) {
    output.appendLine(`generator:stop error: ${err}`);
  }
}

/** Runs mage generator:reset after confirmation. */
export async function generatorReset(
  output: vscode.OutputChannel
): Promise<void> {
  if (!(await confirmDestructive("reset all generation state"))) {
    return;
  }
  try {
    runMageTarget("generator:reset");
  } catch (err) {
    output.appendLine(`generator:reset error: ${err}`);
  }
}

/**
 * Shows a quick-pick of generation branches and runs
 * mage generator:switch in the terminal. The switch target
 * itself handles the interactive branch selection via the CLI,
 * but we pre-filter to show available branches in VS Code first.
 */
export async function generatorSwitch(
  output: vscode.OutputChannel
): Promise<void> {
  const root = workspaceRoot();
  if (!root) {
    vscode.window.showErrorMessage("No workspace folder open.");
    return;
  }

  let branches: string[];
  try {
    const raw = execSync(
      `git branch --list '${GENERATION_PREFIX}*'`,
      { cwd: root, encoding: "utf-8" }
    );
    branches = raw
      .split("\n")
      .map((line) => line.replace(/^[*+]?\s*/, "").trim())
      .filter((line) => line.length > 0);
  } catch (err) {
    output.appendLine(`generator:switch: failed to list branches: ${err}`);
    vscode.window.showErrorMessage(
      "Failed to list generation branches. Check the output channel for details."
    );
    return;
  }

  if (branches.length === 0) {
    vscode.window.showInformationMessage("No generation branches found.");
    return;
  }

  const selected = await vscode.window.showQuickPick(branches, {
    placeHolder: "Select a generation branch to switch to",
  });
  if (!selected) {
    return;
  }

  // The mage generator:switch target handles the actual checkout.
  // We send the switch command to the terminal.
  try {
    runMageTarget("generator:switch");
  } catch (err) {
    output.appendLine(`generator:switch error: ${err}`);
  }
}

/** Runs mage cobbler:measure in the integrated terminal. */
export function cobblerMeasure(output: vscode.OutputChannel): void {
  try {
    runMageTarget("cobbler:measure");
  } catch (err) {
    output.appendLine(`cobbler:measure error: ${err}`);
  }
}

/** Runs mage cobbler:stitch in the integrated terminal. */
export function cobblerStitch(output: vscode.OutputChannel): void {
  try {
    runMageTarget("cobbler:stitch");
  } catch (err) {
    output.appendLine(`cobbler:stitch error: ${err}`);
  }
}

/**
 * Shows quick-pick for selecting two version tags and displays
 * their file-level diff in the comparison tree view. (prd006 R3.1, R3.2)
 */
export async function compareTags(
  output: vscode.OutputChannel,
  comparisonBrowser: ComparisonBrowserProvider
): Promise<void> {
  const root = workspaceRoot();
  if (!root) {
    vscode.window.showErrorMessage("No workspace folder open.");
    return;
  }

  const tags = listVersionTags(root);
  if (tags.length === 0) {
    vscode.window.showInformationMessage(
      "No version tags found. Tags must match the v[REL].[DATE].[REVISION] pattern."
    );
    return;
  }

  if (tags.length < 2) {
    vscode.window.showInformationMessage(
      "At least two version tags are required for comparison."
    );
    return;
  }

  const first = await vscode.window.showQuickPick(tags, {
    placeHolder: "Select the first tag",
  });
  if (!first) {
    return;
  }

  const remaining = tags.filter((t) => t !== first);
  const second = await vscode.window.showQuickPick(remaining, {
    placeHolder: "Select the second tag",
  });
  if (!second) {
    return;
  }

  const entries = computeDiff(root, first, second);
  comparisonBrowser.setComparison(first, second, entries);
  output.appendLine(`Comparing ${first} .. ${second}: ${entries.length} file(s) changed`);
}

/**
 * Compares two generation branches by resolving each to its best
 * available tag ref. Called from the Generation Browser context menu.
 * (prd006 R3.5, R2.8)
 */
export async function compareGenerations(
  output: vscode.OutputChannel,
  comparisonBrowser: ComparisonBrowserProvider,
  generationA: string,
  generationB: string
): Promise<void> {
  const root = workspaceRoot();
  if (!root) {
    vscode.window.showErrorMessage("No workspace folder open.");
    return;
  }

  const refA = resolveGenerationRef(root, generationA);
  if (!refA) {
    vscode.window.showErrorMessage(
      `Cannot resolve ref for generation: ${generationA}`
    );
    return;
  }

  const refB = resolveGenerationRef(root, generationB);
  if (!refB) {
    vscode.window.showErrorMessage(
      `Cannot resolve ref for generation: ${generationB}`
    );
    return;
  }

  const entries = computeDiff(root, refA, refB);
  comparisonBrowser.setComparison(refA, refB, entries);
  output.appendLine(
    `Comparing generations ${generationA} (${refA}) .. ${generationB} (${refB}): ${entries.length} file(s) changed`
  );
}

/**
 * Opens a VS Code diff editor for a file between two refs.
 * Called when a file node in the comparison tree is clicked.
 * (prd006 R3.4)
 */
export async function openComparisonDiff(node: FileNode): Promise<void> {
  const filePath = node.entry.newPath ?? node.entry.path;
  const leftUri = gitRefUri(node.refA, node.entry.path);
  const rightUri = gitRefUri(node.refB, filePath);
  const title = `${filePath} (${node.refA} ↔ ${node.refB})`;
  await vscode.commands.executeCommand("vscode.diff", leftUri, rightUri, title);
}

/** Encodes a git ref and file path into a mage-git-ref: URI. */
function gitRefUri(ref: string, filePath: string): vscode.Uri {
  return vscode.Uri.parse(
    `mage-git-ref:${filePath}?${encodeURIComponent(JSON.stringify({ ref, path: filePath }))}`
  );
}
