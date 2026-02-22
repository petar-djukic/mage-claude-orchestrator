// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"testing"
)

func TestNumberLines_Normal(t *testing.T) {
	input := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := numberLines(input)
	want := "1 | package main\n3 | import \"fmt\"\n5 | func main() {\n6 | \tfmt.Println(\"hello\")\n7 | }"
	if got != want {
		t.Errorf("numberLines:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestNumberLines_BlankLinesOmitted(t *testing.T) {
	input := "a\n\n\nb\n"
	got := numberLines(input)
	want := "1 | a\n4 | b"
	if got != want {
		t.Errorf("numberLines:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestNumberLines_SingleLine(t *testing.T) {
	input := "package main\n"
	got := numberLines(input)
	want := "1 | package main"
	if got != want {
		t.Errorf("numberLines:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestNumberLines_Empty(t *testing.T) {
	got := numberLines("")
	if got != "" {
		t.Errorf("numberLines empty: got %q, want empty", got)
	}
}

func TestNumberLines_WhitespaceOnlyLines(t *testing.T) {
	input := "a\n  \n\t\nb\n"
	got := numberLines(input)
	want := "1 | a\n4 | b"
	if got != want {
		t.Errorf("numberLines:\ngot:  %q\nwant: %q", got, want)
	}
}
