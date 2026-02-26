// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- totalIssues ---

func TestTotalIssues_Zero(t *testing.T) {
	doc := AnalysisDoc{}
	if got := doc.totalIssues(); got != 0 {
		t.Errorf("totalIssues() = %d, want 0", got)
	}
}

func TestTotalIssues_ConsistencyOnly(t *testing.T) {
	doc := AnalysisDoc{ConsistencyErrors: 3}
	if got := doc.totalIssues(); got != 3 {
		t.Errorf("totalIssues() = %d, want 3", got)
	}
}

func TestTotalIssues_GapsOnly(t *testing.T) {
	doc := AnalysisDoc{
		CodeStatus: &CodeStatusReport{
			Gaps: []string{"gap1", "gap2"},
		},
	}
	if got := doc.totalIssues(); got != 2 {
		t.Errorf("totalIssues() = %d, want 2", got)
	}
}

func TestTotalIssues_Combined(t *testing.T) {
	doc := AnalysisDoc{
		ConsistencyErrors: 5,
		CodeStatus: &CodeStatusReport{
			Gaps: []string{"gap1", "gap2", "gap3"},
		},
	}
	if got := doc.totalIssues(); got != 8 {
		t.Errorf("totalIssues() = %d, want 8", got)
	}
}

// --- collectConsistencyDetails ---

func TestCollectConsistencyDetails_Empty(t *testing.T) {
	r := &AnalyzeResult{}
	details := collectConsistencyDetails(r)
	if len(details) != 0 {
		t.Errorf("got %d details, want 0", len(details))
	}
}

func TestCollectConsistencyDetails_AllFields(t *testing.T) {
	r := &AnalyzeResult{
		OrphanedPRDs:              []string{"prd-orphan"},
		ReleasesWithoutTestSuites: []string{"rel01.0"},
		OrphanedTestSuites:        []string{"test-rel99.0"},
		BrokenTouchpoints:         []string{"uc001->prd-missing"},
		UseCasesNotInRoadmap:      []string{"rel01.0-uc099"},
		SchemaErrors:              []string{"bad-field.yaml"},
		ConstitutionDrift:         []string{"design.yaml"},
		BrokenCitations:           []string{"uc001->prd001:R99"},
	}
	details := collectConsistencyDetails(r)

	if len(details) != 8 {
		t.Fatalf("got %d details, want 8", len(details))
	}

	// Verify prefixes to ensure correct categorization.
	prefixes := []string{
		"orphaned PRD:",
		"release without test suite:",
		"orphaned test suite:",
		"broken touchpoint:",
		"use case not in roadmap:",
		"schema error:",
		"constitution drift:",
		"broken citation:",
	}
	for i, prefix := range prefixes {
		if !strings.HasPrefix(details[i], prefix) {
			t.Errorf("details[%d] = %q, want prefix %q", i, details[i], prefix)
		}
	}
}

func TestCollectConsistencyDetails_MultiplePerField(t *testing.T) {
	r := &AnalyzeResult{
		OrphanedPRDs:   []string{"prd-a", "prd-b"},
		SchemaErrors:   []string{"err1", "err2", "err3"},
		BrokenCitations: []string{"cite1"},
	}
	details := collectConsistencyDetails(r)
	if len(details) != 6 {
		t.Errorf("got %d details, want 6", len(details))
	}
}

// --- writeAnalysisDoc / loadAnalysisDoc ---

func TestWriteAndLoadAnalysisDoc(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analysis.yaml")

	doc := &AnalysisDoc{
		ConsistencyErrors:  2,
		ConsistencyDetails: []string{"orphaned PRD: prd-x", "schema error: bad.yaml"},
		CodeStatus: &CodeStatusReport{
			Releases: []ReleaseCodeStatus{{
				Version:       "01.0",
				Name:          "Core",
				SpecStatus:    "done",
				CodeReadiness: "partial",
			}},
			Gaps: []string{"release 01.0: spec done but code partial"},
		},
	}

	if err := writeAnalysisDoc(doc, path); err != nil {
		t.Fatalf("writeAnalysisDoc: %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Load it back.
	loaded := loadAnalysisDoc(dir)
	if loaded == nil {
		t.Fatal("loadAnalysisDoc returned nil")
	}
	if loaded.ConsistencyErrors != 2 {
		t.Errorf("ConsistencyErrors = %d, want 2", loaded.ConsistencyErrors)
	}
	if len(loaded.ConsistencyDetails) != 2 {
		t.Errorf("ConsistencyDetails len = %d, want 2", len(loaded.ConsistencyDetails))
	}
	if loaded.CodeStatus == nil {
		t.Fatal("CodeStatus is nil")
	}
	if len(loaded.CodeStatus.Gaps) != 1 {
		t.Errorf("Gaps len = %d, want 1", len(loaded.CodeStatus.Gaps))
	}
}

func TestWriteAnalysisDoc_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "dir", "analysis.yaml")

	doc := &AnalysisDoc{ConsistencyErrors: 1}
	if err := writeAnalysisDoc(doc, nested); err != nil {
		t.Fatalf("writeAnalysisDoc: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("file not created in nested directory: %v", err)
	}
}

func TestWriteAnalysisDoc_EmptyDoc(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analysis.yaml")

	doc := &AnalysisDoc{}
	if err := writeAnalysisDoc(doc, path); err != nil {
		t.Fatalf("writeAnalysisDoc: %v", err)
	}

	loaded := loadAnalysisDoc(dir)
	if loaded == nil {
		t.Fatal("loadAnalysisDoc returned nil")
	}
	if loaded.ConsistencyErrors != 0 {
		t.Errorf("ConsistencyErrors = %d, want 0", loaded.ConsistencyErrors)
	}
	if loaded.CodeStatus != nil {
		t.Error("CodeStatus should be nil for empty doc")
	}
}

func TestLoadAnalysisDoc_NoFile(t *testing.T) {
	dir := t.TempDir()
	loaded := loadAnalysisDoc(dir)
	if loaded != nil {
		t.Errorf("expected nil for missing file, got %+v", loaded)
	}
}

func TestLoadAnalysisDoc_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, analysisFileName)
	os.WriteFile(path, []byte("{{invalid yaml"), 0o644)

	loaded := loadAnalysisDoc(dir)
	if loaded != nil {
		t.Errorf("expected nil for invalid YAML, got %+v", loaded)
	}
}
