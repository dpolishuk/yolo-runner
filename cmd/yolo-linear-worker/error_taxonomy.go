package main

import "strings"

type linearSessionErrorClass struct {
	category    string
	remediation string
}

var linearSessionErrorTaxonomy = []struct {
	match func(string) bool
	class linearSessionErrorClass
}{
	{match: containsLinearSessionAny("http 401", "http 403", "unauthorized", "forbidden", "authorization", "auth", "token"), class: linearSessionErrorClass{category: "auth", remediation: "Verify LINEAR_TOKEN/LINEAR_API_TOKEN and endpoint permissions, then retry the queued session."}},
	{match: containsLinearSessionAny("graphql", "agent activity mutation", "issueupdate", "query delegated issue workflow state"), class: linearSessionErrorClass{category: "graphql", remediation: "Check Linear GraphQL query/mutation payloads and endpoint health, then retry the queued session."}},
	{match: containsLinearSessionAny("run linear session job", "runner timeout", "opencode stall", "runner returned", "runner failed", "runner blocked"), class: linearSessionErrorClass{category: "runtime", remediation: "Inspect backend runner logs and the latest runner-logs artifacts, then rerun the Linear session step."}},
	{match: containsLinearSessionAny("queued linear webhook job", "decode queued job line", "unsupported queued job contract version", "missing job identifiers"), class: linearSessionErrorClass{category: "webhook", remediation: "Validate webhook payload contract fields (contractVersion, sessionId, stepId, action) and re-enqueue a valid job."}},
}

func FormatLinearSessionActionableError(err error) string {
	if err == nil {
		return ""
	}
	cause := normalizeLinearSessionCause(trimLinearSessionGenericExitStatus(err.Error()))
	class := classifyLinearSessionError(cause)
	return "Category: " + class.category + "\nCause: " + cause + "\nNext step: " + class.remediation
}

func classifyLinearSessionError(cause string) linearSessionErrorClass {
	text := strings.ToLower(cause)
	for _, entry := range linearSessionErrorTaxonomy {
		if entry.match(text) {
			return entry.class
		}
	}
	return linearSessionErrorClass{
		category:    "unknown",
		remediation: "Check worker and runner logs for details, then retry; escalate with the full error text if it persists.",
	}
}

func normalizeLinearSessionCause(cause string) string {
	parts := strings.Split(cause, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		line := strings.TrimSpace(part)
		if line == "" || isLinearSessionPureExitStatusLine(line) {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return strings.TrimSpace(cause)
	}
	return strings.Join(lines, " | ")
}

func trimLinearSessionGenericExitStatus(cause string) string {
	trimmed := strings.TrimSpace(cause)
	lower := strings.ToLower(trimmed)
	const suffix = ": exit status "

	idx := strings.LastIndex(lower, suffix)
	if idx == -1 {
		return trimmed
	}
	statusPart := strings.TrimSpace(trimmed[idx+len(suffix):])
	if statusPart == "" {
		return trimmed
	}
	for _, r := range statusPart {
		if r < '0' || r > '9' {
			return trimmed
		}
	}
	if idx == 0 {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:idx])
}

func isLinearSessionPureExitStatusLine(line string) bool {
	line = strings.TrimSpace(strings.ToLower(line))
	if !strings.HasPrefix(line, "exit status ") {
		return false
	}
	n := strings.TrimSpace(strings.TrimPrefix(line, "exit status "))
	if n == "" {
		return false
	}
	for _, r := range n {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsLinearSessionAny(parts ...string) func(string) bool {
	return func(text string) bool {
		for _, part := range parts {
			if strings.Contains(text, part) {
				return true
			}
		}
		return false
	}
}
