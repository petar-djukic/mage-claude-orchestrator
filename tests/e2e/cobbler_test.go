// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package e2e_test

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/mesh-intelligence/cobbler-scaffold/pkg/orchestrator"
)

// TestCobbler_MeasureCreatesIssues verifies that mage cobbler:measure
// produces at least one ready issue in beads.
func TestCobbler_MeasureCreatesIssues(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
	})

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := runMage(t, dir, "cobbler:measure"); err != nil {
		t.Fatalf("cobbler:measure: %v", err)
	}

	n := countReadyIssues(t, dir)
	if n == 0 {
		t.Error("expected at least 1 ready issue after cobbler:measure, got 0")
	}
	t.Logf("cobbler:measure created %d issue(s)", n)
}

// TestCobbler_StitchExecutesTask verifies that cobbler:stitch picks a ready
// issue created by measure and executes it: the task is closed, code is
// committed, and the task branch is cleaned up.
//
//	E2E_CLAUDE=1 go test -v -count=1 -timeout 900s -run TestCobbler_StitchExecutesTask ./tests/e2e/...
func TestCobbler_StitchExecutesTask(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
		cfg.Cobbler.MaxStitchIssuesPerCycle = 1
		cfg.Claude.MaxTimeSec = 600
	})

	// Full reset and start a generation branch.
	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	genBranch := gitBranch(t, dir)
	t.Logf("generation branch: %s", genBranch)

	// Measure: create 1 issue.
	if err := runMage(t, dir, "cobbler:measure"); err != nil {
		t.Fatalf("cobbler:measure: %v", err)
	}
	nBefore := countReadyIssues(t, dir)
	if nBefore == 0 {
		t.Fatal("expected at least 1 ready issue after measure, got 0")
	}
	t.Logf("after measure: %d ready issue(s)", nBefore)

	// Record git HEAD before stitch.
	headBefore := gitHead(t, dir)

	// Stitch: execute the issue.
	if err := runMage(t, dir, "cobbler:stitch"); err != nil {
		t.Fatalf("cobbler:stitch: %v", err)
	}

	// Verify: no ready issues remain.
	nAfter := countReadyIssues(t, dir)
	if nAfter != 0 {
		t.Errorf("expected 0 ready issues after stitch, got %d", nAfter)
	}

	// Verify: git HEAD advanced (stitch merged code).
	headAfter := gitHead(t, dir)
	if headAfter == headBefore {
		t.Error("expected git HEAD to advance after stitch, but it did not")
	}
	t.Logf("HEAD before=%s after=%s", headBefore[:8], headAfter[:8])

	// Verify: no leftover task branches.
	taskBranches := gitListBranchesMatching(t, dir, "task/")
	if len(taskBranches) > 0 {
		t.Errorf("expected no task branches after stitch, got %v", taskBranches)
	}
}

// TestCobbler_BeadsResetClearsAfterMeasure verifies that beads:reset clears
// issues created by measure.
func TestCobbler_BeadsResetClearsAfterMeasure(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
	})

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := runMage(t, dir, "cobbler:measure"); err != nil {
		t.Fatalf("cobbler:measure: %v", err)
	}
	if err := runMage(t, dir, "beads:reset"); err != nil {
		t.Fatalf("beads:reset: %v", err)
	}

	if n := countReadyIssues(t, dir); n != 0 {
		t.Errorf("expected 0 ready issues after beads:reset, got %d", n)
	}
}

// TestGenerator_RunOneCycle verifies that a complete start/run/stop cycle
// with cycles=1 returns to main with the expected tags.
func TestGenerator_RunOneCycle(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
		cfg.Generation.Cycles = 1
	})

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	genBranch := gitBranch(t, dir)

	if err := runMage(t, dir, "generator:run"); err != nil {
		t.Fatalf("generator:run: %v", err)
	}
	if err := runMage(t, dir, "generator:stop"); err != nil {
		t.Fatalf("generator:stop: %v", err)
	}

	if branch := gitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after stop, got %q", branch)
	}
	for _, suffix := range []string{"-start", "-finished", "-merged"} {
		tag := genBranch + suffix
		if !gitTagExists(t, dir, tag) {
			t.Errorf("expected tag %q to exist after stop", tag)
		}
	}
}

// TestGenerator_Resume verifies that generator:resume recovers from an
// interrupted run (switch to main immediately after start, no prior work)
// and completes cleanly.
func TestGenerator_Resume(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
		cfg.Generation.Cycles = 1
		cfg.Claude.MaxTimeSec = 600 // generous single-measure timeout
	})

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}

	// Simulate interruption immediately after start — no work done yet.
	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Generation.Branch = "main"
	})
	if err := runMage(t, dir, "generator:switch"); err != nil {
		t.Fatalf("generator:switch to main: %v", err)
	}
	if branch := gitBranch(t, dir); branch != "main" {
		t.Fatalf("expected main after switch, got %q", branch)
	}

	// Clear generation branch override so resume auto-detects.
	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Generation.Branch = ""
	})
	if err := runMage(t, dir, "generator:resume"); err != nil {
		t.Fatalf("generator:resume: %v", err)
	}

	// Resume runs cycles then stops. If still on a generation branch, stop.
	// The WIP commit from generator:switch wrote branch="main" to the
	// generation branch's config. Clear it and commit the fix so that
	// generator:stop can (a) auto-detect the current branch and (b) checkout
	// main cleanly without an uncommitted tracked-file conflict.
	if branch := gitBranch(t, dir); branch != "main" {
		writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
			cfg.Generation.Branch = ""
		})
		commitCmd := exec.Command("git", "commit", "-am", "Clear generation.branch after resume")
		commitCmd.Dir = dir
		if out, err := commitCmd.CombinedOutput(); err != nil {
			t.Fatalf("committing config fix: %v\n%s", err, out)
		}
		if err := runMage(t, dir, "generator:stop"); err != nil {
			t.Errorf("generator:stop after resume: %v", err)
		}
	}

	if branch := gitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after resume+stop, got %q", branch)
	}
	if branches := gitListBranchesMatching(t, dir, "generation-"); len(branches) > 0 {
		t.Errorf("expected no generation branches after resume+stop, got %v", branches)
	}
}

// TestMeasure_TimingByLimit runs measure with limits 1 through 5 and logs
// the wall-clock time and issue count for each. All five runs share a single
// scaffolded repo (reset+init once) so the only variable is the limit.
//
//	E2E_CLAUDE=1 go test -v -count=1 -timeout 0 -run TestMeasure_TimingByLimit ./tests/e2e/...
func TestMeasure_TimingByLimit(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	for limit := 1; limit <= 5; limit++ {
		t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
			// Reset beads between runs so each starts with zero issues.
			if err := runMage(t, dir, "beads:reset"); err != nil {
				t.Fatalf("beads:reset: %v", err)
			}
			if err := runMage(t, dir, "init"); err != nil {
				t.Fatalf("init: %v", err)
			}

			writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
				cfg.Cobbler.MaxMeasureIssues = limit
			})

			start := time.Now()
			if err := runMage(t, dir, "cobbler:measure"); err != nil {
				t.Fatalf("cobbler:measure (limit=%d): %v", limit, err)
			}
			elapsed := time.Since(start).Round(time.Second)

			n := countReadyIssues(t, dir)
			t.Logf("limit=%d issues=%d time=%s", limit, n, elapsed)
		})
	}
}

// TestGenerator_Stitch100 runs a full generation with 100 stitch iterations.
// Measure creates 5 issues per pass (500-1000 LOC each); stitch processes up
// to 10 per cycle; the generator runs until 100 total tasks are stitched or
// all issues are closed, whichever comes first.
//
// This is a stress test. At ~3-4 min per Claude invocation it takes several
// hours. Run it explicitly:
//
//	E2E_CLAUDE=1 go test -v -count=1 -timeout 0 -run TestGenerator_Stitch100 ./tests/e2e/...
func TestGenerator_Stitch100(t *testing.T) {
	requiresClaude(t)
	dir := setupRepo(t)
	setupClaude(t, dir)

	writeConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 5
		cfg.Cobbler.EstimatedLinesMin = 500
		cfg.Cobbler.EstimatedLinesMax = 1000
		cfg.Cobbler.MaxStitchIssues = 100
		cfg.Cobbler.MaxStitchIssuesPerCycle = 10
		cfg.Claude.MaxTimeSec = 600
	})

	if err := runMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := runMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	genBranch := gitBranch(t, dir)

	if err := runMage(t, dir, "generator:run"); err != nil {
		t.Fatalf("generator:run: %v", err)
	}
	if err := runMage(t, dir, "generator:stop"); err != nil {
		t.Fatalf("generator:stop: %v", err)
	}

	if branch := gitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after stop, got %q", branch)
	}
	for _, suffix := range []string{"-start", "-finished", "-merged"} {
		tag := genBranch + suffix
		if !gitTagExists(t, dir, tag) {
			t.Errorf("expected tag %q to exist after stop", tag)
		}
	}
}
