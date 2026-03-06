// Copyright (c) 2026 Petar Djukic. All rights reserved.
// SPDX-License-Identifier: MIT

package orchestrator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ReleaseUpdate marks a release as complete in the project files. It sets all
// use-case statuses to "implemented" for the named release in
// docs/road-map.yaml and removes the release version from project.releases in
// configuration.yaml (DefaultConfigFile). Both files are rewritten using
// yaml.v3 node round-trip to preserve document structure and comments.
//
// Returns an error if the release version is not found in road-map.yaml, or
// if either file fails schema validation.
func (o *Orchestrator) ReleaseUpdate(version string) error {
	if err := updateRoadmapUCStatuses(version, "implemented"); err != nil {
		return err
	}
	if err := removeReleaseFromConfig(DefaultConfigFile, version); err != nil {
		return err
	}
	logf("release:update %s: done", version)
	return nil
}

// ReleaseClear reverses ReleaseUpdate. It resets all use-case statuses to
// "spec_complete" for the named release in docs/road-map.yaml and
// re-appends the release version to project.releases in configuration.yaml.
//
// Returns an error if the release version is not found in road-map.yaml, or
// if either file fails schema validation.
func (o *Orchestrator) ReleaseClear(version string) error {
	if err := updateRoadmapUCStatuses(version, "spec_complete"); err != nil {
		return err
	}
	if err := addReleaseToConfig(DefaultConfigFile, version); err != nil {
		return err
	}
	logf("release:clear %s: done", version)
	return nil
}

// updateRoadmapUCStatuses loads docs/road-map.yaml via yaml.v3 node API,
// finds the release matching version, sets all its use_cases[*].status
// values to newStatus, and writes the file back.
func updateRoadmapUCStatuses(version, newStatus string) error {
	const path = "docs/road-map.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("release update: read %s: %w", path, err)
	}

	// Validate structure via typed unmarshal before mutation.
	var doc RoadmapDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("release update: parse %s: %w", path, err)
	}
	found := false
	for _, rel := range doc.Releases {
		if rel.Version == version {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("release update: version %q not found in %s", version, path)
	}

	// Node round-trip to preserve comments and structure.
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("release update: node parse %s: %w", path, err)
	}

	if err := setRoadmapUCStatuses(&root, version, newStatus); err != nil {
		return fmt.Errorf("release update: mutate %s: %w", path, err)
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("release update: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("release update: write %s: %w", path, err)
	}
	logf("release update: set use-case statuses to %q for release %s in %s", newStatus, version, path)
	return nil
}

// setRoadmapUCStatuses mutates the yaml.Node tree of road-map.yaml, finding
// the release with the given version and setting all its use_cases[*].status
// scalar values to newStatus.
func setRoadmapUCStatuses(root *yaml.Node, version, newStatus string) error {
	// Unwrap document node.
	doc := root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		doc = doc.Content[0]
	}
	// doc should be a mapping node for the roadmap document.
	releases := mappingValue(doc, "releases")
	if releases == nil || releases.Kind != yaml.SequenceNode {
		return fmt.Errorf("releases key not found or not a sequence")
	}
	for _, relNode := range releases.Content {
		versionNode := mappingValue(relNode, "version")
		if versionNode == nil || versionNode.Value != version {
			continue
		}
		ucSeq := mappingValue(relNode, "use_cases")
		if ucSeq == nil || ucSeq.Kind != yaml.SequenceNode {
			continue
		}
		for _, ucNode := range ucSeq.Content {
			statusNode := mappingValue(ucNode, "status")
			if statusNode != nil {
				statusNode.Value = newStatus
			}
		}
		return nil
	}
	return fmt.Errorf("version %q not found in releases node tree", version)
}

// removeReleaseFromConfig loads configPath, removes version from
// project.releases, and writes it back via node round-trip.
func removeReleaseFromConfig(configPath, version string) error {
	return mutateConfigReleases(configPath, version, false)
}

// addReleaseToConfig loads configPath, appends version to project.releases
// if not already present, and writes it back via node round-trip.
func addReleaseToConfig(configPath, version string) error {
	return mutateConfigReleases(configPath, version, true)
}

// mutateConfigReleases is the shared implementation for removeReleaseFromConfig
// and addReleaseToConfig. When add is true, version is appended (if absent);
// when add is false, version is removed.
func mutateConfigReleases(configPath, version string, add bool) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("release config: read %s: %w", configPath, err)
	}

	// Validate via typed unmarshal.
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("release config: parse %s: %w", configPath, err)
	}

	// Node round-trip.
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("release config: node parse %s: %w", configPath, err)
	}

	if err := mutateProjectReleasesNode(&root, version, add); err != nil {
		// project.releases is optional; log and skip rather than hard-fail.
		logf("release config: project.releases not found in %s, skipping: %v", configPath, err)
		return nil
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("release config: marshal %s: %w", configPath, err)
	}
	if err := os.WriteFile(configPath, out, 0o644); err != nil {
		return fmt.Errorf("release config: write %s: %w", configPath, err)
	}
	action := "removed"
	if add {
		action = "added"
	}
	logf("release config: %s %q in project.releases in %s", action, version, configPath)
	return nil
}

// mutateProjectReleasesNode finds project.releases in the node tree and either
// removes or appends version. Returns an error if project.releases is absent.
func mutateProjectReleasesNode(root *yaml.Node, version string, add bool) error {
	doc := root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		doc = doc.Content[0]
	}
	projectNode := mappingValue(doc, "project")
	if projectNode == nil {
		return fmt.Errorf("project key not found")
	}
	releasesNode := mappingValue(projectNode, "releases")
	if releasesNode == nil {
		return fmt.Errorf("project.releases key not found")
	}
	if releasesNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("project.releases is not a sequence")
	}

	if add {
		// Append only if not already present.
		for _, child := range releasesNode.Content {
			if child.Value == version {
				return nil // already present
			}
		}
		releasesNode.Content = append(releasesNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: version,
		})
	} else {
		// Remove all occurrences of version.
		filtered := releasesNode.Content[:0]
		for _, child := range releasesNode.Content {
			if child.Value != version {
				filtered = append(filtered, child)
			}
		}
		releasesNode.Content = filtered
	}
	return nil
}

// mappingValue returns the value node for the given key in a yaml mapping
// node, or nil if not found.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// releaseVersionsFromConfig returns the list of releases in project.releases
// from the given config file. Used in tests.
func releaseVersionsFromConfig(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Project.Releases, nil
}

// roadmapUCStatuses returns a map of UC ID → status for the given release
// version. Used in tests.
func roadmapUCStatuses(roadmapPath, version string) (map[string]string, error) {
	data, err := os.ReadFile(roadmapPath)
	if err != nil {
		return nil, err
	}
	var doc RoadmapDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	for _, rel := range doc.Releases {
		if rel.Version == version {
			out := make(map[string]string, len(rel.UseCases))
			for _, uc := range rel.UseCases {
				out[uc.ID] = uc.Status
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("version %q not found", version)
}

