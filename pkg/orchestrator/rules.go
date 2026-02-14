// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// collectProjectRules reads all .md files from .claude/rules/ under repoDir
// and returns their concatenated content with filename headers. Returns ""
// if the directory does not exist or contains no markdown files.
func collectProjectRules(repoDir string) string {
	rulesDir := filepath.Join(repoDir, ".claude", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			continue
		}
		buf.WriteString("### " + e.Name() + "\n\n")
		buf.Write(data)
		buf.WriteString("\n\n")
	}
	return buf.String()
}
