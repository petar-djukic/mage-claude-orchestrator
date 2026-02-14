// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// orchestratorModule is the Go module path for this orchestrator library.
const orchestratorModule = "github.com/mesh-intelligence/mage-claude-orchestrator"

// Scaffold sets up a target Go repository to use the orchestrator.
// It copies the orchestrator.go template into magefiles/, detects
// project structure, generates configuration.yaml, and wires the
// Go module dependencies.
func (o *Orchestrator) Scaffold(targetDir, orchestratorRoot string) error {
	logf("scaffold: targetDir=%s orchestratorRoot=%s", targetDir, orchestratorRoot)

	mageDir := filepath.Join(targetDir, "magefiles")

	// 1. Copy orchestrator.go template into magefiles/.
	dst := filepath.Join(mageDir, "orchestrator.go")
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("magefiles/orchestrator.go already exists in %s", targetDir)
	}
	src := filepath.Join(orchestratorRoot, "orchestrator.go")
	logf("scaffold: copying %s -> %s", src, dst)
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("copying orchestrator.go: %w", err)
	}

	// 2. Detect project structure.
	modulePath, err := detectModulePath(targetDir)
	if err != nil {
		return fmt.Errorf("detecting module path: %w", err)
	}
	logf("scaffold: detected module_path=%s", modulePath)

	mainPkg := detectMainPackage(targetDir, modulePath)
	logf("scaffold: detected main_package=%s", mainPkg)

	srcDirs := detectSourceDirs(targetDir)
	logf("scaffold: detected go_source_dirs=%v", srcDirs)

	binName := detectBinaryName(modulePath)
	logf("scaffold: detected binary_name=%s", binName)

	// 3. Generate configuration.yaml in the target root.
	cfg := DefaultConfig()
	cfg.ModulePath = modulePath
	cfg.BinaryName = binName
	cfg.MainPackage = mainPkg
	cfg.GoSourceDirs = srcDirs

	cfgPath := filepath.Join(targetDir, DefaultConfigFile)
	logf("scaffold: writing %s", cfgPath)
	if err := writeScaffoldConfig(cfgPath, cfg); err != nil {
		return fmt.Errorf("writing configuration.yaml: %w", err)
	}

	// 4. Wire magefiles/go.mod.
	logf("scaffold: wiring magefiles/go.mod")
	absOrch, err := filepath.Abs(orchestratorRoot)
	if err != nil {
		return fmt.Errorf("resolving orchestrator path: %w", err)
	}
	if err := scaffoldMageGoMod(mageDir, modulePath, absOrch); err != nil {
		return fmt.Errorf("wiring magefiles/go.mod: %w", err)
	}

	// 5. Verify.
	logf("scaffold: verifying with mage -l")
	if err := verifyMage(targetDir); err != nil {
		return fmt.Errorf("mage verification: %w", err)
	}

	logf("scaffold: done")
	return nil
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// detectModulePath reads go.mod in the target directory and extracts
// the module path from the first "module" directive.
func detectModulePath(targetDir string) (string, error) {
	modPath := filepath.Join(targetDir, "go.mod")
	f, err := os.Open(modPath)
	if err != nil {
		return "", fmt.Errorf("opening go.mod: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s", modPath)
}

// detectMainPackage scans cmd/ for directories containing main.go.
// Returns the module-relative import path of the first main package
// found, or empty string if none exist.
func detectMainPackage(targetDir, modulePath string) string {
	cmdDir := filepath.Join(targetDir, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mainGo := filepath.Join(cmdDir, e.Name(), "main.go")
		if _, err := os.Stat(mainGo); err == nil {
			return modulePath + "/cmd/" + e.Name()
		}
	}
	// Check for main.go directly in cmd/.
	if _, err := os.Stat(filepath.Join(cmdDir, "main.go")); err == nil {
		return modulePath + "/cmd"
	}
	return ""
}

// detectSourceDirs returns existing Go source directories in the target.
func detectSourceDirs(targetDir string) []string {
	candidates := []string{"cmd/", "pkg/", "internal/", "tests/"}
	var found []string
	for _, d := range candidates {
		if _, err := os.Stat(filepath.Join(targetDir, d)); err == nil {
			found = append(found, d)
		}
	}
	return found
}

// detectBinaryName extracts a binary name from the module path by
// using its last path component.
func detectBinaryName(modulePath string) string {
	parts := strings.Split(modulePath, "/")
	if len(parts) == 0 {
		return "app"
	}
	return parts[len(parts)-1]
}

// writeScaffoldConfig marshals cfg as YAML and writes it to path.
func writeScaffoldConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	header := "# Orchestrator configuration — generated by scaffold.\n# See docs/ARCHITECTURE.md for field descriptions.\n\n"
	return os.WriteFile(path, append([]byte(header), data...), 0o644)
}

// scaffoldMageGoMod ensures magefiles/go.mod exists with the orchestrator
// dependency and replace directive pointing to the local checkout.
// If magefiles/go.mod does not exist, it creates one.
func scaffoldMageGoMod(mageDir, rootModule, orchestratorRoot string) error {
	goMod := filepath.Join(mageDir, "go.mod")

	// Create magefiles/go.mod if it does not exist.
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		mageModule := rootModule + "/magefiles"
		logf("scaffold: creating %s (module %s)", goMod, mageModule)
		initCmd := exec.Command(binGo, "mod", "init", mageModule)
		initCmd.Dir = mageDir
		if err := initCmd.Run(); err != nil {
			return fmt.Errorf("go mod init: %w", err)
		}
	}

	// Add replace directive.
	replaceCmd := exec.Command(binGo, "mod", "edit",
		"-replace", orchestratorModule+"="+orchestratorRoot)
	replaceCmd.Dir = mageDir
	if err := replaceCmd.Run(); err != nil {
		return fmt.Errorf("go mod edit -replace: %w", err)
	}

	// Tidy resolves imports from orchestrator.go and adds required modules.
	tidyCmd := exec.Command(binGo, "mod", "tidy")
	tidyCmd.Dir = mageDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

// verifyMage runs mage -l in the target directory to confirm the
// orchestrator template is correctly wired.
func verifyMage(targetDir string) error {
	cmd := exec.Command(binMage, "-l")
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
