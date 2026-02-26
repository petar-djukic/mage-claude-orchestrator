// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

// --- countLines ---

func TestCountLines_MultipleLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("line 1\nline 2\nline 3\n"), 0644)

	got, err := countLines(path)
	if err != nil {
		t.Fatalf("countLines: %v", err)
	}
	if got != 3 {
		t.Errorf("countLines = %d, want 3", got)
	}
}

func TestCountLines_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	os.WriteFile(path, []byte(""), 0644)

	got, err := countLines(path)
	if err != nil {
		t.Fatalf("countLines: %v", err)
	}
	if got != 0 {
		t.Errorf("countLines(empty) = %d, want 0", got)
	}
}

func TestCountLines_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "noeol.go")
	os.WriteFile(path, []byte("line 1\nline 2"), 0644)

	got, err := countLines(path)
	if err != nil {
		t.Fatalf("countLines: %v", err)
	}
	if got != 2 {
		t.Errorf("countLines(no-eol) = %d, want 2", got)
	}
}

func TestCountLines_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := countLines("/nonexistent/file.go")
	if err == nil {
		t.Error("countLines(missing) should return error")
	}
}

// --- countWordsInFile ---

func TestCountWordsInFile_Basic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "words.txt")
	os.WriteFile(path, []byte("hello world foo bar"), 0644)

	got, err := countWordsInFile(path)
	if err != nil {
		t.Fatalf("countWordsInFile: %v", err)
	}
	if got != 4 {
		t.Errorf("countWordsInFile = %d, want 4", got)
	}
}

func TestCountWordsInFile_MultipleSpaces(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "spaces.txt")
	os.WriteFile(path, []byte("  hello   world  \n\n  foo  "), 0644)

	got, err := countWordsInFile(path)
	if err != nil {
		t.Fatalf("countWordsInFile: %v", err)
	}
	if got != 3 {
		t.Errorf("countWordsInFile(multi-space) = %d, want 3", got)
	}
}

func TestCountWordsInFile_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte(""), 0644)

	got, err := countWordsInFile(path)
	if err != nil {
		t.Fatalf("countWordsInFile: %v", err)
	}
	if got != 0 {
		t.Errorf("countWordsInFile(empty) = %d, want 0", got)
	}
}

func TestCountWordsInFile_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := countWordsInFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("countWordsInFile(missing) should return error")
	}
}
