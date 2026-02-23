// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
)

// BeadsInit initializes the beads database using the current branch as prefix.
// Safe to call when beads is already initialized (no-op).
func (o *Orchestrator) BeadsInit() error {
	logf("beads:init: checking if already initialized")
	if o.beadsInitialized() {
		logf("beads:init: already initialized, skipping")
		return nil
	}
	branch, err := gitCurrentBranch()
	if err != nil {
		logf("beads:init: gitCurrentBranch failed (%v), defaulting to main", err)
		branch = "main"
	}
	logf("beads:init: initializing with prefix=%s", branch)

	if err := os.Remove("AGENTS.md"); err != nil && !os.IsNotExist(err) {
		logf("beads:init: remove AGENTS.md warning: %v", err)
	}
	return o.beadsInitWith(branch)
}

// BeadsReset destroys and reinitializes the beads database.
// Uses the current branch as the new prefix.
func (o *Orchestrator) BeadsReset() error {
	logf("beads:reset: starting")
	branch, err := gitCurrentBranch()
	if err != nil {
		logf("beads:reset: gitCurrentBranch failed (%v), defaulting to main", err)
		branch = "main"
	}
	logf("beads:reset: branch=%s", branch)

	if err := o.beadsResetDB(); err != nil {
		logf("beads:reset: beadsResetDB failed: %v", err)
		return err
	}
	logf("beads:reset: reinitializing with prefix=%s", branch)
	return o.beadsInitWith(branch)
}

// beadsInitialized returns true if the beads directory exists.
func (o *Orchestrator) beadsInitialized() bool {
	_, err := os.Stat(o.cfg.Cobbler.BeadsDir)
	exists := err == nil
	logf("beadsInitialized: %s exists=%v", o.cfg.Cobbler.BeadsDir, exists)
	return exists
}

// requireBeads checks that beads is initialized and returns an error
// with fix instructions if not.
func (o *Orchestrator) requireBeads() error {
	if o.beadsInitialized() {
		return nil
	}
	return fmt.Errorf("beads database not found\n\n  Run 'mage beads:init' to create one, or\n  Run 'mage generator:start' to begin a new generation (which initializes beads)")
}

// beadsInitWith initializes the beads database with the given prefix
// and commits the resulting beads directory.
func (o *Orchestrator) beadsInitWith(prefix string) error {
	logf("beadsInit: prefix=%s", prefix)
	if err := bdInit(prefix); err != nil {
		logf("beadsInit: bdInit failed: %v", err)
		return fmt.Errorf("bd init: %w", err)
	}
	logf("beadsInit: bdInit succeeded, committing")
	o.beadsCommit("Initialize beads database")
	logf("beadsInit: done")
	return nil
}

// beadsResetDB syncs state, stops the daemon, destroys the database,
// and commits empty JSONL files so bd init does not reimport from
// git history.
func (o *Orchestrator) beadsResetDB() error {
	if !o.beadsInitialized() {
		logf("beadsResetDB: no database found, nothing to reset")
		return nil
	}
	logf("beadsResetDB: syncing beads state")
	if err := bdSync(); err != nil {
		logf("beadsResetDB: bdSync warning: %v", err)
	}

	logf("beadsResetDB: running bd admin reset")
	if err := o.bdAdminReset(); err != nil {
		logf("beadsResetDB: bdAdminReset failed: %v", err)
		return err
	}
	logf("beadsResetDB: bd admin reset succeeded")

	// bd init scans git history for issues.jsonl. Commit empty JSONL
	// files so the next init starts with a clean slate.
	logf("beadsResetDB: creating empty JSONL files in %s", o.cfg.Cobbler.BeadsDir)
	if err := os.MkdirAll(o.cfg.Cobbler.BeadsDir, 0o755); err != nil {
		logf("beadsResetDB: MkdirAll failed: %v", err)
		return fmt.Errorf("recreating %s: %w", o.cfg.Cobbler.BeadsDir, err)
	}
	for _, name := range []string{"issues.jsonl", "interactions.jsonl"} {
		if err := os.WriteFile(filepath.Join(o.cfg.Cobbler.BeadsDir, name), nil, 0o644); err != nil {
			logf("beadsResetDB: WriteFile %s warning: %v", name, err)
		}
	}

	logf("beadsResetDB: staging and committing empty JSONL files")
	if err := gitStageDir(o.cfg.Cobbler.BeadsDir); err != nil {
		logf("beadsResetDB: gitStageDir warning: %v", err)
	}
	if err := gitCommit("Reset beads: clear issue history"); err != nil {
		logf("beadsResetDB: gitCommit warning: %v", err)
	}

	// Remove the directory again so bd init creates it fresh.
	logf("beadsResetDB: removing %s so bd init creates it fresh", o.cfg.Cobbler.BeadsDir)
	if err := os.RemoveAll(o.cfg.Cobbler.BeadsDir); err != nil {
		logf("beadsResetDB: RemoveAll warning: %v", err)
	}

	logf("beadsResetDB: done")
	return nil
}
