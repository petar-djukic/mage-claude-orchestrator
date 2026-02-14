<!-- Copyright (c) 2026 Petar Djukic. All rights reserved. SPDX-License-Identifier: MIT -->

# Command: Test Clone

Test the orchestrator library by deploying it into a cloned Go repository and running the test plan. The orchestrator manages the target project through mage targets. Failures indicate bugs in the orchestrator code, which get fixed in this repository.

## Arguments

$ARGUMENTS is the Git repository URL or local path of the target Go project. If a second argument is provided, it is the branch or ref to check out before stripping history.

Examples:
- `/test-clone https://github.com/org/project`
- `/test-clone /path/to/local/repo`
- `/test-clone https://github.com/org/project feature-branch`

## Context

This repository (`mage-claude-orchestrator`) is a Go library that other projects import into their magefiles. The test plan (`test-plan.yaml` at this repo's root) defines test cases that exercise the orchestrator's mage targets (init, reset, build, generator lifecycle, cobbler workflows, etc.) against a consuming project.

The orchestrator repo root is: the current working directory when the skill is invoked.

## Workflow

### 1. Read the test plan

Read `test-plan.yaml` from the orchestrator repo root. Parse the preconditions and test cases. Identify which tests require Claude (sections 5, 7, 8) and which can run without it (sections 1-4, 6).

### 2. Create isolated workspace

```bash
ORCH_ROOT="$(pwd)"
WORK_DIR=$(mktemp -d -t test-clone-XXXXXX)
echo "Orchestrator root: $ORCH_ROOT"
echo "Test workspace: $WORK_DIR"
```

### 3. Clone the target repository

```bash
git clone <repo-url> "$WORK_DIR/repo"
```

If a branch was specified, check it out before stripping history:
```bash
cd "$WORK_DIR/repo" && git checkout <branch>
```

### 4. Strip git history and initialize fresh repository

The orchestrator's git operations (generation branches, tags, worktrees) need a clean starting state.

```bash
cd "$WORK_DIR/repo"
rm -rf .git
git init
git add -A
git commit -m "Initial commit from test-clone"
```

### 5. Install mage and orchestrator actions

#### 5a. Ensure mage is available

```bash
which mage || go install github.com/magefile/mage@latest
```

#### 5b. Detect the target project

Read the target repo's `go.mod` to extract the module path. Scan for a main package (look for `package main` in `cmd/` or root). Identify Go source directories.

#### 5c. Create magefiles

If the target repo already has `magefiles/` that import the orchestrator, skip to 5d.

Otherwise, create `magefiles/` with targets that wire up the orchestrator. The magefiles must expose every target tested in `test-plan.yaml`.

Create `magefiles/magefile.go`:

```go
//go:build mage

package main

import (
    "fmt"
    "os"
    "os/exec"

    "github.com/mesh-intelligence/mage-claude-orchestrator/pkg/orchestrator"
)

var o *orchestrator.Orchestrator

func init() {
    var err error
    o, err = orchestrator.NewFromFile("configuration.yaml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "config error: %v\n", err)
        os.Exit(1)
    }
}

func Init() error    { return o.Init() }
func Reset() error   { return o.FullReset() }
func Stats() error   { return o.Stats() }

func Build() error {
    cmd := exec.Command("go", "build", "-o", "bin/"+binaryName(), mainPkg())
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func Lint() error {
    cmd := exec.Command("golangci-lint", "run", "./...")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func Install() error {
    cmd := exec.Command("go", "install", mainPkg())
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func Clean() error {
    return os.RemoveAll("bin/")
}

// binaryName and mainPkg read from the orchestrator config.
// Adjust these based on the target project's configuration.yaml.
func binaryName() string { return o.Config().BinaryName }
func mainPkg() string    { return o.Config().MainPackage }
```

Create namespace files for sub-targets (`magefiles/cobbler.go`, `magefiles/beads.go`, `magefiles/generator.go`, `magefiles/test.go`) that expose the orchestrator methods as mage namespace targets (e.g., `type Cobbler mg.Namespace`).

Adapt the magefiles to match the target project's structure. The test-plan.yaml expects these targets at minimum:

| Target | Orchestrator method |
|--------|---------------------|
| init | Orchestrator.Init() |
| reset | Orchestrator.FullReset() |
| stats | Orchestrator.Stats() |
| cobbler:reset | Orchestrator.CobblerReset() |
| cobbler:measure | Orchestrator.RunMeasure() |
| beads:init | Orchestrator.BeadsInit() |
| beads:reset | Orchestrator.BeadsReset() |
| generator:start | Orchestrator.GeneratorStart() |
| generator:stop | Orchestrator.GeneratorStop() |
| generator:run | Orchestrator.GeneratorRun() |
| generator:resume | Orchestrator.GeneratorResume() |
| generator:list | Orchestrator.GeneratorList() |
| generator:reset | Orchestrator.GeneratorReset() |

Build, lint, test, install, and clean targets call standard Go tooling using paths from configuration.yaml.

#### 5d. Create or update configuration.yaml

If the target repo does not have a `configuration.yaml`, create one using `orchestrator.WriteDefaultConfig()` logic, then fill in the detected project-specific fields:

- `module_path` — from the target's go.mod
- `binary_name` — from the main package or directory name
- `main_package` — detected cmd/ path
- `go_source_dirs` — directories containing .go files (e.g., cmd/, pkg/, internal/)
- `binary_dir` — "bin"
- `magefiles_dir` — "magefiles"
- `spec_globs` — point at any docs/ if present
- `seed_files` — empty for test purposes
- `version_file` — point at a file with `const Version = "..."` if one exists

#### 5e. Wire up the orchestrator dependency

Point the magefiles' go.mod at the local orchestrator checkout:

```bash
cd "$WORK_DIR/repo/magefiles"
go mod edit -replace "github.com/mesh-intelligence/mage-claude-orchestrator=$ORCH_ROOT"
go mod tidy
```

If the target repo uses a root go.mod instead of magefiles/go.mod, apply the replace there.

#### 5f. Verify mage discovers targets

```bash
cd "$WORK_DIR/repo"
mage -l
```

All targets from the test plan must appear. If any are missing, fix the magefiles before proceeding.

### 6. Satisfy preconditions

Verify each precondition from test-plan.yaml:

- On main branch with clean working tree — already done by step 4
- bd CLI available on PATH — check `which bd`; if missing, report and stop
- mage available — verified in step 5a
- configuration.yaml present — created in step 5d

### 7. Run test cases (sections 1-4, 6 — no Claude required)

Execute each test case from test-plan.yaml in order. For each test case:

1. Run the setup commands
2. Run the command
3. Check the expected exit code
4. Verify the expected state (directory existence, branch names, file contents, stdout)
5. Run cleanup: `mage reset` between tests to return to a clean state

Track results:

| # | Test name | Result | Notes |
|---|-----------|--------|-------|
| 1 | ...       | PASS/FAIL | ... |

Skip sections 5, 7, and 8 (require Claude) unless the user explicitly requested them.

### 8. Fix failures

When a test fails, the bug is in the orchestrator library code, not the target project.

1. Read the error output
2. Identify the root cause in `$ORCH_ROOT/pkg/orchestrator/`
3. Read the relevant orchestrator source file
4. Fix the orchestrator code **in the orchestrator repo** (`$ORCH_ROOT`)
5. The go.mod replace ensures the target repo picks up the fix immediately
6. Re-run the failing test in `$WORK_DIR/repo` to verify
7. Commit the fix in `$ORCH_ROOT`: `cd "$ORCH_ROOT" && git add -A && git commit -m "Fix: <description>"`

Repeat until the test passes. If the fix breaks other tests, revert and try a different approach.

Keep a running log:

| Fix # | Test | Root cause | Orchestrator file | Change |
|-------|------|------------|-------------------|--------|
| 1     | ...  | ...        | ...               | ...    |

If a failure is caused by misconfiguration in the target workspace (not an orchestrator bug), fix it in `$WORK_DIR/repo` instead and note it as a setup issue, not an orchestrator fix.

### 9. Re-run full suite

After fixing individual failures, re-run ALL test cases from sections 1-4 and 6 to catch regressions. Repeat the fix cycle if new failures appear.

### 10. Report results

Summarize:

1. Target repository and branch tested
2. Total test cases: run / passed / failed / skipped
3. Table of fixes applied to the orchestrator (from step 8)
4. Any tests skipped (Claude-dependent sections, unfixable tests)
5. `mage stats` output from the target workspace

### 11. Clean up

```bash
rm -rf "$WORK_DIR"
```

Report that the workspace has been deleted. Orchestrator fixes remain committed in `$ORCH_ROOT`.

## Error handling

- If the clone fails, report the error and stop
- If mage or bd cannot be installed, report and stop
- If a fix introduces new failures, revert and try a different approach
- If you cannot fix a test after 3 attempts, log it as unfixable, skip it, and continue. Report all unfixable tests in the final summary.
- If you need to modify the target project's magefiles or config (not the orchestrator), do so in `$WORK_DIR/repo` and note it as a setup adjustment.

## Important

- Orchestrator fixes go into THIS repo (`$ORCH_ROOT`). Commit them here.
- Target workspace setup (magefiles, config) lives in `$WORK_DIR/repo`. It is ephemeral.
- The go.mod replace directive ensures the target always uses the local orchestrator source.
- Do not push any changes. Everything is local.
- Do not run Claude-dependent tests (sections 5, 7, 8) unless explicitly requested.
