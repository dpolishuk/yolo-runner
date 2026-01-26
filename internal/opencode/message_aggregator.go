package opencode

import (
	"strings"

	acp "github.com/ironpark/acp-go"
)

type AgentMessageAggregator struct {
	buffer strings.Builder
}

func NewAgentMessageAggregator() *AgentMessageAggregator {
	return &AgentMessageAggregator{}
}

func (a *AgentMessageAggregator) ProcessUpdate(update *acp.SessionUpdate) string {
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
		return a.processMessageChunk(&message.Content)
	}
	if message := update.GetUsermessagechunk(); message != nil {
		return formatMessage("user_message", &message.Content)
	}
	if thought := update.GetAgentthoughtchunk(); thought != nil {
		return formatMessage("agent_thought", &thought.Content)
	}
	if plan := update.GetPlan(); plan != nil {
		return ""
	}
	if commands := update.GetAvailablecommandsupdate(); commands != nil {
		return ""
	}
	if mode := update.GetCurrentmodeupdate(); mode != nil {
		return ""
	}
	return ""
}

func (a *AgentMessageAggregator) processMessageChunk(content *acp.ContentBlock) string {
	if content == nil {
		return ""
	}

	text := ""
	if content.IsText() {
		text = content.GetText().Text
	}
	if text == "" {
		return ""
	}

	a.buffer.WriteString(text)

	// Check if the accumulated content contains a newline
	accumulated := a.buffer.String()
	if strings.Contains(accumulated, "\n") {
		// Find the last newline and output everything up to and including it
		lastNewlineIndex := strings.LastIndex(accumulated, "\n")
		output := accumulated[:lastNewlineIndex+1]

		// Keep any remaining content after the last newline in the buffer
		remaining := accumulated[lastNewlineIndex+1:]
		a.buffer.Reset()
		a.buffer.WriteString(remaining)

		contentBlock := acp.NewContentBlockText(output)
		return formatMessage("agent_message", &contentBlock)
	}

	// No newline yet, don't output anything
	return ""
}
