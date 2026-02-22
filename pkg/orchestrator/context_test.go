// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"reflect"
	"testing"
)

func TestNumberLines_Normal(t *testing.T) {
	input := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	got := numberLines(input)
	want := []string{
		"1 | package main",
		"3 | import \"fmt\"",
		"5 | func main() {",
		"6 | \tfmt.Println(\"hello\")",
		"7 | }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("numberLines:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestNumberLines_BlankLinesOmitted(t *testing.T) {
	input := "a\n\n\nb\n"
	got := numberLines(input)
	want := []string{
		"1 | a",
		"4 | b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("numberLines:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestNumberLines_SingleLine(t *testing.T) {
	input := "package main\n"
	got := numberLines(input)
	want := []string{"1 | package main"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("numberLines:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestNumberLines_Empty(t *testing.T) {
	got := numberLines("")
	if len(got) != 0 {
		t.Errorf("numberLines empty: got %v, want nil/empty", got)
	}
}

func TestNumberLines_WhitespaceOnlyLines(t *testing.T) {
	input := "a\n  \n\t\nb\n"
	got := numberLines(input)
	want := []string{
		"1 | a",
		"4 | b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("numberLines:\ngot:  %v\nwant: %v", got, want)
	}
}
