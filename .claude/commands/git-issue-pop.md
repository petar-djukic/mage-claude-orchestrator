<!-- Copyright (c) 2026 Petar Djukic. All rights reserved. SPDX-License-Identifier: MIT -->

Pop a GitHub issue from `petar-djukic/cobbler-scaffold`, decompose it into a local beads epic with sub-issues, and close the GitHub issue when the epic is complete.

## Input

$ARGUMENTS

If arguments contain an issue number (e.g. `42` or `#42`), use that issue. If arguments contain a URL, extract the issue number. If no number is given, list open issues and ask the user to pick one.

## Phase 1 -- Fetch the GitHub Issue

1. Fetch the issue:
   ```
   gh issue view <number> --repo petar-djukic/cobbler-scaffold --json number,title,body,labels,state
   ```
2. If the issue is not open, stop and report its state.
3. Display the issue title, body, and labels to the user.

## Phase 2 -- Gather Project Context

Same as `/make-work`:

1. Read VISION.yaml, ARCHITECTURE.yaml, road-map.yaml, and `docs/constitutions/design.yaml`.
2. Read READMEs for product requirements and use cases relevant to the issue.
3. Run `bd list` to see existing epics and issues.
4. Run `mage analyze` to identify spec issues.
5. Run `mage stats` for current LOC and documentation metrics.
6. Summarize the current project state.

## Phase 3 -- Propose Epic and Issues

Using the GitHub issue as the work item, propose:

1. A beads epic whose title matches the GitHub issue title, prefixed with the GitHub issue number (e.g. `GH-42: Feature title`).
2. Sub-issues that decompose the GitHub issue into actionable work, following the same crumb-format rules as `/make-work`:
   - Type: documentation or code
   - Required Reading: mandatory list of files the agent must read
   - Files to Create/Modify: explicit file list
   - Structure: Requirements, Design Decisions (optional), Acceptance Criteria
   - Code task sizing: 300-700 lines of production code, no more than 5 files
   - No more than 10 sub-issues

3. Present the proposed breakdown to the user for approval. Do not create anything until the user agrees.

## Phase 4 -- Create Local Work Items

After user approval:

1. Create the epic:
   ```
   bd create "GH-<number>: <title>" --type epic --description "<GitHub issue body + link>"
   ```
   Include in the epic description:
   - The full GitHub issue body
   - A reference line: `GitHub: petar-djukic/cobbler-scaffold#<number>`

2. Create sub-issues under the epic:
   ```
   bd create "<sub-issue title>" --parent <epic-id> --type <documentation|code> --description "<structured description>"
   ```

3. Sync and commit:
   ```
   bd sync
   git add -A
   git commit -m "Pop GH-<number>: <title> into local epic"
   ```

## Phase 5 -- Close the Loop

When ALL sub-issues in the epic are closed (check with `bd epic close-eligible`):

1. Close the beads epic:
   ```
   bd epic close-eligible
   ```

2. Close the GitHub issue:
   ```
   gh issue close <number> --repo petar-djukic/cobbler-scaffold --comment "Completed locally. Epic: <epic-id>"
   ```

3. Sync and commit:
   ```
   bd sync
   git add -A
   git commit -m "Close GH-<number>: <title>"
   ```

**Note:** Phase 5 may happen in a later session. When running `/do-work` and completing the last issue in an epic that has a `GH-` prefix in its title, check if the epic is close-eligible and execute Phase 5 automatically.
