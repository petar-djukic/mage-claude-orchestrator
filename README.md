# mage-claude-orchestrator

Go library for automating AI code generation via Claude Code. Consuming projects import this library through their Magefile and run generation cycles that propose tasks (measure) and execute them in isolated git worktrees (stitch). Claude runs inside a podman container for process isolation.

## Prerequisites

| Tool | Purpose |
|------|---------|
| Go 1.24+ | Build and run |
| [Mage](https://magefile.org/) | Build system; orchestrator methods are exposed as Mage targets |
| [Beads](https://github.com/mesh-intelligence/beads) (bd) | Git-backed issue tracking |
| Podman | Container runtime for Claude execution |

## Podman Setup

Claude runs inside a podman container. The orchestrator wraps every Claude invocation in `podman run` with the repository directory mounted into the container.

### 1. Install Podman

On macOS:

```bash
brew install podman
podman machine init
podman machine start
```

On Linux (Fedora/RHEL):

```bash
sudo dnf install podman
```

On Linux (Debian/Ubuntu):

```bash
sudo apt install podman
```

### 2. Prepare the Claude Image

The container image must have the `claude` CLI installed. Build or pull an image that includes it, then set `podman_image` in your configuration.yaml.

### 3. Verify

```bash
podman run --rm <your-image> claude --version
```

If this prints a version string, podman is ready.

### 4. Configure

Set `podman_image` in `configuration.yaml`:

```yaml
podman_image: "your-claude-image:latest"
```

Optional extra arguments (environment variables, additional mounts):

```yaml
podman_args:
  - "-e"
  - "ANTHROPIC_API_KEY=sk-..."
```

The orchestrator runs a pre-flight check before every measure and stitch phase. If podman is not installed or cannot start a container, it exits with instructions pointing here.

## Configuration

All options live in `configuration.yaml` at the repository root. Generate a default file:

```bash
mage generator:init
```

This creates `configuration.yaml` with all defaults filled in. Edit the project-specific fields (`module_path`, `binary_name`, `main_package`, `go_source_dirs`, `podman_image`) before running.

### Configuration Reference

| Field | Default | Description |
|-------|---------|-------------|
| module_path | (required) | Go module path |
| binary_name | (required) | Compiled binary name |
| binary_dir | bin | Output directory for binaries |
| main_package | (required) | Path to main.go entry point |
| go_source_dirs | (required) | Directories with Go source files |
| version_file | | Path to version.go file |
| gen_prefix | generation- | Prefix for generation branch names |
| beads_dir | .beads/ | Beads database directory |
| cobbler_dir | .cobbler/ | Cobbler scratch directory |
| magefiles_dir | magefiles | Directory skipped when deleting Go files |
| secrets_dir | .secrets | Directory containing token files |
| default_token_file | claude.json | Default credential filename |
| spec_globs | {} | Label to glob pattern for word-count stats |
| seed_files | {} | Destination to template source file paths |
| measure_prompt | | Path to custom measure prompt template |
| stitch_prompt | | Path to custom stitch prompt template |
| claude_args | (see below) | CLI arguments for Claude execution |
| silence_agent | true | Suppress Claude stdout |
| max_issues | 10 | Max tasks per measure or stitch phase |
| cycles | 1 | Number of measure+stitch cycles per run |
| estimated_lines_min | 250 | Minimum estimated lines per task |
| estimated_lines_max | 350 | Maximum estimated lines per task |
| cleanup_dirs | [] | Directories to remove after generation stop or reset |
| podman_image | (required) | Container image for Claude execution |
| podman_args | [] | Additional podman run arguments |
| user_prompt | | Additional context for the measure prompt |
| generation_branch | | Specific generation branch (auto-detected if empty) |
| token_file | | Override credential filename |

Default claude_args: `--dangerously-skip-permissions -p --verbose --output-format stream-json`

## Quick Start

In your consuming project's Magefile:

```go
package main

import orchestrator "github.com/mesh-intelligence/mage-claude-orchestrator"

var o *orchestrator.Orchestrator

func init() {
    var err error
    o, err = orchestrator.NewFromFile("configuration.yaml")
    if err != nil {
        panic(err)
    }
}

func GeneratorStart() error { return o.GeneratorStart() }
func GeneratorRun() error   { return o.GeneratorRun() }
func GeneratorStop() error  { return o.GeneratorStop() }
```

Then run:

```bash
mage generator:init        # Create configuration.yaml with defaults
# Edit configuration.yaml: set module_path, binary_name, podman_image, etc.
mage generator:start       # Create generation branch from main
mage generator:run         # Run measure+stitch cycles
mage generator:stop        # Merge generation into main
```

If a run is interrupted, `mage generator:resume` recovers state and continues. To discard a generation, `mage generator:reset` returns to a clean main.

## License

MIT
