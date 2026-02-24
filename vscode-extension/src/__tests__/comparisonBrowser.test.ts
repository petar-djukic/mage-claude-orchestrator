// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from "vitest";
import {
  parseDiffOutput,
  groupByDirectory,
  statusLabel,
  VERSION_TAG_PATTERN,
  DiffEntry,
  FileStatus,
} from "../comparisonBrowser";

describe("VERSION_TAG_PATTERN", () => {
  it("matches standard version tags", () => {
    expect(VERSION_TAG_PATTERN.test("v0.20260224.1")).toBe(true);
    expect(VERSION_TAG_PATTERN.test("v1.20250101.0")).toBe(true);
    expect(VERSION_TAG_PATTERN.test("v0.20260224.12")).toBe(true);
  });

  it("does not match generation tags", () => {
    expect(VERSION_TAG_PATTERN.test("generation-2025-02-24-start")).toBe(false);
  });

  it("does not match bare v prefix", () => {
    expect(VERSION_TAG_PATTERN.test("v1")).toBe(false);
    expect(VERSION_TAG_PATTERN.test("v")).toBe(false);
  });
});

describe("parseDiffOutput", () => {
  it("parses added, modified, and deleted files", () => {
    const raw = "A\tnew-file.ts\nM\texisting.ts\nD\tremoved.ts\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toEqual([
      { status: "A", path: "new-file.ts" },
      { status: "M", path: "existing.ts" },
      { status: "D", path: "removed.ts" },
    ]);
  });

  it("parses renames with old and new paths", () => {
    const raw = "R100\told-name.ts\tnew-name.ts\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toEqual([
      { status: "R", path: "old-name.ts", newPath: "new-name.ts" },
    ]);
  });

  it("handles copies", () => {
    const raw = "C100\tsource.ts\tcopy.ts\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toEqual([
      { status: "C", path: "source.ts", newPath: "copy.ts" },
    ]);
  });

  it("skips empty lines", () => {
    const raw = "\nM\tfile.ts\n\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toHaveLength(1);
  });

  it("returns empty array for empty input", () => {
    expect(parseDiffOutput("")).toEqual([]);
    expect(parseDiffOutput("\n\n")).toEqual([]);
  });

  it("skips lines with unrecognized status codes", () => {
    const raw = "X\tunknown.ts\nM\tvalid.ts\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toEqual([{ status: "M", path: "valid.ts" }]);
  });

  it("handles type changes", () => {
    const raw = "T\tchanged-type.ts\n";
    const entries = parseDiffOutput(raw);
    expect(entries).toEqual([{ status: "T", path: "changed-type.ts" }]);
  });
});

describe("groupByDirectory", () => {
  it("groups files by parent directory", () => {
    const entries: DiffEntry[] = [
      { status: "A", path: "src/foo.ts" },
      { status: "M", path: "src/bar.ts" },
      { status: "D", path: "tests/baz.ts" },
    ];
    const groups = groupByDirectory(entries);
    expect(groups.size).toBe(2);
    expect(groups.get("src")).toHaveLength(2);
    expect(groups.get("tests")).toHaveLength(1);
  });

  it("uses . for root-level files", () => {
    const entries: DiffEntry[] = [{ status: "M", path: "README.md" }];
    const groups = groupByDirectory(entries);
    expect(groups.get(".")).toHaveLength(1);
  });

  it("uses newPath for renames when grouping", () => {
    const entries: DiffEntry[] = [
      { status: "R", path: "old/file.ts", newPath: "new/file.ts" },
    ];
    const groups = groupByDirectory(entries);
    expect(groups.has("new")).toBe(true);
    expect(groups.has("old")).toBe(false);
  });

  it("handles deeply nested paths", () => {
    const entries: DiffEntry[] = [
      { status: "A", path: "a/b/c/deep.ts" },
    ];
    const groups = groupByDirectory(entries);
    expect(groups.get("a/b/c")).toHaveLength(1);
  });
});

describe("statusLabel", () => {
  const cases: [FileStatus, string][] = [
    ["A", "added"],
    ["M", "modified"],
    ["D", "deleted"],
    ["R", "renamed"],
    ["C", "copied"],
    ["T", "type changed"],
  ];

  for (const [status, label] of cases) {
    it(`returns "${label}" for status "${status}"`, () => {
      expect(statusLabel(status)).toBe(label);
    });
  }
});
