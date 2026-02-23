//go:build usecase

// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package uc007_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestRel01_UC007_Build(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	if err := testutil.RunMage(t, dir, "build"); err != nil {
		t.Fatalf("mage build: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "bin"))
	if err != nil {
		t.Fatalf("reading bin/: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one binary in bin/ after mage build")
	}
}

func TestRel01_UC007_Install(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	if err := testutil.RunMage(t, dir, "install"); err != nil {
		t.Fatalf("mage install: %v", err)
	}
}

func TestRel01_UC007_Clean(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	if err := testutil.RunMage(t, dir, "build"); err != nil {
		t.Fatalf("mage build (setup): %v", err)
	}
	if err := testutil.RunMage(t, dir, "clean"); err != nil {
		t.Fatalf("mage clean: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "bin"))
	if len(entries) > 0 {
		t.Errorf("expected bin/ to be empty after mage clean, found %v", entries)
	}
}

func TestRel01_UC007_Stats(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupRepo(t, snapshotDir)
	out, err := testutil.RunMageOut(t, dir, "stats:loc")
	if err != nil {
		t.Fatalf("mage stats: %v", err)
	}
	if !strings.Contains(out, "go_loc") {
		t.Errorf("expected 'go_loc' in mage stats output, got:\n%s", out)
	}
}
