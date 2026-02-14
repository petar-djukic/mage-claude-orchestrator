<!-- Copyright (c) 2026 Petar Djukic. All rights reserved. SPDX-License-Identifier: MIT -->

# Command: Test Clone

Test the orchestrator library by deploying it into a cloned Go repository and running the test plan. Failures indicate bugs in the orchestrator code, which get fixed in this repository.

## Arguments

$ARGUMENTS is the Git repository URL or local path of the target Go project. If a second argument is provided, it is the branch or ref to check out before stripping history.

Examples:
- `/test-clone https://github.com/org/project`
- `/test-clone /path/to/local/repo`
- `/test-clone https://github.com/org/project feature-branch`

## Workflow

### 1. Read the test plan

Read `test-plan.yaml` from the orchestrator repo root. Parse preconditions and test cases. Sections 5, 7, 8 require Claude; sections 1-4, 6 do not.

### 2. Create isolated workspace

```bash
ORCH_ROOT="$(pwd)"
WORK_DIR=$(mktemp -d -t test-clone-XXXXXX)
```

### 3. Clone, strip history, init fresh

```bash
git clone <repo-url> "$WORK_DIR/repo"
# If branch specified: cd "$WORK_DIR/repo" && git checkout <branch>
cd "$WORK_DIR/repo"
rm -rf .git
git init
git add -A
git commit -m "Initial commit from test-clone"
```

### 4. Scaffold with mage

One command installs the orchestrator into the target:

```bash
cd "$ORCH_ROOT"
mage test:scaffold "$WORK_DIR/repo"
```

This copies `orchestrator.go`, detects the project structure (module path, main package, source dirs), generates `configuration.yaml`, wires `go.mod` with a replace directive, and verifies `mage -l` in the target.

If scaffold fails, read the error and fix `$ORCH_ROOT/pkg/orchestrator/scaffold.go` before retrying.

### 5. Verify preconditions

- On main branch with clean working tree (done by step 3)
- `bd` CLI available on PATH: `which bd`
- `mage` available: `which mage`
- `configuration.yaml` present (created by scaffold)

### 6. Run test cases (sections 1-4, 6)

Execute each test case from `test-plan.yaml` in order. For each:

1. Run setup commands in `$WORK_DIR/repo`
2. Run the command
3. Check expected exit code
4. Verify expected state (directory existence, branch names, file contents, stdout)
5. Run `mage reset` between tests for clean state

Track results:

| # | Test name | Result | Notes |
|---|-----------|--------|-------|
| 1 | ...       | PASS/FAIL | ... |

Skip sections 5, 7, 8 (require Claude) unless explicitly requested.

### 7. Fix failures

Bugs are in the orchestrator library, not the target project.

1. Read the error output
2. Identify root cause in `$ORCH_ROOT/pkg/orchestrator/`
3. Fix the orchestrator code in `$ORCH_ROOT`
4. The replace directive picks up the fix immediately
5. Re-run the failing test in `$WORK_DIR/repo`
6. Commit the fix: `cd "$ORCH_ROOT" && git add -A && git commit -m "Fix: <description>"`

If a fix breaks other tests, revert. After 3 attempts on a single test, mark it as unfixable and continue.

If the failure is a setup issue (not an orchestrator bug), fix it in `$WORK_DIR/repo` and note it separately.

### 8. Full regression pass

After fixing failures, re-run ALL test cases from sections 1-4 and 6. Repeat until clean.

### 9. Report and clean up

Summarize:

1. Target repository and branch tested
2. Total test cases: run / passed / failed / skipped
3. Fixes applied to the orchestrator
4. Skipped tests (Claude-dependent or unfixable)
5. `mage stats` output from `$ORCH_ROOT`

```bash
rm -rf "$WORK_DIR"
```

## Rules

- Orchestrator fixes go into THIS repo (`$ORCH_ROOT`). Commit them here.
- Target workspace (`$WORK_DIR/repo`) is ephemeral.
- Do not push any changes. Everything is local.
- Do not run Claude-dependent tests (sections 5, 7, 8) unless explicitly requested.
