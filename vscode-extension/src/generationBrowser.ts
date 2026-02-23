// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

// prd: prd006-vscode-extension R2, R6
// prd: prd002-generation-lifecycle R1
// uc: rel02.0-uc002-generation-browser

import * as vscode from "vscode";
import { execSync } from "child_process";

const GENERATION_PREFIX = "generation-";

// ---- Types ----

export type LifecycleState = "started" | "finished" | "merged" | "abandoned";

interface Generation {
  name: string;
  state: LifecycleState;
  tags: string[];
  versionTag: string | undefined;
  isCurrent: boolean;
}

/** Discriminated union for tree node types. */
export type GenerationItem = GenerationNode | TagNode;

interface GenerationNode {
  kind: "generation";
  generation: Generation;
}

interface TagNode {
  kind: "tag";
  label: string;
}

// ---- Provider ----

/**
 * TreeDataProvider for the mageOrchestrator.status view. Discovers
 * generations from git tags, derives lifecycle state from tag suffixes,
 * and highlights the current generation.
 */
export class GenerationBrowserProvider
  implements vscode.TreeDataProvider<GenerationItem>
{
  private _onDidChangeTreeData = new vscode.EventEmitter<
    GenerationItem | undefined | void
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  constructor(private workspaceRoot: string) {}

  refresh(): void {
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: GenerationItem): vscode.TreeItem {
    switch (element.kind) {
      case "generation":
        return this.generationTreeItem(element.generation);
      case "tag":
        return this.tagTreeItem(element.label);
    }
  }

  getChildren(element?: GenerationItem): GenerationItem[] {
    if (!element) {
      return this.discoverGenerations().map(
        (g): GenerationNode => ({ kind: "generation", generation: g })
      );
    }

    if (element.kind === "generation") {
      const children: TagNode[] = element.generation.tags.map(
        (t): TagNode => ({ kind: "tag", label: t })
      );
      if (element.generation.versionTag) {
        children.push({ kind: "tag", label: element.generation.versionTag });
      }
      return children;
    }

    return [];
  }

  // ---- Tree item builders ----

  private generationTreeItem(gen: Generation): vscode.TreeItem {
    const label = gen.versionTag
      ? `${gen.name} (${gen.versionTag})`
      : gen.name;
    const ti = new vscode.TreeItem(
      label,
      vscode.TreeItemCollapsibleState.Collapsed
    );
    ti.description = gen.state;
    ti.contextValue = "generation";
    ti.iconPath = gen.isCurrent
      ? new vscode.ThemeIcon("play")
      : new vscode.ThemeIcon("git-branch");
    ti.tooltip = gen.isCurrent
      ? `${gen.name} (current) — ${gen.state}`
      : `${gen.name} — ${gen.state}`;
    return ti;
  }

  private tagTreeItem(label: string): vscode.TreeItem {
    const ti = new vscode.TreeItem(
      label,
      vscode.TreeItemCollapsibleState.None
    );
    ti.contextValue = "generationTag";
    ti.iconPath = new vscode.ThemeIcon("tag");
    return ti;
  }

  // ---- Git discovery ----

  /**
   * Discovers all generations from git tags. Finds tags matching
   * {GENERATION_PREFIX}*-start and derives lifecycle state from the
   * presence of -finished, -merged, and -abandoned tags.
   */
  private discoverGenerations(): Generation[] {
    const allTags = this.listTags();
    if (allTags.length === 0) {
      return [];
    }

    const currentBranch = this.currentBranch();

    // Find all generation names from -start tags.
    const genNames: string[] = [];
    for (const tag of allTags) {
      if (tag.startsWith(GENERATION_PREFIX) && tag.endsWith("-start")) {
        const name = tag.slice(0, tag.length - "-start".length);
        genNames.push(name);
      }
    }

    const tagSet = new Set(allTags);

    // Resolve version tags: map commit SHA to v* tag.
    const versionTagsByCommit = this.buildVersionTagIndex(allTags);

    const generations: Generation[] = [];
    for (const name of genNames) {
      const tags: string[] = [];
      const suffixes: string[] = ["start", "finished", "merged", "abandoned"];
      for (const suffix of suffixes) {
        const tag = `${name}-${suffix}`;
        if (tagSet.has(tag)) {
          tags.push(tag);
        }
      }

      const state = deriveState(tagSet, name);
      const versionTag = this.resolveVersionTag(
        name,
        tagSet,
        versionTagsByCommit
      );
      const isCurrent = currentBranch === name;

      generations.push({ name, state, tags, versionTag, isCurrent });
    }

    // Sort: current first, then by name descending (newest first).
    generations.sort((a, b) => {
      if (a.isCurrent !== b.isCurrent) {
        return a.isCurrent ? -1 : 1;
      }
      return b.name.localeCompare(a.name);
    });

    return generations;
  }

  /** Lists all git tags. Returns empty array on error. */
  private listTags(): string[] {
    try {
      const raw = execSync("git tag --list", {
        cwd: this.workspaceRoot,
        encoding: "utf-8",
      });
      return raw
        .split("\n")
        .map((t) => t.trim())
        .filter((t) => t.length > 0);
    } catch {
      return [];
    }
  }

  /** Returns the current git branch name, or empty string. */
  private currentBranch(): string {
    try {
      return execSync("git branch --show-current", {
        cwd: this.workspaceRoot,
        encoding: "utf-8",
      }).trim();
    } catch {
      return "";
    }
  }

  /**
   * Builds an index from commit SHA to version tag for all v* tags.
   * Used to associate version tags with generations via their -merged commit.
   */
  private buildVersionTagIndex(allTags: string[]): Map<string, string> {
    const index = new Map<string, string>();
    for (const tag of allTags) {
      if (!tag.startsWith("v")) {
        continue;
      }
      try {
        const sha = execSync(`git rev-parse "${tag}^{}"`, {
          cwd: this.workspaceRoot,
          encoding: "utf-8",
        }).trim();
        if (sha) {
          index.set(sha, tag);
        }
      } catch {
        // Ignore tags that fail to resolve.
      }
    }
    return index;
  }

  /**
   * Resolves a version tag for a generation by checking if the -merged
   * tag commit matches any v* tag commit.
   */
  private resolveVersionTag(
    genName: string,
    tagSet: Set<string>,
    versionIndex: Map<string, string>
  ): string | undefined {
    const mergedTag = `${genName}-merged`;
    if (!tagSet.has(mergedTag)) {
      return undefined;
    }
    try {
      const sha = execSync(`git rev-parse "${mergedTag}^{}"`, {
        cwd: this.workspaceRoot,
        encoding: "utf-8",
      }).trim();
      return versionIndex.get(sha);
    } catch {
      return undefined;
    }
  }
}

/** Derives lifecycle state from which suffixed tags exist. */
export function deriveState(tagSet: Set<string>, genName: string): LifecycleState {
  if (tagSet.has(`${genName}-abandoned`)) {
    return "abandoned";
  }
  if (tagSet.has(`${genName}-merged`)) {
    return "merged";
  }
  if (tagSet.has(`${genName}-finished`)) {
    return "finished";
  }
  return "started";
}
