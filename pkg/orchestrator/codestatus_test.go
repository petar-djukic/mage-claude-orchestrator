// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ucPrefixFromID ---

func TestUCPrefixFromID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"rel01.0-uc001-orchestrator-initialization", "rel01.0-uc001"},
		{"rel02.0-uc006-specification-browser", "rel02.0-uc006"},
		{"rel03.0-uc001-cross-generation-comparison", "rel03.0-uc001"},
		{"rel12.3-uc999-long-name", "rel12.3-uc999"},
		{"not-a-use-case", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := ucPrefixFromID(tc.input); got != tc.want {
			t.Errorf("ucPrefixFromID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- testDirForUC ---

func TestTestDirForUC(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"rel01.0-uc001-orchestrator-initialization", filepath.Join("tests", "rel01.0", "uc001")},
		{"rel02.0-uc006-specification-browser", filepath.Join("tests", "rel02.0", "uc006")},
		{"not-a-use-case", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := testDirForUC(tc.input); got != tc.want {
			t.Errorf("testDirForUC(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- countTestFiles ---

func TestCountTestFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "init_test.go"), []byte("package x"), 0o644)
	os.WriteFile(filepath.Join(dir, "bench_test.go"), []byte("package x"), 0o644)
	os.WriteFile(filepath.Join(dir, "helper.go"), []byte("package x"), 0o644)

	if got := countTestFiles(dir); got != 2 {
		t.Errorf("countTestFiles = %d, want 2", got)
	}
}

func TestCountTestFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	if got := countTestFiles(dir); got != 0 {
		t.Errorf("countTestFiles = %d, want 0", got)
	}
}

func TestCountTestFiles_NoDir(t *testing.T) {
	if got := countTestFiles("/nonexistent/path"); got != 0 {
		t.Errorf("countTestFiles = %d, want 0", got)
	}
}

// --- scanTestDirectories ---

func TestScanTestDirectories(t *testing.T) {
	root := t.TempDir()
	// Create tests/rel01.0/uc001/ with a test file.
	uc001 := filepath.Join(root, "rel01.0", "uc001")
	os.MkdirAll(uc001, 0o755)
	os.WriteFile(filepath.Join(uc001, "init_test.go"), []byte("package x"), 0o644)

	// Create tests/rel01.0/uc002/ with two test files.
	uc002 := filepath.Join(root, "rel01.0", "uc002")
	os.MkdirAll(uc002, 0o755)
	os.WriteFile(filepath.Join(uc002, "life_test.go"), []byte("package x"), 0o644)
	os.WriteFile(filepath.Join(uc002, "bench_test.go"), []byte("package x"), 0o644)

	// Create tests/rel02.0/uc001/ with no test files.
	uc201 := filepath.Join(root, "rel02.0", "uc001")
	os.MkdirAll(uc201, 0o755)
	os.WriteFile(filepath.Join(uc201, "helper.go"), []byte("package x"), 0o644)

	got := scanTestDirectories(root)
	if got["rel01.0-uc001"] != 1 {
		t.Errorf("rel01.0-uc001: got %d, want 1", got["rel01.0-uc001"])
	}
	if got["rel01.0-uc002"] != 2 {
		t.Errorf("rel01.0-uc002: got %d, want 2", got["rel01.0-uc002"])
	}
	if got["rel02.0-uc001"] != 0 {
		t.Errorf("rel02.0-uc001: got %d, want 0 (no test files)", got["rel02.0-uc001"])
	}
}

func TestScanTestDirectories_Empty(t *testing.T) {
	root := t.TempDir()
	got := scanTestDirectories(root)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestScanTestDirectories_NoDir(t *testing.T) {
	got := scanTestDirectories("/nonexistent/tests")
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestScanTestDirectories_SkipsNonRelDirs(t *testing.T) {
	root := t.TempDir()
	// Create an "internal" directory that should be skipped.
	internal := filepath.Join(root, "internal", "testutil")
	os.MkdirAll(internal, 0o755)
	os.WriteFile(filepath.Join(internal, "helper_test.go"), []byte("package x"), 0o644)

	got := scanTestDirectories(root)
	if len(got) != 0 {
		t.Errorf("got %v, want empty (internal/ should be skipped)", got)
	}
}

// --- computeCodeStatus ---

func TestComputeCodeStatus_AllImplemented(t *testing.T) {
	roadmap := &RoadmapDoc{
		Releases: []RoadmapRelease{{
			Version: "01.0",
			Name:    "Core",
			Status:  "done",
			UseCases: []RoadmapUseCase{
				{ID: "rel01.0-uc001-init", Status: "done"},
				{ID: "rel01.0-uc002-lifecycle", Status: "done"},
			},
		}},
	}
	scan := map[string]int{
		"rel01.0-uc001": 1,
		"rel01.0-uc002": 3,
	}
	report := computeCodeStatus(roadmap, scan)

	if len(report.Releases) != 1 {
		t.Fatalf("got %d releases, want 1", len(report.Releases))
	}
	if report.Releases[0].CodeReadiness != "all implemented" {
		t.Errorf("CodeReadiness: got %q, want %q", report.Releases[0].CodeReadiness, "all implemented")
	}
	if report.Releases[0].UseCases[0].CodeStatus != "implemented" {
		t.Errorf("UC[0] CodeStatus: got %q, want %q", report.Releases[0].UseCases[0].CodeStatus, "implemented")
	}
	if report.Releases[0].UseCases[0].TestFiles != 1 {
		t.Errorf("UC[0] TestFiles: got %d, want 1", report.Releases[0].UseCases[0].TestFiles)
	}
}

func TestComputeCodeStatus_Partial(t *testing.T) {
	roadmap := &RoadmapDoc{
		Releases: []RoadmapRelease{{
			Version: "01.0",
			Name:    "Core",
			Status:  "done",
			UseCases: []RoadmapUseCase{
				{ID: "rel01.0-uc001-init", Status: "done"},
				{ID: "rel01.0-uc002-lifecycle", Status: "done"},
			},
		}},
	}
	scan := map[string]int{
		"rel01.0-uc001": 1,
		// uc002 missing from scan
	}
	report := computeCodeStatus(roadmap, scan)

	if report.Releases[0].CodeReadiness != "partial" {
		t.Errorf("CodeReadiness: got %q, want %q", report.Releases[0].CodeReadiness, "partial")
	}
	if report.Releases[0].UseCases[1].CodeStatus != "not started" {
		t.Errorf("UC[1] CodeStatus: got %q, want %q", report.Releases[0].UseCases[1].CodeStatus, "not started")
	}
}

func TestComputeCodeStatus_None(t *testing.T) {
	roadmap := &RoadmapDoc{
		Releases: []RoadmapRelease{{
			Version: "02.0",
			Name:    "Extension",
			Status:  "done",
			UseCases: []RoadmapUseCase{
				{ID: "rel02.0-uc001-lifecycle", Status: "done"},
			},
		}},
	}
	scan := map[string]int{}
	report := computeCodeStatus(roadmap, scan)

	if report.Releases[0].CodeReadiness != "none" {
		t.Errorf("CodeReadiness: got %q, want %q", report.Releases[0].CodeReadiness, "none")
	}
}

func TestComputeCodeStatus_SkipsEmptyReleases(t *testing.T) {
	roadmap := &RoadmapDoc{
		Releases: []RoadmapRelease{
			{Version: "01.0", Name: "Core", Status: "done", UseCases: []RoadmapUseCase{
				{ID: "rel01.0-uc001-init", Status: "done"},
			}},
			{Version: "99.0", Name: "Unscheduled", Status: "not started", UseCases: nil},
		},
	}
	scan := map[string]int{"rel01.0-uc001": 1}
	report := computeCodeStatus(roadmap, scan)

	if len(report.Releases) != 1 {
		t.Errorf("got %d releases, want 1 (empty release should be skipped)", len(report.Releases))
	}
}

func TestComputeCodeStatus_MultipleReleases(t *testing.T) {
	roadmap := &RoadmapDoc{
		Releases: []RoadmapRelease{
			{Version: "01.0", Name: "Core", Status: "done", UseCases: []RoadmapUseCase{
				{ID: "rel01.0-uc001-init", Status: "done"},
			}},
			{Version: "02.0", Name: "Ext", Status: "done", UseCases: []RoadmapUseCase{
				{ID: "rel02.0-uc001-lifecycle", Status: "done"},
			}},
		},
	}
	scan := map[string]int{"rel01.0-uc001": 2}
	report := computeCodeStatus(roadmap, scan)

	if len(report.Releases) != 2 {
		t.Fatalf("got %d releases, want 2", len(report.Releases))
	}
	if report.Releases[0].CodeReadiness != "all implemented" {
		t.Errorf("rel01.0 CodeReadiness: got %q, want %q", report.Releases[0].CodeReadiness, "all implemented")
	}
	if report.Releases[1].CodeReadiness != "none" {
		t.Errorf("rel02.0 CodeReadiness: got %q, want %q", report.Releases[1].CodeReadiness, "none")
	}
}

// --- detectSpecCodeGaps ---

func TestDetectSpecCodeGaps_NoGaps(t *testing.T) {
	report := &CodeStatusReport{
		Releases: []ReleaseCodeStatus{{
			Version:       "01.0",
			SpecStatus:    "done",
			CodeReadiness: "all implemented",
			UseCases: []UCCodeStatus{
				{ID: "rel01.0-uc001-init", SpecStatus: "done", CodeStatus: "implemented"},
			},
		}},
	}
	gaps := detectSpecCodeGaps(report)
	if len(gaps) != 0 {
		t.Errorf("got %v, want no gaps", gaps)
	}
}

func TestDetectSpecCodeGaps_ReleaseLevelGap(t *testing.T) {
	report := &CodeStatusReport{
		Releases: []ReleaseCodeStatus{{
			Version:       "01.0",
			SpecStatus:    "done",
			CodeReadiness: "partial",
			UseCases: []UCCodeStatus{
				{ID: "rel01.0-uc001-init", SpecStatus: "done", CodeStatus: "implemented"},
				{ID: "rel01.0-uc002-lifecycle", SpecStatus: "done", CodeStatus: "not started"},
			},
		}},
	}
	gaps := detectSpecCodeGaps(report)
	if len(gaps) != 2 {
		t.Fatalf("got %d gaps, want 2", len(gaps))
	}
}

func TestDetectSpecCodeGaps_UCLevelGap(t *testing.T) {
	report := &CodeStatusReport{
		Releases: []ReleaseCodeStatus{{
			Version:       "01.0",
			SpecStatus:    "not started",
			CodeReadiness: "partial",
			UseCases: []UCCodeStatus{
				{ID: "rel01.0-uc001-init", SpecStatus: "done", CodeStatus: "not started"},
				{ID: "rel01.0-uc002-lifecycle", SpecStatus: "not started", CodeStatus: "not started"},
			},
		}},
	}
	gaps := detectSpecCodeGaps(report)
	// Release spec is "not started" so no release-level gap. But UC001 has a gap.
	if len(gaps) != 1 {
		t.Fatalf("got %d gaps, want 1", len(gaps))
	}
}

func TestDetectSpecCodeGaps_SpecNotStarted_NoGap(t *testing.T) {
	report := &CodeStatusReport{
		Releases: []ReleaseCodeStatus{{
			Version:       "99.0",
			SpecStatus:    "not started",
			CodeReadiness: "none",
			UseCases: []UCCodeStatus{
				{ID: "rel99.0-uc001-future", SpecStatus: "not started", CodeStatus: "not started"},
			},
		}},
	}
	gaps := detectSpecCodeGaps(report)
	if len(gaps) != 0 {
		t.Errorf("got %v, want no gaps", gaps)
	}
}

// --- statusIcon ---

func TestStatusIcon(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"done", "[ok]"},
		{"implemented", "[ok]"},
		{"all implemented", "[ok]"},
		{"partial", "[~~]"},
		{"not started", "[  ]"},
		{"none", "[  ]"},
		{"unknown", "[??]"},
	}
	for _, tc := range cases {
		if got := statusIcon(tc.input); got != tc.want {
			t.Errorf("statusIcon(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
