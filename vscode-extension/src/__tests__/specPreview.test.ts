// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from "vitest";
import {
  detectDocType,
  extractLabelValue,
  escapeHtml,
  renderPrdHtml,
  renderUseCaseHtml,
  renderTestSuiteHtml,
  renderEngineeringHtml,
  type PrdDoc,
  type UseCaseDoc,
  type TestSuiteDoc,
  type EngineeringDoc,
} from "../specPreview";

// ---- detectDocType ----

describe("detectDocType", () => {
  it("detects prd from product-requirements path", () => {
    expect(
      detectDocType("/workspace/docs/specs/product-requirements/prd001-orchestrator-core.yaml")
    ).toBe("prd");
  });

  it("detects useCase from use-cases path", () => {
    expect(
      detectDocType("/workspace/docs/specs/use-cases/rel01.0-uc001-orchestrator-initialization.yaml")
    ).toBe("useCase");
  });

  it("detects testSuite from test-suites path", () => {
    expect(
      detectDocType("/workspace/docs/specs/test-suites/test-rel01.0.yaml")
    ).toBe("testSuite");
  });

  it("detects engineering from engineering path", () => {
    expect(
      detectDocType("/workspace/docs/engineering/eng02-prompt-templates.yaml")
    ).toBe("engineering");
  });

  it("returns unknown for an unrecognized path with no doc", () => {
    expect(detectDocType("/workspace/some/other/file.yaml")).toBe("unknown");
  });

  it("falls back to key detection for prd when path is non-standard", () => {
    expect(
      detectDocType("/tmp/spec.yaml", { id: "x", requirements: {} })
    ).toBe("prd");
  });

  it("falls back to key detection for useCase when path is non-standard", () => {
    expect(
      detectDocType("/tmp/uc.yaml", { id: "x", actor: "Developer", flow: [] })
    ).toBe("useCase");
  });

  it("falls back to key detection for testSuite when path is non-standard", () => {
    expect(
      detectDocType("/tmp/ts.yaml", { id: "x", traces: [], test_cases: [] })
    ).toBe("testSuite");
  });

  it("falls back to key detection for engineering when path is non-standard", () => {
    expect(
      detectDocType("/tmp/eng.yaml", { id: "x", introduction: "..." })
    ).toBe("engineering");
  });
});

// ---- extractLabelValue ----

describe("extractLabelValue", () => {
  it("extracts label and value from a string in 'KEY: value' form", () => {
    expect(extractLabelValue("G1: Define a config")).toEqual({
      label: "G1",
      value: "Define a config",
    });
  });

  it("extracts dotted label from requirement item string", () => {
    expect(extractLabelValue("R1.2: Must include BinaryName")).toEqual({
      label: "R1.2",
      value: "Must include BinaryName",
    });
  });

  it("returns empty label when string has no colon separator", () => {
    expect(extractLabelValue("plain text")).toEqual({
      label: "",
      value: "plain text",
    });
  });

  it("extracts from single-key object (YAML inline mapping form)", () => {
    expect(extractLabelValue({ G1: "Define a config" })).toEqual({
      label: "G1",
      value: "Define a config",
    });
  });

  it("trims whitespace from value extracted from object", () => {
    expect(extractLabelValue({ F1: "  step text  " })).toEqual({
      label: "F1",
      value: "step text",
    });
  });

  it("converts null to empty strings via null-coalescing", () => {
    const result = extractLabelValue(null);
    expect(result.label).toBe("");
    expect(result.value).toBe("");
  });
});

// ---- escapeHtml ----

describe("escapeHtml", () => {
  it("escapes ampersands", () => {
    expect(escapeHtml("a & b")).toBe("a &amp; b");
  });

  it("escapes angle brackets", () => {
    expect(escapeHtml("<script>")).toBe("&lt;script&gt;");
  });

  it("escapes double quotes", () => {
    expect(escapeHtml('"quoted"')).toBe("&quot;quoted&quot;");
  });

  it("converts non-string input to string", () => {
    expect(escapeHtml(42)).toBe("42");
    expect(escapeHtml(null)).toBe("");
  });
});

// ---- renderPrdHtml ----

describe("renderPrdHtml", () => {
  const minimalPrd: PrdDoc = {
    id: "prd001-test",
    title: "Test PRD",
    problem: "There is a problem.\n",
    goals: [{ G1: "Solve the problem" }, { G2: "Do it well" }],
    requirements: {
      R1: {
        title: "Core Requirements",
        items: [{ "R1.1": "Must compile" }, { "R1.2": "Must test" }],
      },
    },
  };

  it("produces a DOCTYPE html document", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("<!DOCTYPE html>");
    expect(html).toContain("<html");
  });

  it("includes the title in an h1 tag", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("<h1>Test PRD</h1>");
  });

  it("falls back to file name when title is absent", () => {
    const html = renderPrdHtml("prd001.yaml", {});
    expect(html).toContain("<h1>prd001.yaml</h1>");
  });

  it("renders the id badge", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("prd001-test");
  });

  it("renders the problem section", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("<h2>Problem</h2>");
    expect(html).toContain("There is a problem.");
  });

  it("renders goals as a table with IDs and text", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("<h2>Goals</h2>");
    expect(html).toContain("G1");
    expect(html).toContain("Solve the problem");
    expect(html).toContain("G2");
  });

  it("renders requirements with group headings and items", () => {
    const html = renderPrdHtml("prd001.yaml", minimalPrd);
    expect(html).toContain("<h2>Requirements</h2>");
    expect(html).toContain("R1");
    expect(html).toContain("Core Requirements");
    expect(html).toContain("R1.1");
    expect(html).toContain("Must compile");
  });

  it("omits goals section when goals is absent", () => {
    const html = renderPrdHtml("f.yaml", { title: "T" });
    expect(html).not.toContain("<h2>Goals</h2>");
  });

  it("omits requirements section when requirements is absent", () => {
    const html = renderPrdHtml("f.yaml", { title: "T" });
    expect(html).not.toContain("<h2>Requirements</h2>");
  });

  it("uses VS Code CSS variables", () => {
    const html = renderPrdHtml("f.yaml", minimalPrd);
    expect(html).toContain("var(--vscode-font-family)");
    expect(html).toContain("var(--vscode-foreground)");
    expect(html).toContain("var(--vscode-editor-background)");
  });

  it("escapes HTML special characters in content", () => {
    const html = renderPrdHtml("f.yaml", {
      title: "<Dangerous>",
      problem: "A & B\n",
    });
    expect(html).not.toContain("<Dangerous>");
    expect(html).toContain("&lt;Dangerous&gt;");
    expect(html).toContain("A &amp; B");
  });
});

// ---- renderUseCaseHtml ----

describe("renderUseCaseHtml", () => {
  const minimalUc: UseCaseDoc = {
    id: "rel01.0-uc001",
    title: "Orchestrator Init",
    summary: "A project sets up the orchestrator.\n",
    actor: "Developer",
    trigger: "First-time setup",
    flow: [{ F1: "Create Config" }, { F2: "Call New()" }],
    touchpoints: [{ T1: "Config struct per prd001 R1" }],
    success_criteria: [{ S1: "Non-nil Orchestrator returned" }],
  };

  it("produces a DOCTYPE html document", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<!DOCTYPE html>");
  });

  it("includes the title in an h1 tag", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<h1>Orchestrator Init</h1>");
  });

  it("renders summary", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<h2>Summary</h2>");
    expect(html).toContain("A project sets up the orchestrator.");
  });

  it("renders actor and trigger in a table", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("Developer");
    expect(html).toContain("First-time setup");
  });

  it("renders flow as an ordered list", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<h2>Flow</h2>");
    expect(html).toContain("<ol>");
    expect(html).toContain("Create Config");
    expect(html).toContain("Call New()");
  });

  it("renders touchpoints as a table", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<h2>Touchpoints</h2>");
    expect(html).toContain("T1");
    expect(html).toContain("Config struct per prd001 R1");
  });

  it("renders success criteria as a list", () => {
    const html = renderUseCaseHtml("uc001.yaml", minimalUc);
    expect(html).toContain("<h2>Success Criteria</h2>");
    expect(html).toContain("Non-nil Orchestrator returned");
  });

  it("omits flow section when flow is absent", () => {
    const html = renderUseCaseHtml("f.yaml", { title: "T" });
    expect(html).not.toContain("<h2>Flow</h2>");
  });

  it("omits touchpoints section when touchpoints is absent", () => {
    const html = renderUseCaseHtml("f.yaml", { title: "T" });
    expect(html).not.toContain("<h2>Touchpoints</h2>");
  });
});

// ---- renderTestSuiteHtml ----

describe("renderTestSuiteHtml", () => {
  const minimalTs: TestSuiteDoc = {
    id: "test-rel01.0",
    title: "Release 01.0 Test Suite",
    release: "rel01.0",
    traces: ["rel01.0-uc001", "rel01.0-uc002"],
    preconditions: ["Clean git repo"],
    test_cases: [
      {
        use_case: "rel01.0-uc001",
        name: "New applies defaults",
        go_test: "TestRel01_UC001_NewAppliesDefaults",
      },
    ],
  };

  it("produces a DOCTYPE html document", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("<!DOCTYPE html>");
  });

  it("includes the title in an h1 tag", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("<h1>Release 01.0 Test Suite</h1>");
  });

  it("renders the release identifier", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("rel01.0");
  });

  it("renders traced use cases", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("<h2>Traced Use Cases</h2>");
    expect(html).toContain("rel01.0-uc001");
    expect(html).toContain("rel01.0-uc002");
  });

  it("renders preconditions", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("<h2>Preconditions</h2>");
    expect(html).toContain("Clean git repo");
  });

  it("renders test cases in a table", () => {
    const html = renderTestSuiteHtml("ts.yaml", minimalTs);
    expect(html).toContain("<h2>Test Cases</h2>");
    expect(html).toContain("New applies defaults");
    expect(html).toContain("TestRel01_UC001_NewAppliesDefaults");
  });

  it("handles empty test_cases list", () => {
    const html = renderTestSuiteHtml("ts.yaml", { ...minimalTs, test_cases: [] });
    expect(html).not.toContain("<h2>Test Cases</h2>");
  });

  it("falls back to file name when title is absent", () => {
    const html = renderTestSuiteHtml("ts.yaml", {});
    expect(html).toContain("<h1>ts.yaml</h1>");
  });
});

// ---- renderEngineeringHtml ----

describe("renderEngineeringHtml", () => {
  const withSections: EngineeringDoc = {
    id: "eng02-prompt-templates",
    title: "Prompt Template Conventions",
    introduction: "We use Go text/template.\n",
    sections: [
      { title: "Embedded Defaults", content: "Two default templates exist.\n" },
      { title: "Custom Overrides", content: "Consuming projects can override.\n" },
    ],
  };

  it("produces a DOCTYPE html document", () => {
    const html = renderEngineeringHtml("eng02.yaml", withSections);
    expect(html).toContain("<!DOCTYPE html>");
  });

  it("includes the title in an h1 tag", () => {
    const html = renderEngineeringHtml("eng02.yaml", withSections);
    expect(html).toContain("<h1>Prompt Template Conventions</h1>");
  });

  it("renders introduction", () => {
    const html = renderEngineeringHtml("eng02.yaml", withSections);
    expect(html).toContain("<h2>Introduction</h2>");
    expect(html).toContain("We use Go text/template.");
  });

  it("renders named sections from the sections field", () => {
    const html = renderEngineeringHtml("eng02.yaml", withSections);
    expect(html).toContain("<h2>Embedded Defaults</h2>");
    expect(html).toContain("Two default templates exist.");
    expect(html).toContain("<h2>Custom Overrides</h2>");
  });

  it("falls back to top-level string fields when sections is absent", () => {
    const doc: EngineeringDoc = {
      id: "eng99",
      title: "Minimal",
      introduction: "Intro.\n",
      extra_notes: "Some extra notes.\n",
    };
    const html = renderEngineeringHtml("eng99.yaml", doc);
    expect(html).toContain("<h2>extra_notes</h2>");
    expect(html).toContain("Some extra notes.");
  });

  it("falls back to file name when title is absent", () => {
    const html = renderEngineeringHtml("eng.yaml", {});
    expect(html).toContain("<h1>eng.yaml</h1>");
  });

  it("omits introduction section when introduction is absent", () => {
    const html = renderEngineeringHtml("eng.yaml", { title: "T", sections: [] });
    expect(html).not.toContain("<h2>Introduction</h2>");
  });
});
