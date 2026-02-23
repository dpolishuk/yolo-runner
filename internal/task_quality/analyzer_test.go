package taskquality

import (
	"strings"
	"testing"
)

func TestAssessTaskQuality_LowSampleScoresBelowThresholdAndReturnsIssues(t *testing.T) {
	task := TaskInput{
		Title:               "Improve reliability",
		Description:         "Make the system more stable and robust across all flows.",
		AcceptanceCriteria:  "Make the system more stable and robust.",
		Deliverables:        "",
		TestingPlan:         "",
		DefinitionOfDone:    "Code compiles.",
		DependenciesContext: "",
	}

	result := AssessTaskQuality(task)
	if result.Score >= 80 {
		t.Fatalf("expected low-quality score below 80, got %d", result.Score)
	}

	issues := joinIssues(result.Issues)
	requiredMissing := []string{
		"missing required field: Deliverables",
		"missing required field: Testing Plan",
		"missing required field: Dependencies/Context",
	}
	for _, needle := range requiredMissing {
		if !contains(issues, needle) {
			t.Fatalf("expected issue containing %q in %q", needle, issues)
		}
	}
	if !contains(issues, "acceptance criteria should use Given/When/Then format") {
		t.Fatalf("expected issue for ambiguous acceptance format, got %q", issues)
	}
	if !contains(issues, "clarity: avoid vague wording") {
		t.Fatalf("expected issue for vague wording, got %q", issues)
	}
}

func TestAssessTaskQuality_HighQualitySampleScoresAboveThresholdAndIsClean(t *testing.T) {
	task := TaskInput{
		Title:       "Implement task timeout classification for runner stalls",
		Description: "Add deterministic detection of stalled runner tasks in yolo-agent so stalled tasks are blocked with explicit reasons.",
		AcceptanceCriteria: "- Given a task running longer than the configured watchdog timeout with no output, when the task remains quiet for the full window, then the task status becomes blocked and triage reason includes `stall`.\n" +
			"- Given a task that receives output again, when run is resumed, then it transitions back to in_progress with no task closure.\n" +
			"- Given a task that times out repeatedly, when blocked conditions persist, then the task is marked failed only after retry policy is exhausted.",
		Deliverables: `- internal/task_quality/analyzer.go
- cmd/yolo-agent integration for task quality gating (future task)
- Unit tests with both good and bad samples`,
		TestingPlan: `- go test ./internal/task_quality -run TestAssessTaskQuality
- go test ./internal/agent -run TestRunTask
- manual check: ensure blocked tasks emit triage reason with stall`,
		DefinitionOfDone: `- Tests cover low and high quality samples.
- Evidence captured: test failures before analyzer implementation and pass after.
- No behavior regression in existing task scheduling paths.`,
		DependenciesContext: `- Dependencies: yolo-agent runner watchdog path.
- Non-goals: UI/UX changes to TUI task list.
- Risks: command logs may be noisy during long-running tasks.
- Assumptions: task descriptions follow issue template sections.`,
	}

	result := AssessTaskQuality(task)
	if result.Score < 90 {
		t.Fatalf("expected high-quality score at or above 90, got %d", result.Score)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no quality issues, got: %q", result.Issues)
	}
}

func joinIssues(issues []string) string {
	return strings.Join(issues, "\n")
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
