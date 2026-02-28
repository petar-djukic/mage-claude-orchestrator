// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

// prd: prd006-vscode-extension

import * as fs from "fs";
import * as path from "path";
import * as yaml from "js-yaml";
import * as vscode from "vscode";

// ---- Document type detection ----

/** The four document types this module handles. */
export type DocType = "prd" | "useCase" | "testSuite" | "engineering" | "unknown";

/**
 * Detects the document type from the file path using path pattern matching.
 * Falls back to YAML key inspection when the path does not match any pattern.
 */
export function detectDocType(filePath: string, doc?: unknown): DocType {
  const norm = filePath.replace(/\\/g, "/");

  if (/product-requirements\/prd\d/.test(norm)) return "prd";
  if (/use-cases\/rel\d/.test(norm)) return "useCase";
  if (/test-suites\/test-rel/.test(norm)) return "testSuite";
  if (/\/engineering\/eng\d/.test(norm)) return "engineering";

  // Fallback: key-based detection for non-standard paths.
  if (doc !== null && typeof doc === "object") {
    const keys = Object.keys(doc as Record<string, unknown>);
    if (keys.includes("requirements")) return "prd";
    if (keys.includes("flow") || keys.includes("actor")) return "useCase";
    if (keys.includes("traces") && keys.includes("test_cases")) return "testSuite";
    if (keys.includes("introduction")) return "engineering";
  }

  return "unknown";
}

// ---- Shared HTML helpers ----

/** Escapes HTML special characters to prevent XSS in rendered content. */
export function escapeHtml(text: unknown): string {
  return String(text ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function baseStyle(): string {
  return `
    body {
      font-family: var(--vscode-font-family);
      color: var(--vscode-foreground);
      background: var(--vscode-editor-background);
      padding: 16px;
      font-size: var(--vscode-font-size);
      max-width: 900px;
    }
    h1 { font-size: 1.4em; margin-bottom: 6px; }
    h2 { font-size: 1.1em; margin-top: 24px; margin-bottom: 8px;
         border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 4px; }
    h3 { font-size: 1em; margin-top: 16px; margin-bottom: 6px; }
    p  { margin: 0 0 12px 0; line-height: 1.5; }
    pre {
      font-family: var(--vscode-font-family);
      font-size: var(--vscode-font-size);
      white-space: pre-wrap;
      margin: 0 0 12px 0;
      line-height: 1.5;
    }
    table { border-collapse: collapse; width: 100%; margin-bottom: 16px; }
    th, td { border: 1px solid var(--vscode-panel-border); padding: 6px 10px; text-align: left; }
    th { background: var(--vscode-editor-lineHighlightBackground); font-weight: bold; }
    ul, ol { margin: 0 0 12px 0; padding-left: 20px; line-height: 1.6; }
    li { margin-bottom: 4px; }
    .id-badge { font-size: 0.85em; color: var(--vscode-descriptionForeground); margin-bottom: 20px; }
    .req-group { margin-bottom: 20px; }
  `;
}

function wrapHtml(title: string, body: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${escapeHtml(title)}</title>
  <style>${baseStyle()}</style>
</head>
<body>
${body}
</body>
</html>`;
}

/**
 * Extracts the label and text from a list item that may be either a plain
 * string ("G1: text") or a single-key object ({G1: "text"}) as produced
 * by the YAML parser for inline mappings like `- G1: text`.
 */
export function extractLabelValue(item: unknown): { label: string; value: string } {
  if (typeof item === "string") {
    const m = item.match(/^([\w.]+):\s*(.+)$/s);
    if (m) return { label: m[1], value: m[2].trim() };
    return { label: "", value: item };
  }
  if (item !== null && typeof item === "object") {
    const entries = Object.entries(item as Record<string, unknown>);
    if (entries.length > 0) {
      const [k, v] = entries[0];
      return { label: k, value: typeof v === "string" ? v.trim() : JSON.stringify(v) };
    }
  }
  return { label: "", value: String(item ?? "") };
}

// ---- PRD renderer ----

/** Parsed shape of a PRD YAML document. */
export interface PrdDoc {
  id?: string;
  title?: string;
  problem?: string;
  goals?: unknown[];
  requirements?: Record<string, { title?: string; items?: unknown[] }>;
  non_goals?: unknown[];
  acceptance_criteria?: unknown[];
}

/** Renders a PRD YAML document as a styled HTML string. */
export function renderPrdHtml(fileName: string, doc: PrdDoc): string {
  const title = doc.title ?? fileName;
  const id = doc.id ?? "";

  let body = `  <h1>${escapeHtml(title)}</h1>\n`;
  if (id) body += `  <div class="id-badge">${escapeHtml(id)}</div>\n`;

  if (doc.problem) {
    body += `  <h2>Problem</h2>\n  <pre>${escapeHtml(doc.problem.trimEnd())}</pre>\n`;
  }

  if (doc.goals && doc.goals.length > 0) {
    body += `  <h2>Goals</h2>\n  <table>\n    <tr><th>ID</th><th>Goal</th></tr>\n`;
    for (const g of doc.goals) {
      const { label, value } = extractLabelValue(g);
      body += `    <tr><td>${escapeHtml(label)}</td><td>${escapeHtml(value)}</td></tr>\n`;
    }
    body += `  </table>\n`;
  }

  if (doc.requirements) {
    const entries = Object.entries(doc.requirements);
    if (entries.length > 0) {
      body += `  <h2>Requirements</h2>\n`;
      for (const [key, group] of entries) {
        const groupTitle = group?.title ? `: ${group.title}` : "";
        body += `  <div class="req-group">\n    <h3>${escapeHtml(key)}${escapeHtml(groupTitle)}</h3>\n`;
        if (group?.items && group.items.length > 0) {
          body += `    <ul>\n`;
          for (const item of group.items) {
            const { label, value } = extractLabelValue(item);
            if (label) {
              body += `      <li><strong>${escapeHtml(label)}</strong>: ${escapeHtml(value)}</li>\n`;
            } else {
              body += `      <li>${escapeHtml(value)}</li>\n`;
            }
          }
          body += `    </ul>\n`;
        }
        body += `  </div>\n`;
      }
    }
  }

  if (doc.non_goals && doc.non_goals.length > 0) {
    body += `  <h2>Non-Goals</h2>\n  <ul>\n`;
    for (const ng of doc.non_goals) {
      body += `    <li>${escapeHtml(String(ng))}</li>\n`;
    }
    body += `  </ul>\n`;
  }

  return wrapHtml(title, body);
}

// ---- Use-case renderer ----

/** Parsed shape of a use-case YAML document. */
export interface UseCaseDoc {
  id?: string;
  title?: string;
  summary?: string;
  actor?: string;
  trigger?: string;
  flow?: unknown[];
  touchpoints?: unknown[];
  success_criteria?: unknown[];
  out_of_scope?: unknown[];
}

/** Renders a use-case YAML document as a styled HTML string. */
export function renderUseCaseHtml(fileName: string, doc: UseCaseDoc): string {
  const title = doc.title ?? fileName;
  const id = doc.id ?? "";

  let body = `  <h1>${escapeHtml(title)}</h1>\n`;
  if (id) body += `  <div class="id-badge">${escapeHtml(id)}</div>\n`;

  if (doc.summary) {
    body += `  <h2>Summary</h2>\n  <pre>${escapeHtml(doc.summary.trimEnd())}</pre>\n`;
  }

  if (doc.actor || doc.trigger) {
    body += `  <table>\n`;
    if (doc.actor) body += `    <tr><th>Actor</th><td>${escapeHtml(doc.actor)}</td></tr>\n`;
    if (doc.trigger) body += `    <tr><th>Trigger</th><td>${escapeHtml(doc.trigger)}</td></tr>\n`;
    body += `  </table>\n`;
  }

  if (doc.flow && doc.flow.length > 0) {
    body += `  <h2>Flow</h2>\n  <ol>\n`;
    for (const step of doc.flow) {
      const { value } = extractLabelValue(step);
      body += `    <li>${escapeHtml(value)}</li>\n`;
    }
    body += `  </ol>\n`;
  }

  if (doc.touchpoints && doc.touchpoints.length > 0) {
    body += `  <h2>Touchpoints</h2>\n  <table>\n    <tr><th>ID</th><th>Description</th></tr>\n`;
    for (const tp of doc.touchpoints) {
      const { label, value } = extractLabelValue(tp);
      body += `    <tr><td>${escapeHtml(label)}</td><td>${escapeHtml(value)}</td></tr>\n`;
    }
    body += `  </table>\n`;
  }

  if (doc.success_criteria && doc.success_criteria.length > 0) {
    body += `  <h2>Success Criteria</h2>\n  <ul>\n`;
    for (const s of doc.success_criteria) {
      const { value } = extractLabelValue(s);
      body += `    <li>${escapeHtml(value)}</li>\n`;
    }
    body += `  </ul>\n`;
  }

  if (doc.out_of_scope && doc.out_of_scope.length > 0) {
    body += `  <h2>Out of Scope</h2>\n  <ul>\n`;
    for (const item of doc.out_of_scope) {
      body += `    <li>${escapeHtml(String(item))}</li>\n`;
    }
    body += `  </ul>\n`;
  }

  return wrapHtml(title, body);
}

// ---- Test-suite renderer ----

/** A single test case entry in a test-suite YAML document. */
export interface TestCase {
  use_case?: string;
  name?: string;
  go_test?: string;
  inputs?: Record<string, unknown>;
  expected?: Record<string, unknown>;
}

/** Parsed shape of a test-suite YAML document. */
export interface TestSuiteDoc {
  id?: string;
  title?: string;
  release?: string;
  traces?: string[];
  tags?: string[];
  preconditions?: string[];
  test_cases?: TestCase[];
}

/** Renders a test-suite YAML document as a styled HTML string. */
export function renderTestSuiteHtml(fileName: string, doc: TestSuiteDoc): string {
  const title = doc.title ?? fileName;
  const id = doc.id ?? "";

  let body = `  <h1>${escapeHtml(title)}</h1>\n`;
  if (id) body += `  <div class="id-badge">${escapeHtml(id)}</div>\n`;

  if (doc.release) {
    body += `  <p><strong>Release:</strong> ${escapeHtml(doc.release)}</p>\n`;
  }

  if (doc.traces && doc.traces.length > 0) {
    body += `  <h2>Traced Use Cases</h2>\n  <ul>\n`;
    for (const t of doc.traces) {
      body += `    <li>${escapeHtml(String(t))}</li>\n`;
    }
    body += `  </ul>\n`;
  }

  if (doc.preconditions && doc.preconditions.length > 0) {
    body += `  <h2>Preconditions</h2>\n  <ul>\n`;
    for (const p of doc.preconditions) {
      body += `    <li>${escapeHtml(String(p))}</li>\n`;
    }
    body += `  </ul>\n`;
  }

  if (doc.test_cases && doc.test_cases.length > 0) {
    body += `  <h2>Test Cases</h2>\n`;
    body += `  <table>\n    <tr><th>#</th><th>Name</th><th>Go Test</th><th>Use Case</th></tr>\n`;
    doc.test_cases.forEach((tc, i) => {
      body +=
        `    <tr>` +
        `<td>${i + 1}</td>` +
        `<td>${escapeHtml(tc.name ?? "")}</td>` +
        `<td>${escapeHtml(tc.go_test ?? "")}</td>` +
        `<td>${escapeHtml(tc.use_case ?? "")}</td>` +
        `</tr>\n`;
    });
    body += `  </table>\n`;
  }

  return wrapHtml(title, body);
}

// ---- Engineering guideline renderer ----

/** A named section within an engineering guideline YAML document. */
export interface EngSection {
  title?: string;
  content?: string;
}

/** Parsed shape of an engineering guideline YAML document. */
export interface EngineeringDoc {
  id?: string;
  title?: string;
  introduction?: string;
  sections?: EngSection[];
  [key: string]: unknown;
}

/** Renders an engineering guideline YAML document as a styled HTML string. */
export function renderEngineeringHtml(fileName: string, doc: EngineeringDoc): string {
  const title = doc.title ?? fileName;
  const id = doc.id ?? "";

  let body = `  <h1>${escapeHtml(title)}</h1>\n`;
  if (id) body += `  <div class="id-badge">${escapeHtml(id)}</div>\n`;

  if (doc.introduction) {
    body += `  <h2>Introduction</h2>\n  <pre>${escapeHtml(String(doc.introduction).trimEnd())}</pre>\n`;
  }

  if (doc.sections && doc.sections.length > 0) {
    for (const s of doc.sections) {
      if (s.title) body += `  <h2>${escapeHtml(s.title)}</h2>\n`;
      if (s.content) body += `  <pre>${escapeHtml(String(s.content).trimEnd())}</pre>\n`;
    }
  } else {
    // Render remaining top-level string/object fields as generic sections.
    const skip = new Set(["id", "title", "introduction"]);
    for (const [k, v] of Object.entries(doc)) {
      if (skip.has(k) || v === undefined || v === null) continue;
      body += `  <h2>${escapeHtml(k)}</h2>\n`;
      if (typeof v === "string") {
        body += `  <pre>${escapeHtml(v.trimEnd())}</pre>\n`;
      } else {
        body += `  <pre>${escapeHtml(JSON.stringify(v, null, 2))}</pre>\n`;
      }
    }
  }

  return wrapHtml(title, body);
}

// ---- SpecPreview panel ----

/**
 * Manages a singleton WebviewPanel that renders spec and engineering YAML
 * files as HTML. Supports PRD, use-case, test-suite, and engineering guideline
 * formats. Calling show() with a new URI replaces the panel content in place;
 * a second call while the panel is visible brings it to the foreground.
 */
export class SpecPreview {
  private panel: vscode.WebviewPanel | undefined;

  /**
   * Opens (or refreshes) the preview panel for the given URI. Reads the YAML
   * file, detects the document type by path pattern, and dispatches to the
   * appropriate renderer. Shows an error message when the file is unreadable
   * or the type is not recognized.
   */
  show(uri: vscode.Uri): void {
    const filePath = uri.fsPath;
    const fileName = path.basename(filePath);

    let doc: unknown;
    try {
      const raw = fs.readFileSync(filePath, "utf-8");
      doc = yaml.load(raw);
    } catch (err) {
      vscode.window.showErrorMessage(`Cobbler: failed to read ${fileName}: ${err}`);
      return;
    }

    const docType = detectDocType(filePath, doc);
    let html: string;
    switch (docType) {
      case "prd":
        html = renderPrdHtml(fileName, doc as PrdDoc);
        break;
      case "useCase":
        html = renderUseCaseHtml(fileName, doc as UseCaseDoc);
        break;
      case "testSuite":
        html = renderTestSuiteHtml(fileName, doc as TestSuiteDoc);
        break;
      case "engineering":
        html = renderEngineeringHtml(fileName, doc as EngineeringDoc);
        break;
      default:
        vscode.window.showErrorMessage(
          `Cobbler: ${fileName} is not a recognized spec or engineering document`
        );
        return;
    }

    if (this.panel) {
      this.panel.reveal(vscode.ViewColumn.Beside);
    } else {
      this.panel = vscode.window.createWebviewPanel(
        "mageOrchestrator.specPreview",
        fileName,
        vscode.ViewColumn.Beside,
        { enableScripts: false }
      );
      this.panel.onDidDispose(() => {
        this.panel = undefined;
      });
    }

    this.panel.title = fileName;
    this.panel.webview.html = html;
  }

  /** Disposes the preview panel if it is currently open. */
  dispose(): void {
    this.panel?.dispose();
  }
}
