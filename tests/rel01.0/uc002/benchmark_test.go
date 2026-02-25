//go:build benchmark

// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package uc002_test

import (
	"fmt"
	"os"
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
		fmt.Fprintf(os.Stderr, "benchmark: resolving orchRoot: %v\n", err)
		os.Exit(1)
	}
	snapshot, cleanup, prepErr := testutil.PrepareSnapshot(orchRoot)
	if prepErr != nil {
		fmt.Fprintf(os.Stderr, "benchmark: preparing snapshot: %v\n", prepErr)
		os.Exit(1)
	}
	snapshotDir = snapshot
	exitCode := m.Run()
	cleanup()
	os.Exit(exitCode)
}

// Stitch100 runs generator:run with max_stitch_issues=100 and
// max_stitch_issues_per_cycle=10 to verify the orchestrator handles a large
// number of stitch iterations. This test makes hundreds of Claude API calls
// and can take over an hour. Run with: mage test:benchmark
func TestRel01_UC002_Stitch100(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	testutil.SetupClaude(t, dir)

	testutil.WriteConfigOverride(t, dir, func(cfg *orchestrator.Config) {
		cfg.Cobbler.MaxMeasureIssues = 5
		cfg.Cobbler.MaxStitchIssues = 100
		cfg.Cobbler.MaxStitchIssuesPerCycle = 10
	})

	if err := testutil.RunMage(t, dir, "reset"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:start"); err != nil {
		t.Fatalf("generator:start: %v", err)
	}
	if err := testutil.RunMage(t, dir, "generator:run"); err != nil {
		t.Fatalf("generator:run: %v", err)
	}
}
