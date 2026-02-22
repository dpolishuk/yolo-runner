package docs

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type releaseWorkflow struct {
	Jobs map[string]releaseWorkflowJob `yaml:"jobs"`
}

type releaseWorkflowJob struct {
	Strategy struct {
		Matrix map[string]interface{} `yaml:"matrix"`
	} `yaml:"strategy"`
	Steps []releaseWorkflowStep `yaml:"steps"`
}

type releaseWorkflowStep struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

func TestReleaseWorkflowDefinesBuildMatrixAndChecksumArtifacts(t *testing.T) {
	workflow := readRepoFile(t, ".github", "workflows", "release.yml")

	var parsed releaseWorkflow
	if err := yaml.Unmarshal([]byte(workflow), &parsed); err != nil {
		t.Fatalf("unmarshal workflow YAML: %v", err)
	}

	job := findReleaseMatrixJob(t, parsed)
	matrixEntries := releaseMatrixEntries(job.Strategy.Matrix)
	if len(matrixEntries) == 0 {
		t.Fatalf("release workflow must define strategy.matrix.include entries")
	}

	seenArtifactNames := map[string]struct{}{}
	for _, entry := range matrixEntries {
		artifact, ok := entry["artifact"]
		if !ok || strings.TrimSpace(artifact) == "" {
			t.Fatalf("release matrix entry missing artifact name: %#v", entry)
		}
		seenArtifactNames[artifact] = struct{}{}
	}

	expectedArtifacts := releaseArtifactsFromInstallMatrix(t)
	for artifact := range expectedArtifacts {
		if _, ok := seenArtifactNames[artifact]; !ok {
			t.Fatalf("release matrix missing supported artifact %q", artifact)
		}
	}
	for artifact := range seenArtifactNames {
		if _, ok := expectedArtifacts[artifact]; !ok {
			t.Fatalf("release matrix defines unsupported artifact %q", artifact)
		}
	}

	runContent := ""
	for _, step := range job.Steps {
		runContent += step.Run
		runContent += " "
	}
	if !strings.Contains(strings.ToLower(runContent), "sha256sum") {
		t.Fatalf("release workflow missing checksum generation (expected sha256sum usage)")
	}
	if !strings.Contains(runContent, "checksums-${{ matrix.artifact }}.txt") {
		t.Fatalf("release workflow missing checksum artifact generation (expected checksums-${{ matrix.artifact }}.txt)")
	}
}

func TestReleaseWorkflowContainsSupportedPlatformsInArtifactNaming(t *testing.T) {
	workflow := readRepoFile(t, ".github", "workflows", "release.yml")
	expectedArtifacts := releaseArtifactsFromInstallMatrix(t)

	for artifact := range expectedArtifacts {
		if !strings.Contains(workflow, artifact) {
			t.Fatalf("release workflow missing artifact naming for matrix entry %q", artifact)
		}
	}

	unsupported := "yolo-runner_windows_arm64"
	if strings.Contains(workflow, unsupported) {
		t.Fatalf("release workflow includes unsupported matrix artifact %q", unsupported)
	}
}

func TestReleaseWorkflowDefinesCrossPlatformBuildAndPublishSteps(t *testing.T) {
	workflow := readRepoFile(t, ".github", "workflows", "release.yml")

	if !strings.Contains(workflow, "actions/checkout@v4") {
		t.Fatalf("release workflow missing checkout step")
	}
	if !strings.Contains(workflow, "actions/setup-go@") {
		t.Fatalf("release workflow missing setup-go step")
	}

	var parsed releaseWorkflow
	if err := yaml.Unmarshal([]byte(workflow), &parsed); err != nil {
		t.Fatalf("unmarshal workflow YAML: %v", err)
	}
	job := findReleaseMatrixJob(t, parsed)

	buildStepFound := false
	publishStepFound := false
	packageStepFound := false
	checksumStepFound := false

	for _, step := range job.Steps {
		run := strings.ToLower(step.Run)
		if !buildStepFound && strings.Contains(run, "go build") && strings.Contains(run, "${{ matrix.os }}") && strings.Contains(run, "${{ matrix.arch }}") {
			buildStepFound = true
		}
		if !packageStepFound && strings.Contains(run, "${{ matrix.artifact }}") &&
			(strings.Contains(run, "tar -czf") || strings.Contains(run, "zip")) {
			packageStepFound = true
		}
		if !checksumStepFound && strings.Contains(run, "sha256sum") && strings.Contains(run, "checksums-${{ matrix.artifact }}.txt") {
			checksumStepFound = true
		}
		if strings.Contains(step.Uses, "softprops/action-gh-release") {
			files, ok := step.With["files"]
			if !ok {
				t.Fatalf("release workflow publish step missing files input")
			}
			if !strings.Contains(files, "dist/${{ matrix.artifact }}") {
				t.Fatalf("release workflow publish step must include matrix artifact")
			}
			if !strings.Contains(files, "dist/checksums-${{ matrix.artifact }}.txt") {
				t.Fatalf("release workflow publish step must include matching checksum artifact path")
			}
			publishStepFound = true
		}
	}

	if !buildStepFound {
		t.Fatalf("release workflow missing matrix-aware go build step")
	}
	if !packageStepFound {
		t.Fatalf("release workflow missing matrix artifact packaging step")
	}
	if !checksumStepFound {
		t.Fatalf("release workflow missing checksum generation step for checksums-${{ matrix.artifact }}.txt")
	}
	if !publishStepFound {
		t.Fatalf("release workflow missing artifact publish step using softprops/action-gh-release")
	}
}

func TestReleaseWorkflowUsesChecksumFilenamePerArtifact(t *testing.T) {
	workflow := readRepoFile(t, ".github", "workflows", "release.yml")

	if !strings.Contains(workflow, "dist/checksums-${{ matrix.artifact }}.txt") {
		t.Fatalf("release workflow must use per-artifact checksum filename with matrix artifact placeholder")
	}
	if strings.Contains(workflow, "dist/checksums.txt") {
		t.Fatalf("release workflow must not use shared checksum filename dist/checksums.txt in matrix release jobs")
	}
}

func findReleaseMatrixJob(t *testing.T, wf releaseWorkflow) releaseWorkflowJob {
	t.Helper()
	for _, job := range wf.Jobs {
		if _, ok := job.Strategy.Matrix["os"]; ok {
			return job
		}
		if _, ok := job.Strategy.Matrix["include"]; ok {
			return job
		}
	}
	t.Fatalf("release workflow missing strategy.matrix")
	return releaseWorkflowJob{}
}

func releaseMatrixEntries(matrix map[string]interface{}) []map[string]string {
	rawInclude, ok := matrix["include"]
	if !ok {
		return nil
	}

	include, ok := rawInclude.([]interface{})
	if !ok {
		return nil
	}

	entries := make([]map[string]string, 0, len(include))
	for _, rawEntry := range include {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			continue
		}

		parsed := map[string]string{}
		for key, rawValue := range entry {
			value, ok := rawValue.(string)
			if !ok {
				continue
			}
			parsed[key] = value
		}
		entries = append(entries, parsed)
	}

	return entries
}

func releaseArtifactsFromInstallMatrix(t *testing.T) map[string]struct{} {
	t.Helper()

	matrix := readRepoFile(t, "docs", "install-matrix.md")
	artifacts := map[string]struct{}{}

	for _, line := range strings.Split(matrix, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Contains(strings.ToLower(line), "platform") || strings.Contains(line, "| ---") {
			continue
		}

		cells := splitMarkdownRow(line)
		if len(cells) < 2 {
			continue
		}

		platform := strings.TrimSpace(cells[0])
		arch := strings.TrimSpace(cells[1])
		if arch == "" {
			continue
		}

		artifactOS, ok := releaseArtifactOS(platform)
		if !ok {
			continue
		}

		artifactSuffix := "tar.gz"
		if artifactOS == "windows" {
			artifactSuffix = "zip"
		}

		artifact := fmt.Sprintf("yolo-runner_%s_%s.%s", artifactOS, arch, artifactSuffix)
		artifacts[artifact] = struct{}{}
	}

	if len(artifacts) == 0 {
		t.Fatalf("no release artifacts derived from install matrix")
	}

	return artifacts
}

func releaseArtifactOS(platform string) (string, bool) {
	switch strings.ToLower(platform) {
	case "macos":
		return "darwin", true
	case "linux":
		return "linux", true
	case "windows":
		return "windows", true
	default:
		return "", false
	}
}
