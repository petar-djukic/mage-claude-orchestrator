//go:build usecase

// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package uc002_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mesh-intelligence/cobbler-scaffold/pkg/orchestrator"
	"github.com/mesh-intelligence/cobbler-scaffold/tests/rel01.0/internal/testutil"
)

var (
	orchRoot    string
	snapshotDir string
)

func TestMain(m *testing.M) {
	var err error
	orchRoot, err = testutil.FindOrchestratorRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: resolving orchRoot: %v\n", err)
		os.Exit(1)
	}
	snapshot, cleanup, prepErr := testutil.PrepareSnapshot(orchRoot)
	if prepErr != nil {
		fmt.Fprintf(os.Stderr, "e2e: preparing snapshot: %v\n", prepErr)
		os.Exit(1)
	}
	snapshotDir = snapshot
	exitCode := m.Run()
	cleanup()
	os.Exit(exitCode)
}

func TestRel01_UC002_StartCreatesGenBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)

	if err := testutil.RunMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	t.Cleanup(func() { testutil.RunMage(t, dir, "reset") }) //nolint:errcheck

	branch := testutil.GitBranch(t, dir)
	if !strings.HasPrefix(branch, "generation-") {
		t.Errorf("expected branch to start with 'generation-', got %q", branch)
	}
}

func TestRel01_UC002_StopMergesAndTags(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)

	if err := testutil.RunMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}

	genBranch := testutil.GitBranch(t, dir)
	if !strings.HasPrefix(genBranch, "generation-") {
		t.Fatalf("expected generation branch after start, got %q", genBranch)
	}

	if err := testutil.RunMage(t, dir, "generator:stop"); err != nil {
		t.Fatalf("generator:stop: %v", err)
	}

	if branch := testutil.GitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after stop, got %q", branch)
	}

	if branches := testutil.GitListBranchesMatching(t, dir, genBranch); len(branches) > 0 {
		t.Errorf("generation branch %q should be deleted after stop, got %v", genBranch, branches)
	}

	for _, suffix := range []string{"-start", "-finished", "-merged"} {
		tag := genBranch + suffix
		if !testutil.GitTagExists(t, dir, tag) {
			t.Errorf("expected tag %q to exist after stop", tag)
		}
	}
}

func TestRel01_UC002_ListShowsMerged(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)

	if err := testutil.RunMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:stop"); err != nil {
		t.Fatalf("generator:stop: %v", err)
	}

	out, err := testutil.RunMageOut(t, dir, "generator:list")
	if err != nil {
		t.Fatalf("generator:list: %v", err)
	}
	if !strings.Contains(out, "merged") {
		t.Errorf("expected 'merged' in generator:list output, got:\n%s", out)
	}
}

func TestRel01_UC002_ResetReturnsToCleanMain(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)

	if err := testutil.RunMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:reset"); err != nil {
		t.Fatalf("mage generator:reset: %v", err)
	}

	if branch := testutil.GitBranch(t, dir); branch != "main" {
		t.Errorf("expected main branch after generator:reset, got %q", branch)
	}
	if branches := testutil.GitListBranchesMatching(t, dir, "generation-"); len(branches) > 0 {
		t.Errorf("expected no generation branches after reset, got %v", branches)
	}
}

func TestRel01_UC002_SwitchSavesAndChangesBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)

	if err := testutil.RunMage(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}

	testutil.WriteConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Generation.Branch = "main"
	})
	if err := testutil.RunMage(t, dir, "generator:switch"); err != nil {
		t.Fatalf("generator:switch: %v", err)
	}
	if branch := testutil.GitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after switch, got %q", branch)
	}
}

// RunOneCycle runs 1 measure + 1 stitch cycle via generator:run with
// Cycles=1 and MaxMeasureIssues=1, then verifies generator:stop merges
// and tags correctly.
func TestRel01_UC002_RunOneCycle(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	testutil.SetupClaude(t, dir)

	testutil.WriteConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 1
		cfg.Generation.Cycles = 1
	})

	if err := testutil.RunMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	genBranch := testutil.GitBranch(t, dir)

	if err := testutil.RunMage(t, dir, "generator:run"); err != nil {
		t.Fatalf("generator:run: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:stop"); err != nil {
		t.Fatalf("generator:stop: %v", err)
	}

	if branch := testutil.GitBranch(t, dir); branch != "main" {
		t.Errorf("expected main after stop, got %q", branch)
	}
	for _, suffix := range []string{"-start", "-finished", "-merged"} {
		tag := genBranch + suffix
		if !testutil.GitTagExists(t, dir, tag) {
			t.Errorf("expected tag %q to exist after stop", tag)
		}
	}
}
