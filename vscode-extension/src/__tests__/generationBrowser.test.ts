// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from "vitest";
import { deriveState, LifecycleState } from "../generationBrowser";

describe("deriveState", () => {
  const gen = "generation-001";

  it("returns 'started' when only start tag exists", () => {
    const tags = new Set([`${gen}-start`]);
    expect(deriveState(tags, gen)).toBe("started" satisfies LifecycleState);
  });

  it("returns 'finished' when finished tag exists", () => {
    const tags = new Set([`${gen}-start`, `${gen}-finished`]);
    expect(deriveState(tags, gen)).toBe("finished" satisfies LifecycleState);
  });

  it("returns 'merged' when merged tag exists", () => {
    const tags = new Set([
      `${gen}-start`,
      `${gen}-finished`,
      `${gen}-merged`,
    ]);
    expect(deriveState(tags, gen)).toBe("merged" satisfies LifecycleState);
  });

  it("returns 'abandoned' when abandoned tag exists", () => {
    const tags = new Set([`${gen}-start`, `${gen}-abandoned`]);
    expect(deriveState(tags, gen)).toBe(
      "abandoned" satisfies LifecycleState
    );
  });

  it("returns 'abandoned' when both abandoned and finished exist (abandoned takes precedence)", () => {
    const tags = new Set([
      `${gen}-start`,
      `${gen}-finished`,
      `${gen}-abandoned`,
    ]);
    expect(deriveState(tags, gen)).toBe(
      "abandoned" satisfies LifecycleState
    );
  });

  it("returns 'merged' when merged exists without finished", () => {
    const tags = new Set([`${gen}-start`, `${gen}-merged`]);
    expect(deriveState(tags, gen)).toBe("merged" satisfies LifecycleState);
  });
});
