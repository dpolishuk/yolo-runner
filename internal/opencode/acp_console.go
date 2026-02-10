package opencode

import (
	"fmt"
	"strings"

	acp "github.com/ironpark/acp-go"
)

const acpConsoleSnippetLimit = 120

func formatACPRequest(requestType string, decision string) string {
	return formatACPRequestDetail(requestType, decision, "")
}

func formatACPRequestDetail(requestType string, decision string, detail string) string {
	if requestType == "" && decision == "" {
		return ""
	}
	detail = normalizeACPRequestDetail(detail)
	detailText := ""
	if detail != "" {
		detailText = fmt.Sprintf(" detail=%q", detail)
	}
	if requestType == "" {
		return fmt.Sprintf("request %s%s", decision, detailText)
	}
	if decision == "" {
		return fmt.Sprintf("request %s%s", requestType, detailText)
	}
	return fmt.Sprintf("request %s %s%s", requestType, decision, detailText)
}

func normalizeACPRequestDetail(detail string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(detail, "\n", " "), "\r", " "))
	if trimmed == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(trimmed), " ")
	const maxLen = 160
	if len(normalized) > maxLen {
		return normalized[:maxLen] + "..."
	}
	return normalized
}

func formatSessionUpdate(update *acp.SessionUpdate) string {
	if update == nil {
		return ""
	}
	if toolCall := update.GetToolcall(); toolCall != nil {
		return formatToolCall("tool_call", toolCall.ToolCallId, toolCall.Title, toolCall.Kind, toolCall.Status)
	}
	if toolUpdate := update.GetToolcallupdate(); toolUpdate != nil {
		return formatToolCall("tool_call_update", toolUpdate.ToolCallId, toolUpdate.Title, toolUpdate.Kind, toolUpdate.Status)
	}
	if message := update.GetAgentmessagechunk(); message != nil {
		return formatMessage("agent_message", &message.Content)
	}
	if message := update.GetUsermessagechunk(); message != nil {
		return formatMessage("user_message", &message.Content)
	}
	if thought := update.GetAgentthoughtchunk(); thought != nil {
		// Normalize agent thought text by stripping newlines
		normalizedText := normalizeAgentThoughtText(thought.Content.GetText().Text)
		normalizedContent := acp.NewContentBlockText(normalizedText)
		return formatMessage("agent_thought", &normalizedContent)
	}
	if plan := update.GetPlan(); plan != nil {
		return fmt.Sprintf("plan entries=%d", len(plan.Entries))
	}
	if commands := update.GetAvailablecommandsupdate(); commands != nil {
		return fmt.Sprintf("commands count=%d", len(commands.AvailableCommands))
	}
	if mode := update.GetCurrentmodeupdate(); mode != nil {
		return fmt.Sprintf("mode current=%s", mode.CurrentModeId)
	}
	return ""
}

func formatToolCall(prefix string, id acp.ToolCallId, title string, kind *acp.ToolKind, status *acp.ToolCallStatus) string {
	// Determine emoji and color based on status
	var emoji string
	var color string
	var resetColor = "\x1b[0m"

	if status == nil {
		emoji = "⚪"
		color = "\x1b[37m" // White
	} else {
		switch *status {
		case acp.ToolCallStatusPending:
			emoji = "⏳"
			color = "\x1b[33m" // Yellow
		case acp.ToolCallStatusInProgress:
			emoji = "🔄"
			color = "\x1b[34m" // Blue
		case acp.ToolCallStatusCompleted:
			emoji = "✅"
			color = "\x1b[32m" // Green
		case acp.ToolCallStatusFailed:
			emoji = "❌"
			color = "\x1b[31m" // Red
		default:
			emoji = "⚪"
			color = "\x1b[37m" // White (neutral for unknown status)
		}
	}

	// For tool_call_update, only show emoji + label + title (simplified format)
	if prefix == "tool_call_update" {
		return fmt.Sprintf("%s %s%s%s %s", emoji, color, prefix, resetColor, title)
	}

	// For tool_call, show full details (emoji + label + id + title + kind + status)
	parts := []string{fmt.Sprintf("%s %s%s%s", emoji, color, prefix, resetColor), fmt.Sprintf("id=%s", id)}
	if title != "" {
		parts = append(parts, fmt.Sprintf("title=\"%s\"", title))
	}
	if kind != nil {
		parts = append(parts, fmt.Sprintf("kind=%s", *kind))
	}
	if status != nil {
		parts = append(parts, fmt.Sprintf("status=%s", *status))
	}
	return strings.Join(parts, " ")
}

func formatMessage(prefix string, content *acp.ContentBlock) string {
	if content == nil {
		return prefix
	}
	text := ""
	if content.IsText() {
		text = content.GetText().Text
	}
	if text == "" {
		return prefix
	}
	return fmt.Sprintf("%s %q", prefix, text)
}

func truncateACPText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

// normalizeAgentThoughtText strips newlines and carriage returns from agent thought text
// and replaces them with spaces to prevent breaking TUI layout
func normalizeAgentThoughtText(text string) string {
	// Replace all newlines and carriage returns with spaces
	// This handles \n, \r, and \r\n (Windows line endings)
	text = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)
	return text
}
