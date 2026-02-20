<!-- Copyright (c) 2026 Petar Djukic. All rights reserved. SPDX-License-Identifier: MIT -->

# Command: Align Specs

Read **docs/constitutions/design.yaml** first and hold it as the authority for the rest of this session.

## 1. Inventory

Find all specification files in the project:

- `docs/VISION.yaml`
- `docs/ARCHITECTURE.yaml`
- `docs/SPECIFICATIONS.yaml` (if present)
- `docs/road-map.yaml` (if present)
- `docs/specs/product-requirements/prd*.yaml`
- `docs/specs/use-cases/rel*.yaml`
- `docs/specs/test-suites/test-rel-*.yaml`
- `docs/engineering/eng*.yaml`

List every file found. Skip any that do not exist.

## 2. Align Each File

For each file, read it, then apply the constitution's format rules for that document type:

**Required fields**: Add any field that the constitution schema lists as required but is absent. Use a short placeholder value — do not invent content. Mark placeholders clearly (e.g. `# TODO: fill in`).

**Field names**: Rename any field whose key does not match the schema exactly.

**Style**: Apply the style rules from the constitution — active voice, present tense, remove forbidden terms, no bold in prose paragraphs, no horizontal rules. Do not rewrite substance; fix only what violates a rule.

**Naming conventions**: Check each file's name against the `naming_conventions` block in the constitution (e.g. `prd[NNN]-[feature-name].yaml`, `rel[NN].[N]-uc[NNN]-[short-name].yaml`). Rename the file if it does not match. Update any cross-references in other files that point to the old name.

**Traceability**: `mage analyze` enforces five invariants. Fix what you can; flag the rest as `# TODO: link missing`. Do not fabricate IDs.

1. Every PRD (`prd[NNN]-[name]`) must be referenced by at least one use case touchpoint. Touchpoints are parsed for bare tokens starting with `prd`, so embed the PRD ID directly in the touchpoint string (e.g. `"Component (prd001-feature R3)"`).
2. Every use case must appear in at least one test suite `traces` list. Trace entries are parsed for tokens matching `rel[NN].[N]-uc[NNN]-[name]`, so use the full use case ID verbatim.
3. Use case touchpoints must not reference PRD IDs that do not exist as files.
4. Every use case must appear in `docs/road-map.yaml` under a release's `use_cases` list.
5. Every release in `docs/road-map.yaml` must have a corresponding `docs/specs/test-suites/test-[release-id].yaml` file.

Write each aligned file back to disk in place.

## 3. Validate

Run `mage analyze` after all files are written.

Fix any errors it reports. Re-run until it exits 0 or until remaining errors require human input (in which case list them clearly).

## 4. Commit

Commit all changes:

```
git add -A
git commit -m "Align specifications to design constitution"
```

Report: how many files were modified, what structural changes were made, and any TODOs that require human input.
