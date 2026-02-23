// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

// prd: prd006-vscode-extension R10

package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Binary names for VS Code extension tooling.
const (
	binNpm  = "npm"
	binCode = "code"
	binNpx  = "npx"
)

// vsCodeExtensionDir is the directory containing the VS Code extension source,
// relative to the orchestrator repository root.
const vsCodeExtensionDir = "vscode-extension"

// vsCodeExtensionID is the publisher-qualified extension identifier used by
// the code CLI for install and uninstall operations.
const vsCodeExtensionID = "mesh-intelligence.mage-orchestrator"

// VscodePush compiles the VS Code extension from source, packages it as a
// .vsix archive, and installs it into VS Code. It verifies that npm and
// the code CLI are available before proceeding.
func (o *Orchestrator) VscodePush() error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("vscode:push: getting working directory: %w", err)
	}
	extDir := filepath.Join(root, vsCodeExtensionDir)

	// Verify required tools.
	if _, err := exec.LookPath(binNpm); err != nil {
		return fmt.Errorf("vscode:push: npm is not installed or not on PATH; install Node.js from https://nodejs.org")
	}
	if _, err := exec.LookPath(binCode); err != nil {
		return fmt.Errorf("vscode:push: VS Code CLI (code) is not installed or not on PATH; open VS Code and run 'Shell Command: Install code command in PATH'")
	}

	// Step 1: npm install.
	logf("vscode:push: installing dependencies")
	installCmd := exec.Command(binNpm, "install")
	installCmd.Dir = extDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("vscode:push: npm install failed: %w", err)
	}

	// Step 2: compile TypeScript.
	logf("vscode:push: compiling TypeScript")
	compileCmd := exec.Command(binNpm, "run", "compile")
	compileCmd.Dir = extDir
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr
	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("vscode:push: TypeScript compilation failed: %w", err)
	}

	// Step 3: determine .vsix filename from package.json.
	vsixName, err := vsixFilename(extDir)
	if err != nil {
		return fmt.Errorf("vscode:push: %w", err)
	}

	// Step 4: package as .vsix.
	logf("vscode:push: packaging extension as %s", vsixName)
	packageCmd := exec.Command(binNpx, "@vscode/vsce", "package")
	packageCmd.Dir = extDir
	packageCmd.Stdout = os.Stdout
	packageCmd.Stderr = os.Stderr
	if err := packageCmd.Run(); err != nil {
		return fmt.Errorf("vscode:push: vsce package failed: %w", err)
	}

	// Step 5: install the extension.
	vsixPath := filepath.Join(extDir, vsixName)
	logf("vscode:push: installing extension from %s", vsixPath)
	codeCmd := exec.Command(binCode, "--install-extension", vsixPath)
	codeCmd.Stdout = os.Stdout
	codeCmd.Stderr = os.Stderr
	if err := codeCmd.Run(); err != nil {
		return fmt.Errorf("vscode:push: code --install-extension failed: %w", err)
	}

	logf("vscode:push: done")
	return nil
}

// VscodePop uninstalls the VS Code extension. The operation is idempotent:
// it succeeds even if the extension is not currently installed.
func (o *Orchestrator) VscodePop() error {
	if _, err := exec.LookPath(binCode); err != nil {
		return fmt.Errorf("vscode:pop: VS Code CLI (code) is not installed or not on PATH; open VS Code and run 'Shell Command: Install code command in PATH'")
	}

	logf("vscode:pop: uninstalling extension %s", vsCodeExtensionID)
	cmd := exec.Command(binCode, "--uninstall-extension", vsCodeExtensionID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// The code CLI exits non-zero when the extension is not installed.
		// Check if it is actually installed before treating this as an error.
		listOut, listErr := exec.Command(binCode, "--list-extensions").Output()
		if listErr == nil && slices.Contains(splitLines(string(listOut)), vsCodeExtensionID) {
			return fmt.Errorf("vscode:pop: uninstall failed: %w", err)
		}
		logf("vscode:pop: extension was not installed (nothing to do)")
		return nil
	}

	logf("vscode:pop: done")
	return nil
}

// packageJSON holds the fields we need from a VS Code extension package.json.
type packageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// vsixFilename reads package.json in extDir and returns the expected .vsix
// filename: <name>-<version>.vsix.
func vsixFilename(extDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(extDir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("reading package.json: %w", err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", fmt.Errorf("parsing package.json: %w", err)
	}
	if pkg.Name == "" || pkg.Version == "" {
		return "", fmt.Errorf("package.json missing name or version field")
	}
	return pkg.Name + "-" + pkg.Version + ".vsix", nil
}

// splitLines splits s into non-empty trimmed lines.
func splitLines(s string) []string {
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
