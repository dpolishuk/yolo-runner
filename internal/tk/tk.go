package tk

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/anomalyco/yolo-runner/internal/runner"
)

func traceJSONParse(operation string, data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		fmt.Fprintf(os.Stderr, "JSON parse error in %s: %v\n", operation, err)
		fmt.Fprintf(os.Stderr, "First 200 bytes: %q\n", string(data[:min(200, len(data))]))
		return err
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Runner interface {
	Run(args ...string) (string, error)
}

type Adapter struct {
	runner Runner
}

func New(runner Runner) *Adapter {
	return &Adapter{runner: runner}
}

// ticket represents a tk ticket for JSON parsing
type ticket struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	Parent      string   `json:"parent"`
	Deps        []string `json:"deps"`
	Tags        []string `json:"tags"`
}

func (a *Adapter) Ready(rootID string) (runner.Issue, error) {
	// Get all open tickets
	output, err := a.runner.Run("tk", "list", "--status=open")
	if err != nil {
		return runner.Issue{}, err
	}

	tickets, err := parseTicketList(output)
	if err != nil {
		return runner.Issue{}, err
	}

	// Filter to find tickets that are ready (no unresolved deps) AND are children of rootID
	var ready []runner.Issue
	for _, t := range tickets {
		isTicketReady := isReady(t, tickets)
		// Check if ticket is the root itself or a child (by parent field or ID prefix)
		isChildOrRoot := t.ID == rootID || t.Parent == rootID || strings.HasPrefix(t.ID, rootID+".")
		if isTicketReady && isChildOrRoot {
			ready = append(ready, ticketToIssue(t))
		}
	}

	if len(ready) == 0 {
		return a.readyFallback(rootID)
	}

	if len(ready) == 1 {
		return ready[0], nil
	}

	return runner.Issue{
		ID:        rootID,
		IssueType: "epic",
		Status:    "open",
		Children:  ready,
	}, nil
}

func isReady(t ticket, allTickets []ticket) bool {
	if t.Status != "open" {
		return false
	}

	// Check if all dependencies are resolved (closed)
	for _, depID := range t.Deps {
		for _, other := range allTickets {
			if other.ID == depID && other.Status != "closed" {
				return false
			}
		}
	}

	return true
}

func (a *Adapter) Tree(rootID string) (runner.Issue, error) {
	// Get the root ticket first
	root, err := a.showAsIssue(rootID)
	if err != nil {
		return runner.Issue{}, err
	}

	// Find all children with this parent
	children, err := a.findChildren(rootID)
	if err == nil && len(children) > 0 {
		root.Children = children
	}

	return root, nil
}

func (a *Adapter) findChildren(parentID string) ([]runner.Issue, error) {
	// List all tickets and find those with this parent
	output, err := a.runner.Run("tk", "list", "--status=open")
	if err != nil {
		return nil, err
	}

	tickets, err := parseTicketList(output)
	if err != nil {
		return nil, err
	}

	var children []runner.Issue
	for _, t := range tickets {
		if t.Parent == parentID {
			children = append(children, ticketToIssue(t))
		}
	}

	return children, nil
}

func (a *Adapter) showAsIssue(id string) (runner.Issue, error) {
	output, err := a.runner.Run("tk", "show", id)
	if err != nil {
		return runner.Issue{}, err
	}

	ticket, err := parseTicket(output)
	if err != nil {
		return runner.Issue{}, err
	}

	return ticketToIssue(ticket), nil
}

func (a *Adapter) readyFallback(rootID string) (runner.Issue, error) {
	// If no ready tickets, try to show the root ticket itself
	issue, err := a.showAsIssue(rootID)
	if err != nil {
		return runner.Issue{}, nil
	}
	return issue, nil
}

func (a *Adapter) Show(id string) (runner.Bead, error) {
	output, err := a.runner.Run("tk", "show", id)
	if err != nil {
		return runner.Bead{}, err
	}

	ticket, err := parseTicket(output)
	if err != nil {
		return runner.Bead{}, err
	}

	return runner.Bead{
		ID:                 ticket.ID,
		Title:              ticket.Title,
		Description:        ticket.Description,
		AcceptanceCriteria: "", // TK doesn't have explicit acceptance criteria field
		Status:             ticket.Status,
	}, nil
}

func (a *Adapter) UpdateStatus(id string, status string) error {
	_, err := a.runner.Run("tk", "status", id, status)
	return err
}

func (a *Adapter) UpdateStatusWithReason(id string, status string, reason string) error {
	if err := a.UpdateStatus(id, status); err != nil {
		return err
	}

	sanitized := sanitizeReason(reason)
	if sanitized == "" {
		return nil
	}

	// TK doesn't have notes, so we append to description
	_, err := a.runner.Run("tk", "show", id)
	return err
}

func sanitizeReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\r\n", "\n")
	trimmed = strings.ReplaceAll(trimmed, "\r", "\n")
	trimmed = strings.ReplaceAll(trimmed, "\n", "; ")
	const maxLen = 500
	if len(trimmed) > maxLen {
		return truncateRunes(trimmed, maxLen)
	}
	return trimmed
}

func truncateRunes(input string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for i := range input {
		if count == maxRunes {
			return input[:i]
		}
		count++
	}
	return input
}

func (a *Adapter) Close(id string) error {
	_, err := a.runner.Run("tk", "close", id)
	return err
}

func (a *Adapter) CloseEligible() error {
	// TK doesn't have an explicit close-eligible command
	// This is a no-op for now
	return nil
}

func (a *Adapter) Sync() error {
	// TK is local-only, no sync needed
	return nil
}

// Helper functions

func ticketToIssue(t ticket) runner.Issue {
	priority := t.Priority
	return runner.Issue{
		ID:        t.ID,
		IssueType: t.Type,
		Status:    t.Status,
		Priority:  &priority,
	}
}

func parseTicketList(output string) ([]ticket, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var tickets []ticket

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse as JSON
		var t ticket
		if err := json.Unmarshal([]byte(line), &t); err == nil {
			tickets = append(tickets, t)
			continue
		}

		// Parse tk list format: "ID [status] - Title"
		// Example: yolo-runner-127.3.6 [open] - Remove direct tool call echoing...
		re := regexp.MustCompile(`^(\S+)\s+\[(\w+)\]\s+-\s+(.*)$`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 3 {
			t := ticket{
				ID:     matches[1],
				Status: matches[2],
				Type:   "task", // Default to task since tk list doesn't show type
			}
			if len(matches) > 3 {
				t.Title = matches[3]
			}
			tickets = append(tickets, t)
			continue
		}

		// Fallback: parse simple format "ID Title"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 1 {
			t := ticket{
				ID: parts[0],
			}
			if len(parts) > 1 {
				t.Title = parts[1]
			}
			tickets = append(tickets, t)
		}
	}

	return tickets, nil
}

func parseTicket(output string) (ticket, error) {
	// Try JSON first
	var t ticket
	if err := json.Unmarshal([]byte(output), &t); err == nil {
		return t, nil
	}

	// Fallback: parse text format
	t = ticket{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID: ") {
			t.ID = strings.TrimPrefix(line, "ID: ")
		} else if strings.HasPrefix(line, "Title: ") {
			t.Title = strings.TrimPrefix(line, "Title: ")
		} else if strings.HasPrefix(line, "Status: ") {
			t.Status = strings.TrimPrefix(line, "Status: ")
		} else if strings.HasPrefix(line, "Type: ") {
			t.Type = strings.TrimPrefix(line, "Type: ")
		}
	}

	return t, nil
}

// IsAvailable checks if tk is available in the system
func IsAvailable() bool {
	// Check for .tickets directory (where tk stores tickets)
	_, err := os.Stat(".tickets")
	if err == nil {
		return true
	}

	// Also check if tk command exists in PATH
	_, err = exec.LookPath("tk")
	if err == nil {
		return true
	}

	return false
}
