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
- go test ./internal/task_quality -run TestAssessTaskQuality_ScoreIsBoundedToHundred`,
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

func TestAssessTaskQuality_AcceptanceCriteriaCoverageGapsAreDetected(t *testing.T) {
	task := fullyPopulatedTaskQualityInput()
	task.AcceptanceCriteria = `- Given a user submits a valid request, when the worker processes it, then status becomes in_progress.
- Given a user submits an invalid request, when validation fails, then status becomes blocked with a validation reason.`
	task.TestingPlan = `- go test ./internal/task_quality -run TestAssessTaskQuality`

	result := AssessTaskQuality(task)
	if len(result.Issues) == 0 {
		t.Fatalf("expected issues for acceptance criteria coverage, got none")
	}
	issues := joinIssues(result.Issues)
	if !contains(issues, "acceptance criteria coverage gaps") {
		t.Fatalf("expected acceptance criteria coverage gap issue, got %q", result.Issues)
	}
	if contains(issues, "acceptance criteria should use Given/When/Then format") {
		t.Fatalf("unexpected wording-format issue for valid Given/When/Then criteria")
	}
}

func TestAssessTaskQuality_AdequateAcceptanceCriteriaCoveragePasses(t *testing.T) {
	task := fullyPopulatedTaskQualityInput()
	task.AcceptanceCriteria = `- Given a user submits a valid request, when the worker processes it, then status becomes in_progress.
- Given a user submits an invalid request, when validation fails, then status becomes blocked with a validation reason.`
	task.TestingPlan = `- go test ./internal/task_quality -run TestAssessTaskQuality
- go test ./internal/task_quality -run TestAssessTaskQuality_HighQualitySampleScoresAboveThresholdAndIsClean`

	result := AssessTaskQuality(task)
	if result.Score < 90 {
		t.Fatalf("expected high-quality score with full acceptance coverage, got %d", result.Score)
	}
	issues := joinIssues(result.Issues)
	if contains(issues, "acceptance criteria coverage gaps") {
		t.Fatalf("did not expect acceptance criteria coverage gaps, got: %q", result.Issues)
	}
}

func TestAssessTaskQuality_ScoreIsBoundedToHundred(t *testing.T) {
	task := TaskInput{
		Title:       "Implement deterministic task quality scoring in analyzer",
		Description: "Add deterministic task-quality scoring that returns a bounded score from 0 to 100 for all tasks.",
		AcceptanceCriteria: "- Given a task with complete fields, when the analyzer runs, then the score is between 0 and 100.\n" +
			"- Given a task with missing fields, when the analyzer runs, then the score and issues indicate incompleteness.",
		Deliverables: `- Analyzer scoring implementation updates
- Unit tests covering score bounds and issue collection`,
		TestingPlan: `- go test ./internal/task_quality
- go test ./internal/task_quality -run TestAssessTaskQuality`,
		DefinitionOfDone: `- Analyzer returns bounded score and ordered issues.
- All task quality tests pass.`,
		DependenciesContext: `- Context: Internal task quality rubric.
- Risks: false positives due heuristic matching.
- Non-goals: task execution changes outside analyzer behavior.`,
	}

	result := AssessTaskQuality(task)
	if result.Score < 0 || result.Score > 100 {
		t.Fatalf("expected score to be bounded within 0..100, got %d", result.Score)
	}
}

func fullyPopulatedTaskQualityInput() TaskInput {
	return TaskInput{
		Title:       "Implement acceptance criteria coverage tracking",
		Description: "Add deterministic acceptance criteria coverage reporting for task quality analysis.",
		AcceptanceCriteria: "- Given a task has acceptance criteria, when coverage is computed, then each criterion is accounted for.",
		Deliverables: `- internal/task_quality/analyzer.go
	- Unit tests for acceptance coverage`,
		TestingPlan: `- go test ./internal/task_quality -run TestAssessTaskQuality`,
		DefinitionOfDone: `- Tests demonstrate coverage gap detection and full-coverage pass path.
- Issue comments reference any remaining quality gaps.`,
		DependenciesContext: `- Dependencies: task quality rubric and acceptance criteria parser.
- Risks: missing criteria formatting may produce false gaps.
- Non-goals: execution changes outside analyzer behavior.`,
	}
}

func joinIssues(issues []string) string {
	return strings.Join(issues, "\n")
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
