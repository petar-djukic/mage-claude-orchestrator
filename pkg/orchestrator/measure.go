// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed prompts/measure.tmpl
var defaultMeasurePromptTmpl string

//go:embed constitutions/planning.yaml
var planningConstitution string

// Measure assesses project state and proposes new tasks via Claude.
// Reads all options from Config.
func (o *Orchestrator) Measure() error {
	return o.RunMeasure()
}

// MeasurePrompt prints the measure prompt that would be sent to Claude to stdout.
// This is useful for inspecting or debugging the prompt without invoking Claude.
func (o *Orchestrator) MeasurePrompt() error {
	prompt := o.buildMeasurePrompt("", "", o.cfg.Cobbler.MaxMeasureIssues, "measure-out.yaml")
	fmt.Print(prompt)
	return nil
}

// RunMeasure runs the measure workflow using Config settings.
// It creates a beads tracking issue before invoking Claude and closes it
// afterward with invocation metrics (duration, tokens, LOC).
func (o *Orchestrator) RunMeasure() error {
	measureStart := time.Now()
	logf("measure: starting")
	o.logConfig("measure")

	if err := o.checkClaude(); err != nil {
		return err
	}

	if err := o.requireBeads(); err != nil {
		logf("measure: beads not initialized: %v", err)
		return err
	}

	branch, err := o.resolveBranch(o.cfg.Generation.Branch)
	if err != nil {
		logf("measure: resolveBranch failed: %v", err)
		return err
	}
	logf("measure: resolved branch=%s", branch)

	if err := ensureOnBranch(branch); err != nil {
		logf("measure: ensureOnBranch failed: %v", err)
		return fmt.Errorf("switching to branch: %w", err)
	}

	_ = os.MkdirAll(o.cfg.Cobbler.Dir, 0o755)
	timestamp := time.Now().Format("20060102-150405")
	outputFile := filepath.Join(o.cfg.Cobbler.Dir, fmt.Sprintf("measure-%s.yaml", timestamp))

	// Clean up old measure temp files.
	matches, _ := filepath.Glob(o.cfg.Cobbler.Dir + "measure-*.yaml")
	if len(matches) > 0 {
		logf("measure: cleaning %d old measure temp file(s)", len(matches))
	}
	for _, f := range matches {
		os.Remove(f)
	}

	// Get existing issues and current commit.
	logf("measure: querying existing issues via bd list")
	existingIssues := getExistingIssues()
	issueCount := countJSONArray(existingIssues)
	commitSHA, _ := gitRevParseHEAD()

	logf("measure: found %d existing issue(s), maxMeasureIssues=%d, commit=%s",
		issueCount, o.cfg.Cobbler.MaxMeasureIssues, commitSHA)
	logf("measure: outputFile=%s", outputFile)

	// Create a beads tracking issue for this measure invocation.
	trackingID := o.createMeasureTrackingIssue(branch, commitSHA, issueCount)

	// Snapshot LOC before Claude.
	locBefore := o.captureLOC()
	logf("measure: locBefore prod=%d test=%d", locBefore.Production, locBefore.Test)

	// Build and run prompt.
	prompt := o.buildMeasurePrompt(o.cfg.Cobbler.UserPrompt, existingIssues, o.cfg.Cobbler.MaxMeasureIssues, outputFile)
	logf("measure: prompt built, length=%d bytes", len(prompt))

	logf("measure: invoking Claude")
	claudeStart := time.Now()
	tokens, err := o.runClaude(prompt, "", o.cfg.Silence())
	if err != nil {
		logf("measure: Claude failed after %s: %v", time.Since(claudeStart).Round(time.Second), err)
		o.closeMeasureTrackingIssue(trackingID, claudeStart, measureStart, tokens, locBefore, o.captureLOC(), 0, err)
		return fmt.Errorf("running Claude: %w", err)
	}
	claudeDuration := time.Since(claudeStart)
	logf("measure: Claude completed in %s", claudeDuration.Round(time.Second))

	// Snapshot LOC after Claude (measure doesn't change code, but record for consistency).
	locAfter := o.captureLOC()

	// Import proposed issues.
	if _, statErr := os.Stat(outputFile); statErr != nil {
		logf("measure: output file not found at %s (Claude may not have written it)", outputFile)
		o.closeMeasureTrackingIssue(trackingID, claudeStart, measureStart, tokens, locBefore, locAfter, 0, nil)
		return nil
	}

	fileInfo, _ := os.Stat(outputFile)
	logf("measure: output file found, size=%d bytes", fileInfo.Size())

	logf("measure: importing issues from %s", outputFile)
	importStart := time.Now()
	createdIDs, err := o.importIssues(outputFile)
	if err != nil {
		logf("measure: import failed after %s: %v", time.Since(importStart).Round(time.Second), err)
		o.closeMeasureTrackingIssue(trackingID, claudeStart, measureStart, tokens, locBefore, locAfter, 0, err)
		return fmt.Errorf("importing issues: %w", err)
	}
	logf("measure: imported %d issue(s) in %s", len(createdIDs), time.Since(importStart).Round(time.Second))

	// Record invocation metrics on each created issue.
	rec := InvocationRecord{
		Caller:    "measure",
		StartedAt: claudeStart.UTC().Format(time.RFC3339),
		DurationS: int(claudeDuration.Seconds()),
		Tokens:    claudeTokens{Input: tokens.InputTokens, Output: tokens.OutputTokens},
		LOCBefore: locBefore,
		LOCAfter:  locAfter,
	}
	for _, id := range createdIDs {
		recordInvocation(id, rec)
	}

	if len(createdIDs) == 0 {
		logf("measure: no issues imported, keeping %s for inspection", outputFile)
	} else {
		logf("measure: removing temp file %s (content appended to measure.yaml)", outputFile)
		os.Remove(outputFile)
	}

	// Close tracking issue with final metrics.
	o.closeMeasureTrackingIssue(trackingID, claudeStart, measureStart, tokens, locBefore, locAfter, len(createdIDs), nil)

	logf("measure: completed in %s", time.Since(measureStart).Round(time.Second))
	return nil
}

// createMeasureTrackingIssue creates a beads issue to track the measure
// invocation. The description includes the branch, commit SHA, and number
// of existing issues being analyzed. Returns the issue ID or "" on failure.
func (o *Orchestrator) createMeasureTrackingIssue(branch, commitSHA string, existingIssueCount int) string {
	title := fmt.Sprintf("measure: plan on %s at %s", branch, truncateSHA(commitSHA))
	description := fmt.Sprintf(
		"Measure invocation.\n\nBranch: %s\nCommit: %s\nExisting issues: %d\nMax new issues: %d",
		branch, commitSHA, existingIssueCount, o.cfg.Cobbler.MaxMeasureIssues,
	)

	out, err := bdCreateTask(title, description)
	if err != nil {
		logf("createMeasureTrackingIssue: bd create failed: %v", err)
		return ""
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &created); err != nil || created.ID == "" {
		logf("createMeasureTrackingIssue: parse failed: %v", err)
		return ""
	}

	logf("measure: tracking issue created: %s", created.ID)

	// Claim the issue immediately.
	if err := bdUpdateStatus(created.ID, "in_progress"); err != nil {
		logf("createMeasureTrackingIssue: status update warning: %v", err)
	}
	o.beadsCommit(fmt.Sprintf("Open measure tracking issue %s", created.ID))

	return created.ID
}

// closeMeasureTrackingIssue records invocation metrics and closes the
// tracking issue. If trackingID is empty, this is a no-op.
func (o *Orchestrator) closeMeasureTrackingIssue(trackingID string, claudeStart, measureStart time.Time, tokens ClaudeResult, locBefore, locAfter LocSnapshot, issuesCreated int, claudeErr error) {
	if trackingID == "" {
		return
	}

	rec := InvocationRecord{
		Caller:    "measure",
		StartedAt: claudeStart.UTC().Format(time.RFC3339),
		DurationS: int(time.Since(measureStart).Seconds()),
		Tokens:    claudeTokens{Input: tokens.InputTokens, Output: tokens.OutputTokens},
		LOCBefore: locBefore,
		LOCAfter:  locAfter,
	}
	recordInvocation(trackingID, rec)

	// Add a summary comment.
	status := "success"
	if claudeErr != nil {
		status = fmt.Sprintf("failed: %v", claudeErr)
	}
	summary := fmt.Sprintf("issues_created: %d, status: %s", issuesCreated, status)
	_ = bdCommentAdd(trackingID, summary)

	if err := bdClose(trackingID); err != nil {
		logf("closeMeasureTrackingIssue: bd close warning: %v", err)
	}
	o.beadsCommit(fmt.Sprintf("Close measure tracking issue %s", trackingID))
}

// truncateSHA returns the first 8 characters of a SHA, or the full
// string if shorter.
func truncateSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func getExistingIssues() string {
	if _, err := exec.LookPath(binBd); err != nil {
		logf("getExistingIssues: bd not on PATH: %v", err)
		return "[]"
	}
	out, err := bdListJSON()
	if err != nil {
		logf("getExistingIssues: bd list failed: %v", err)
		return "[]"
	}

	// Extract IDs from the list and fetch full content for each issue.
	var issues []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &issues); err != nil || len(issues) == 0 {
		logf("getExistingIssues: parse or empty: err=%v len=%d", err, len(issues))
		return string(out)
	}

	logf("getExistingIssues: fetching full content for %d issue(s)", len(issues))
	var fullIssues []json.RawMessage
	for _, issue := range issues {
		detail, err := bdShowJSON(issue.ID)
		if err != nil {
			logf("getExistingIssues: bd show %s failed: %v", issue.ID, err)
			continue
		}
		fullIssues = append(fullIssues, json.RawMessage(detail))
	}

	result, err := json.Marshal(fullIssues)
	if err != nil {
		logf("getExistingIssues: marshal failed: %v", err)
		return string(out)
	}
	logf("getExistingIssues: got %d full issue(s), %d bytes", len(fullIssues), len(result))
	return string(result)
}

func countJSONArray(jsonStr string) int {
	var arr []json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &arr); err != nil {
		return 0
	}
	return len(arr)
}

// MeasurePromptData is the template data for the measure prompt.
type MeasurePromptData struct {
	ProjectContext       string
	Limit                int
	OutputPath           string
	UserInput            string
	LinesMin             int
	LinesMax             int
	PlanningConstitution string // kept separate — instructions, not just context
}

func (o *Orchestrator) buildMeasurePrompt(userInput, existingIssues string, limit int, outputPath string) string {
	tmplStr := o.cfg.Cobbler.MeasurePrompt
	if tmplStr == "" {
		tmplStr = defaultMeasurePromptTmpl
	}

	tmpl := template.Must(template.New("measure").Parse(tmplStr))

	planningConst := o.cfg.Cobbler.PlanningConstitution
	if planningConst == "" {
		planningConst = planningConstitution
	}

	projectCtx, err := buildProjectContext(existingIssues)
	if err != nil {
		logf("buildMeasurePrompt: buildProjectContext error: %v", err)
		projectCtx = "# Error building project context\n"
	}

	data := MeasurePromptData{
		ProjectContext:       projectCtx,
		Limit:                limit,
		OutputPath:           outputPath,
		UserInput:            userInput,
		LinesMin:             o.cfg.Cobbler.EstimatedLinesMin,
		LinesMax:             o.cfg.Cobbler.EstimatedLinesMax,
		PlanningConstitution: planningConst,
	}

	logf("buildMeasurePrompt: projectCtx=%d bytes limit=%d userInput=%v",
		len(projectCtx), limit, userInput != "")
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("measure prompt template: %v", err))
	}
	return buf.String()
}

type proposedIssue struct {
	Index       int    `yaml:"index"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Dependency  int    `yaml:"dependency"`
}

func (o *Orchestrator) importIssues(yamlFile string) ([]string, error) {
	logf("importIssues: reading %s", yamlFile)
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("reading YAML file: %w", err)
	}
	logf("importIssues: read %d bytes", len(data))

	var issues []proposedIssue
	if err := yaml.Unmarshal(data, &issues); err != nil {
		logf("importIssues: YAML parse error: %v", err)
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	logf("importIssues: parsed %d proposed issue(s)", len(issues))
	for i, issue := range issues {
		logf("importIssues: [%d] title=%q dep=%d", i, issue.Title, issue.Dependency)
	}

	// Pass 1: create all issues and collect their beads IDs.
	indexToID := make(map[int]string)
	for _, issue := range issues {
		logf("importIssues: creating task %d: %s", issue.Index, issue.Title)
		out, err := bdCreateTask(issue.Title, issue.Description)
		if err != nil {
			logf("importIssues: bd create failed for %q: %v", issue.Title, err)
			continue
		}
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(out, &created); err == nil && created.ID != "" {
			indexToID[issue.Index] = created.ID
			logf("importIssues: created task %d -> beads id=%s", issue.Index, created.ID)
		} else {
			logf("importIssues: bd create returned unparseable output for %q: %s", issue.Title, string(out))
		}
	}

	// Pass 2: wire up dependencies.
	for _, issue := range issues {
		if issue.Dependency < 0 {
			continue
		}
		childID, hasChild := indexToID[issue.Index]
		parentID, hasParent := indexToID[issue.Dependency]
		if !hasChild || !hasParent {
			logf("importIssues: skipping dependency %d->%d (child=%v parent=%v)", issue.Index, issue.Dependency, hasChild, hasParent)
			continue
		}
		logf("importIssues: linking %s (task %d) depends on %s (task %d)", childID, issue.Index, parentID, issue.Dependency)
		if err := bdAddDep(childID, parentID); err != nil {
			logf("importIssues: bd dep add failed: %s -> %s: %v", childID, parentID, err)
		}
	}

	// Collect created IDs in stable order.
	var ids []string
	for _, issue := range issues {
		if id, ok := indexToID[issue.Index]; ok {
			ids = append(ids, id)
		}
	}

	if len(ids) > 0 {
		o.beadsCommit("Add issues from measure")
	}
	logf("importIssues: %d of %d issue(s) imported", len(ids), len(issues))

	// Append new issues to the persistent measure list.
	appendMeasureLog(o.cfg.Cobbler.Dir, issues)

	return ids, nil
}

// appendMeasureLog merges newIssues into the persistent measure.yaml list.
// measure.yaml is a single growing YAML list of all issues proposed across runs.
func appendMeasureLog(cobblerDir string, newIssues []proposedIssue) {
	logPath := filepath.Join(cobblerDir, "measure.yaml")

	var existing []proposedIssue
	if data, err := os.ReadFile(logPath); err == nil {
		if err := yaml.Unmarshal(data, &existing); err != nil {
			logf("appendMeasureLog: could not parse existing list, starting fresh: %v", err)
			existing = nil
		}
	}

	combined := append(existing, newIssues...)
	out, err := yaml.Marshal(combined)
	if err != nil {
		logf("appendMeasureLog: marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(logPath, out, 0o644); err != nil {
		logf("appendMeasureLog: write failed: %v", err)
		return
	}
	logf("appendMeasureLog: %d total issues in %s", len(combined), logPath)
}
