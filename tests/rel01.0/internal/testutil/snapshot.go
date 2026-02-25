//go:build usecase || benchmark

// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mesh-intelligence/cobbler-scaffold/pkg/orchestrator"
)

// ScaffoldModule is the Go module used as the E2E test target.
const ScaffoldModule = "github.com/petar-djukic/sdd-hello-world"

// FindOrchestratorRoot returns the absolute path to the orchestrator
// repository root. It uses `go env GOMOD` to find the module root reliably,
// independent of working directory depth.
func FindOrchestratorRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", fmt.Errorf("go env GOMOD returned empty or /dev/null; not inside a Go module")
	}
	return filepath.Dir(gomod), nil
}

// PrepareSnapshot runs PrepareTestRepo once, copies the working tree (minus
// .git) to a temp directory, and returns that directory plus a cleanup func.
func PrepareSnapshot(orchRoot string) (string, func(), error) {
	cfg, err := orchestrator.LoadConfig(filepath.Join(orchRoot, "configuration.yaml"))
	if err != nil {
		return "", nil, fmt.Errorf("load config: %w", err)
	}
	orch := orchestrator.New(cfg)

	version, err := latestModuleVersion(ScaffoldModule)
	if err != nil {
		return "", nil, fmt.Errorf("resolving latest version of %s: %w", ScaffoldModule, err)
	}
	fmt.Fprintf(os.Stderr, "e2e: using %s@%s\n", ScaffoldModule, version)

	repoDir, err := orch.PrepareTestRepo(ScaffoldModule, version, orchRoot)
	if err != nil {
		return "", nil, fmt.Errorf("PrepareTestRepo: %w", err)
	}
	workDir := filepath.Dir(repoDir)

	snap, err := os.MkdirTemp("", "e2e-snapshot-*")
	if err != nil {
		os.RemoveAll(workDir)
		return "", nil, fmt.Errorf("creating snapshot dir: %w", err)
	}
	if err := CopyDirSkipGit(repoDir, snap); err != nil {
		os.RemoveAll(workDir)
		os.RemoveAll(snap)
		return "", nil, fmt.Errorf("copying snapshot: %w", err)
	}
	os.RemoveAll(workDir)

	cleanup := func() { os.RemoveAll(snap) }
	return snap, cleanup, nil
}

// latestModuleVersion resolves the latest tagged version of a Go module
// using `go list -m -versions`. Returns the last (highest) version.
func latestModuleVersion(module string) (string, error) {
	out, err := exec.Command("go", "list", "-m", "-versions", module).Output()
	if err != nil {
		return "", fmt.Errorf("go list -m -versions %s: %w", module, err)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) < 2 {
		return "", fmt.Errorf("no versions found for %s", module)
	}
	return parts[len(parts)-1], nil
}
