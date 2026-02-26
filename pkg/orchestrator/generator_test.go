// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import "testing"

// --- generationDate ---

func TestGenerationDate_ValidBranch(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "generation-"}}}
	got := o.generationDate("generation-2026-02-12-07-13-55")
	want := "2026-02-12"
	if got != want {
		t.Errorf("generationDate() = %q, want %q", got, want)
	}
}

func TestGenerationDate_NoPrefix(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "generation-"}}}
	got := o.generationDate("main")
	if got != "" {
		t.Errorf("generationDate(main) = %q, want empty", got)
	}
}

func TestGenerationDate_ShortRest(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "generation-"}}}
	got := o.generationDate("generation-20")
	if got != "" {
		t.Errorf("generationDate(short) = %q, want empty", got)
	}
}

func TestGenerationDate_CustomPrefix(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "gen-"}}}
	got := o.generationDate("gen-2026-03-01-12-00-00")
	want := "2026-03-01"
	if got != want {
		t.Errorf("generationDate() = %q, want %q", got, want)
	}
}

// --- generationDateCompact ---

func TestGenerationDateCompact_Valid(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "generation-"}}}
	got := o.generationDateCompact("generation-2026-02-12-07-13-55")
	want := "20260212"
	if got != want {
		t.Errorf("generationDateCompact() = %q, want %q", got, want)
	}
}

func TestGenerationDateCompact_Invalid(t *testing.T) {
	t.Parallel()
	o := &Orchestrator{cfg: Config{Generation: GenerationConfig{Prefix: "generation-"}}}
	got := o.generationDateCompact("main")
	if got != "" {
		t.Errorf("generationDateCompact(main) = %q, want empty", got)
	}
}

// --- generationName ---

func TestGenerationName_StripsSuffixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tag  string
		want string
	}{
		{"generation-2026-02-12-07-13-55-start", "generation-2026-02-12-07-13-55"},
		{"generation-2026-02-12-07-13-55-finished", "generation-2026-02-12-07-13-55"},
		{"generation-2026-02-12-07-13-55-merged", "generation-2026-02-12-07-13-55"},
		{"generation-2026-02-12-07-13-55-abandoned", "generation-2026-02-12-07-13-55"},
		{"generation-2026-02-12-07-13-55", "generation-2026-02-12-07-13-55"},
		{"unrelated-tag", "unrelated-tag"},
	}
	for _, tt := range tests {
		got := generationName(tt.tag)
		if got != tt.want {
			t.Errorf("generationName(%q) = %q, want %q", tt.tag, got, tt.want)
		}
	}
}
