// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVsixFilename_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `{"name": "mage-orchestrator", "version": "0.0.1"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := vsixFilename(dir)
	if err != nil {
		t.Fatalf("vsixFilename: unexpected error: %v", err)
	}
	want := "mage-orchestrator-0.0.1.vsix"
	if got != want {
		t.Errorf("vsixFilename = %q, want %q", got, want)
	}
}

func TestVsixFilename_MissingPackageJSON(t *testing.T) {
	dir := t.TempDir()
	_, err := vsixFilename(dir)
	if err == nil {
		t.Fatal("vsixFilename: expected error for missing package.json, got nil")
	}
}

func TestVsixFilename_EmptyFields(t *testing.T) {
	dir := t.TempDir()
	content := `{"name": "", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := vsixFilename(dir)
	if err == nil {
		t.Fatal("vsixFilename: expected error for empty name, got nil")
	}
}

func TestVsixFilename_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := vsixFilename(dir)
	if err == nil {
		t.Fatal("vsixFilename: expected error for invalid JSON, got nil")
	}
}

func TestSplitLines_MultiLine(t *testing.T) {
	got := splitLines("alpha\nbeta\ngamma")
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("splitLines: got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitLines_EmptyString(t *testing.T) {
	got := splitLines("")
	if len(got) != 0 {
		t.Errorf("splitLines: got %d lines, want 0", len(got))
	}
}

func TestSplitLines_SkipsBlanks(t *testing.T) {
	got := splitLines("alpha\n\n  \nbeta\n")
	want := []string{"alpha", "beta"}
	if len(got) != len(want) {
		t.Fatalf("splitLines: got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	got := splitLines("one\ntwo\n")
	want := []string{"one", "two"}
	if len(got) != len(want) {
		t.Fatalf("splitLines: got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
