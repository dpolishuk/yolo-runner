package main

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEDocumentsYoloRunnerInit(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	readmeContent := string(content)

	// Check that init command is documented
	if !strings.Contains(readmeContent, "yolo-runner init") {
		t.Error("README does not document 'yolo-runner init' command")
	}

	// Check that init usage example is present
	if !strings.Contains(readmeContent, "./bin/yolo-runner init --repo .") {
		t.Error("README does not include init command usage example")
	}

	// Check that init section explains what it does
	if !strings.Contains(readmeContent, "agent installation") {
		t.Error("README does not explain that init performs agent installation")
	}
}

func TestREADMEExplainsWhyRunnerRefusesToStart(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	readmeContent := string(content)

	// Check that README explains runner refuses to start if agent missing
	if !strings.Contains(readmeContent, "runner refuses to start") {
		t.Error("README does not explain that runner refuses to start")
	}

	// Check that README mentions missing agent file as cause
	if !strings.Contains(readmeContent, "missing agent file") {
		t.Error("README does not mention missing agent file as cause")
	}

	// Check that README mentions missing permission: allow as cause
	if !strings.Contains(readmeContent, "permission: allow") {
		t.Error("README does not mention missing permission: allow as cause")
	}

	// Check that README mentions agent validation at startup
	if !strings.Contains(readmeContent, "validates the agent") {
		t.Error("README does not mention agent validation at startup")
	}
}

func TestREADMEIncludesTroubleshootingSteps(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	readmeContent := string(content)

	// Check that there's a troubleshooting section
	if !strings.Contains(readmeContent, "Troubleshooting") {
		t.Error("README does not include a troubleshooting section")
	}

	// Check that recovery steps are documented
	if !strings.Contains(readmeContent, "Recovery steps") {
		t.Error("README does not include recovery steps")
	}

	// Check that the init command is mentioned as recovery step
	if !strings.Contains(readmeContent, "yolo-runner init") {
		t.Error("README does not mention init command in troubleshooting")
	}

	// Check that agent file existence verification is mentioned
	if !strings.Contains(readmeContent, ".opencode/agent/yolo.md") {
		t.Error("README does not mention agent file path verification")
	}

	// Check that permission: allow verification is mentioned
	if !strings.Contains(readmeContent, "permission: allow") {
		t.Error("README does not mention permission: allow verification")
	}
}
