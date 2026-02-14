// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectProjectRules(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		wantEmpty bool
		contains []string
	}{
		{
			name:      "no .claude directory",
			setup:     func(t *testing.T, dir string) {},
			wantEmpty: true,
		},
		{
			name: "empty rules directory",
			setup: func(t *testing.T, dir string) {
				mkDir(t, filepath.Join(dir, ".claude", "rules"))
			},
			wantEmpty: true,
		},
		{
			name: "single markdown file",
			setup: func(t *testing.T, dir string) {
				rulesDir := filepath.Join(dir, ".claude", "rules")
				mkDir(t, rulesDir)
				writeFile(t, filepath.Join(rulesDir, "style.md"), "Use tabs for indentation.\n")
			},
			contains: []string{
				"### style.md",
				"Use tabs for indentation.",
			},
		},
		{
			name: "multiple markdown files",
			setup: func(t *testing.T, dir string) {
				rulesDir := filepath.Join(dir, ".claude", "rules")
				mkDir(t, rulesDir)
				writeFile(t, filepath.Join(rulesDir, "a-style.md"), "Rule A content.\n")
				writeFile(t, filepath.Join(rulesDir, "b-testing.md"), "Rule B content.\n")
			},
			contains: []string{
				"### a-style.md",
				"Rule A content.",
				"### b-testing.md",
				"Rule B content.",
			},
		},
		{
			name: "non-markdown files are ignored",
			setup: func(t *testing.T, dir string) {
				rulesDir := filepath.Join(dir, ".claude", "rules")
				mkDir(t, rulesDir)
				writeFile(t, filepath.Join(rulesDir, "style.md"), "Markdown rule.\n")
				writeFile(t, filepath.Join(rulesDir, "notes.txt"), "Should be ignored.\n")
				writeFile(t, filepath.Join(rulesDir, "config.yaml"), "also: ignored\n")
			},
			contains: []string{
				"### style.md",
				"Markdown rule.",
			},
		},
		{
			name: "subdirectories are ignored",
			setup: func(t *testing.T, dir string) {
				rulesDir := filepath.Join(dir, ".claude", "rules")
				mkDir(t, rulesDir)
				writeFile(t, filepath.Join(rulesDir, "style.md"), "Top-level rule.\n")
				subDir := filepath.Join(rulesDir, "subdir")
				mkDir(t, subDir)
				writeFile(t, filepath.Join(subDir, "nested.md"), "Nested rule.\n")
			},
			contains: []string{
				"### style.md",
				"Top-level rule.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)

			got := collectProjectRules(dir)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q\ngot:\n%s", s, got)
				}
			}
		})
	}
}

func TestCollectProjectRulesExcludesNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".claude", "rules")
	mkDir(t, rulesDir)
	writeFile(t, filepath.Join(rulesDir, "notes.txt"), "Should not appear.\n")

	got := collectProjectRules(dir)
	if got != "" {
		t.Errorf("expected empty for non-markdown files, got %q", got)
	}
}

func TestMeasurePromptIncludesProjectRules(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".claude", "rules")
	mkDir(t, rulesDir)
	writeFile(t, filepath.Join(rulesDir, "test-rule.md"), "Always write tests first.\n")

	// Change to temp dir so buildMeasurePrompt's collectProjectRules(".") picks it up.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	o := New(Config{})
	prompt := o.buildMeasurePrompt("", "[]", 5, "/tmp/out.json")

	if !strings.Contains(prompt, "## Project Rules") {
		t.Error("measure prompt missing '## Project Rules' section")
	}
	if !strings.Contains(prompt, "Always write tests first.") {
		t.Error("measure prompt missing rule content")
	}
	if !strings.Contains(prompt, "### test-rule.md") {
		t.Error("measure prompt missing rule filename header")
	}
}

func TestMeasurePromptOmitsRulesWhenNone(t *testing.T) {
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	o := New(Config{})
	prompt := o.buildMeasurePrompt("", "[]", 5, "/tmp/out.json")

	if strings.Contains(prompt, "## Project Rules") {
		t.Error("measure prompt should not contain '## Project Rules' when no rules exist")
	}
}

func TestStitchPromptIncludesProjectRules(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".claude", "rules")
	mkDir(t, rulesDir)
	writeFile(t, filepath.Join(rulesDir, "coding.md"), "Use guard clauses.\n")

	o := New(Config{})
	task := stitchTask{
		id:          "test-001",
		title:       "Test task",
		issueType:   "task",
		description: "A test description.",
		worktreeDir: dir,
	}

	prompt := o.buildStitchPrompt(task)

	if !strings.Contains(prompt, "## Project Rules") {
		t.Error("stitch prompt missing '## Project Rules' section")
	}
	if !strings.Contains(prompt, "Use guard clauses.") {
		t.Error("stitch prompt missing rule content")
	}
}

func TestStitchPromptOmitsRulesWhenNone(t *testing.T) {
	dir := t.TempDir()

	o := New(Config{})
	task := stitchTask{
		id:          "test-002",
		title:       "Test task",
		issueType:   "task",
		description: "A test description.",
		worktreeDir: dir,
	}

	prompt := o.buildStitchPrompt(task)

	if strings.Contains(prompt, "## Project Rules") {
		t.Error("stitch prompt should not contain '## Project Rules' when no rules exist")
	}
}

// mkDir creates a directory and all parents.
func mkDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeFile creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
