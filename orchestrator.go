// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/mesh-intelligence/mage-claude-orchestrator/pkg/orchestrator"
)

// Cobbler groups the measure and stitch targets.
type Cobbler mg.Namespace

// Generator groups the code-generation trail lifecycle targets.
type Generator mg.Namespace

// Beads groups issue-tracker lifecycle targets.
type Beads mg.Namespace

// baseCfg holds the configuration loaded from configuration.yaml.
var baseCfg orchestrator.Config

func init() {
	var err error
	baseCfg, err = orchestrator.LoadConfig(orchestrator.DefaultConfigFile)
	if err != nil {
		panic(fmt.Sprintf("loading %s: %v", orchestrator.DefaultConfigFile, err))
	}
}

// newOrch creates an Orchestrator from the base config.
func newOrch() *orchestrator.Orchestrator {
	return orchestrator.New(baseCfg)
}

// logf prints a timestamped log line to stderr.
func logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format(time.RFC3339), msg)
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool { return &v }

// --- Cobbler targets ---

// Measure assesses project state and proposes new tasks via Claude.
func (Cobbler) Measure() error { return newOrch().Measure() }

// Stitch picks ready tasks and invokes Claude to execute them.
func (Cobbler) Stitch() error { return newOrch().Stitch() }

// Reset removes the cobbler scratch directory.
func (Cobbler) Reset() error { return newOrch().CobblerReset() }

// --- Generator targets ---

// Start begins a new generation trail.
func (Generator) Start() error { return newOrch().GeneratorStart() }

// Run executes N cycles of measure + stitch within the current generation.
func (Generator) Run() error { return newOrch().GeneratorRun() }

// Resume recovers from an interrupted run and continues.
func (Generator) Resume() error { return newOrch().GeneratorResume() }

// Stop completes a generation trail and merges it into main.
func (Generator) Stop() error { return newOrch().GeneratorStop() }

// List shows active branches and past generations.
func (Generator) List() error { return newOrch().GeneratorList() }

// Switch commits current work and checks out another generation branch.
func (Generator) Switch() error { return newOrch().GeneratorSwitch() }

// Reset destroys generation branches, worktrees, and Go source directories.
func (Generator) Reset() error { return newOrch().GeneratorReset() }

// --- Beads targets ---

// Init initializes the beads issue tracker.
func (Beads) Init() error { return newOrch().BeadsInit() }

// Reset clears beads issue history.
func (Beads) Reset() error { return newOrch().BeadsReset() }
