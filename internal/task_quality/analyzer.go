package taskquality

import (
	"strings"
)

type TaskInput struct {
	Title               string
	Description         string
	AcceptanceCriteria  string
	Deliverables        string
	TestingPlan         string
	DefinitionOfDone    string
	DependenciesContext string
}

type Assessment struct {
	Score  int
	Issues []string
}

func AssessTaskQuality(task TaskInput) Assessment {
	var issues []string
	score := 0

	score += requiredFieldScore(task, &issues)
	score += clarityScore(task, &issues)
	score += acceptanceCoverageScore(task, &issues)
	score += validationRigorScore(task, &issues)
	score += nonGoalRiskScore(task, &issues)

	return Assessment{
		Score:  score,
		Issues: issues,
	}
}

func requiredFieldScore(task TaskInput, issues *[]string) int {
	required := []struct {
		label string
		value string
	}{
		{label: "Title", value: task.Title},
		{label: "Description", value: task.Description},
		{label: "Acceptance Criteria", value: task.AcceptanceCriteria},
		{label: "Deliverables", value: task.Deliverables},
		{label: "Testing Plan", value: task.TestingPlan},
		{label: "Definition of Done", value: task.DefinitionOfDone},
		{label: "Dependencies/Context", value: task.DependenciesContext},
	}

	present := 0
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			*issues = append(*issues, "missing required field: "+item.label)
			continue
		}
		present++
	}

	score := present * 3
	if present == len(required) {
		score += 2
	}
	return score
}

func clarityScore(task TaskInput, issues *[]string) int {
	passed := 0

	titleDesc := strings.ToLower(task.Title + " " + task.Description)
	if !containsAny(titleDesc, []string{"improve", "more robust", "handle all", "make better", "robustness"}) {
		passed++
	} else {
		*issues = append(*issues, "clarity: avoid vague wording")
	}

	if hasGivenWhenThen(task.AcceptanceCriteria) {
		passed++
	} else {
		*issues = append(*issues, "clarity: acceptance criteria lacks testable Given/When/Then statements")
	}

	if hasConcreteScope(task.Description, task.Deliverables) {
		passed++
	} else {
		*issues = append(*issues, "clarity: task scope should be explicit and testable")
	}

	if !containsAny(strings.ToLower(task.Description), []string{"best effort", "as needed", "as soon as possible"}) {
		passed++
	} else {
		*issues = append(*issues, "clarity: deterministic wording is required")
	}

	if hasDependencySignal(task.DependenciesContext) {
		passed++
	} else {
		*issues = append(*issues, "clarity: mention dependencies and context")
	}

	testingPlan := strings.ToLower(task.TestingPlan)
	if strings.TrimSpace(task.TestingPlan) != "" && (strings.Contains(testingPlan, "go test") || strings.Contains(testingPlan, "test")) {
		passed++
	} else {
		*issues = append(*issues, "clarity: include test-first intent in testing plan")
	}

	score := passed * 3
	if passed == 6 {
		score += 2
	}
	return score
}

func acceptanceCoverageScore(task TaskInput, issues *[]string) int {
	trimmed := strings.TrimSpace(task.AcceptanceCriteria)
	if trimmed == "" {
		*issues = append(*issues, "acceptance criteria should exist and be explicit")
		return 0
	}

	lines := 0
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}

	score := 0
	if lines > 0 {
		score += 10
	}

	if hasGivenWhenThen(task.AcceptanceCriteria) {
		score += 10
	} else {
		*issues = append(*issues, "acceptance criteria should use Given/When/Then format")
	}

	return score
}

func validationRigorScore(task TaskInput, issues *[]string) int {
	plan := strings.ToLower(strings.TrimSpace(task.TestingPlan))
	if plan == "" {
		*issues = append(*issues, "validation rigor: testing plan missing")
		return 0
	}

	score := 10
	if hasConcreteTestCommand(plan) {
		score += 10
	} else {
		*issues = append(*issues, "validation rigor: testing plan needs concrete command examples")
	}
	return score
}

func nonGoalRiskScore(task TaskInput, issues *[]string) int {
	context := strings.ToLower(task.DependenciesContext)
	if strings.Contains(context, "non-goals") ||
		strings.Contains(context, "risks") ||
		strings.Contains(context, "assumption") ||
		strings.Contains(context, "blocker") {
		return 20
	}
	*issues = append(*issues, "non-goals or risks should be explicitly documented")
	return 0
}

func hasGivenWhenThen(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "given") && strings.Contains(lower, "when") && strings.Contains(lower, "then")
}

func hasConcreteScope(description string, deliverables string) bool {
	combined := strings.ToLower(description + " " + deliverables)
	return strings.Contains(combined, "internal/") ||
		strings.Contains(combined, ".go") ||
		strings.Contains(combined, "file:") ||
		strings.Contains(combined, "/")
}

func hasDependencySignal(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "depends on") ||
		strings.Contains(lower, "dependency") ||
		strings.Contains(lower, "assumption") ||
		strings.Contains(lower, "service") ||
		strings.Contains(lower, "tool")
}

func hasConcreteTestCommand(text string) bool {
	return strings.Contains(text, "go test") ||
		strings.Contains(text, "make ") ||
		strings.Contains(text, "pytest") ||
		strings.Contains(text, "npm test") ||
		strings.Contains(text, "go run")
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
