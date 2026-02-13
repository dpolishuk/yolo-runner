package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anomalyco/yolo-runner/internal/contracts"
)

const (
	defaultAPIEndpoint   = "https://api.github.com"
	maxProbeResponseSize = 1 << 20
	maxReadResponseSize  = 8 << 20
	issuesPerPage        = 100
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	Owner       string
	Repo        string
	Token       string
	APIEndpoint string
	HTTPClient  HTTPClient
}

type TaskManager struct {
	owner       string
	repo        string
	token       string
	apiEndpoint string
	client      HTTPClient
}

type githubIssuePayload struct {
	Number      int                  `json:"number"`
	Title       string               `json:"title"`
	Body        string               `json:"body"`
	State       string               `json:"state"`
	Labels      []githubLabelPayload `json:"labels"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type githubLabelPayload struct {
	Name string `json:"name"`
}

func NewTaskManager(cfg Config) (*TaskManager, error) {
	owner := strings.TrimSpace(cfg.Owner)
	if owner == "" {
		return nil, errors.New("github owner is required")
	}

	repo := strings.TrimSpace(cfg.Repo)
	if repo == "" {
		return nil, errors.New("github repository is required")
	}

	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		return nil, errors.New("github auth token is required")
	}

	endpoint := strings.TrimSpace(cfg.APIEndpoint)
	if endpoint == "" {
		endpoint = defaultAPIEndpoint
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	if err := probeRepository(context.Background(), client, endpoint, owner, repo, token); err != nil {
		return nil, fmt.Errorf("github auth validation failed: %w", err)
	}

	return &TaskManager{
		owner:       owner,
		repo:        repo,
		token:       token,
		apiEndpoint: endpoint,
		client:      client,
	}, nil
}

func (m *TaskManager) NextTasks(ctx context.Context, parentID string) ([]contracts.TaskSummary, error) {
	rootNumber, err := parseIssueNumber(parentID, "parent task ID")
	if err != nil {
		return nil, err
	}

	issues, err := m.fetchRepositoryIssues(ctx)
	if err != nil {
		return nil, err
	}

	graph, statusByID, err := buildTaskGraph(rootNumber, issues)
	if err != nil {
		return nil, err
	}

	rootID := strconv.Itoa(rootNumber)
	children := graph.ChildrenOf(rootID)
	tasks := make([]contracts.TaskSummary, 0, len(children))
	for _, child := range children {
		if child.Kind != NodeKindIssue {
			continue
		}
		if statusByID[child.ID] != contracts.TaskStatusOpen {
			continue
		}
		if !dependenciesClosed(child.Dependencies, statusByID) {
			continue
		}
		tasks = append(tasks, taskSummaryFromNode(child))
	}
	if len(tasks) > 0 {
		return tasks, nil
	}

	parent, ok := graph.Nodes[rootID]
	if !ok {
		return nil, nil
	}
	if statusByID[rootID] != contracts.TaskStatusOpen {
		return nil, nil
	}
	if !dependenciesClosed(parent.Dependencies, statusByID) {
		return nil, nil
	}
	return []contracts.TaskSummary{taskSummaryFromNode(parent)}, nil
}

func (m *TaskManager) GetTask(ctx context.Context, taskID string) (contracts.Task, error) {
	issueNumber, err := parseIssueNumber(taskID, "task ID")
	if err != nil {
		return contracts.Task{}, err
	}

	issue, err := m.fetchIssue(ctx, issueNumber)
	if err != nil {
		return contracts.Task{}, err
	}
	if issue == nil {
		return contracts.Task{}, nil
	}

	issues, err := m.fetchRepositoryIssues(ctx)
	if err != nil {
		return contracts.Task{}, err
	}

	metadata := map[string]string{}
	if deps := dependencyIDsForIssue(*issue, issues); len(deps) > 0 {
		metadata["dependencies"] = strings.Join(deps, ",")
	}
	if len(metadata) == 0 {
		metadata = nil
	}

	id := strconv.Itoa(issue.Number)
	return contracts.Task{
		ID:          id,
		Title:       fallbackText(issue.Title, id),
		Description: issue.Body,
		Status:      taskStatusFromIssueState(issue.State),
		Metadata:    metadata,
	}, nil
}

func (m *TaskManager) SetTaskStatus(context.Context, string, contracts.TaskStatus) error {
	return errors.New("github SetTaskStatus is not implemented yet")
}

func (m *TaskManager) SetTaskData(context.Context, string, map[string]string) error {
	return errors.New("github SetTaskData is not implemented yet")
}

func (m *TaskManager) fetchRepositoryIssues(ctx context.Context) ([]githubIssuePayload, error) {
	issues := []githubIssuePayload{}
	for page := 1; ; page++ {
		requestURL := strings.TrimRight(m.apiEndpoint, "/") + "/repos/" + url.PathEscape(m.owner) + "/" + url.PathEscape(m.repo) + "/issues?state=all&per_page=" + strconv.Itoa(issuesPerPage) + "&page=" + strconv.Itoa(page)
		statusCode, body, err := m.doGitHubGET(ctx, requestURL, maxReadResponseSize)
		if err != nil {
			return nil, fmt.Errorf("query GitHub issues page %d: %w", page, err)
		}
		if statusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("query GitHub issues page %d: request failed with status %d: %s", page, statusCode, firstAPIError(body))
		}

		pageIssues := []githubIssuePayload{}
		if strings.TrimSpace(string(body)) != "" {
			if err := json.Unmarshal(body, &pageIssues); err != nil {
				return nil, fmt.Errorf("query GitHub issues page %d: cannot parse response: %w", page, err)
			}
		}

		for _, issue := range pageIssues {
			if issue.PullRequest != nil {
				continue
			}
			issues = append(issues, issue)
		}

		if len(pageIssues) < issuesPerPage {
			break
		}
	}
	return issues, nil
}

func (m *TaskManager) fetchIssue(ctx context.Context, issueNumber int) (*githubIssuePayload, error) {
	requestURL := strings.TrimRight(m.apiEndpoint, "/") + "/repos/" + url.PathEscape(m.owner) + "/" + url.PathEscape(m.repo) + "/issues/" + strconv.Itoa(issueNumber)
	statusCode, body, err := m.doGitHubGET(ctx, requestURL, maxReadResponseSize)
	if err != nil {
		return nil, fmt.Errorf("query GitHub issue %d: %w", issueNumber, err)
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("query GitHub issue %d: request failed with status %d: %s", issueNumber, statusCode, firstAPIError(body))
	}
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}

	var issue githubIssuePayload
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("query GitHub issue %d: cannot parse response: %w", issueNumber, err)
	}
	if issue.PullRequest != nil {
		return nil, nil
	}
	if issue.Number <= 0 {
		issue.Number = issueNumber
	}
	return &issue, nil
}

func (m *TaskManager) doGitHubGET(ctx context.Context, requestURL string, maxBodyBytes int64) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("cannot read response: %w", err)
	}
	return resp.StatusCode, body, nil
}

func buildTaskGraph(rootNumber int, issues []githubIssuePayload) (TaskGraph, map[string]contracts.TaskStatus, error) {
	mapped := make([]Issue, 0, len(issues))
	statusByID := make(map[string]contracts.TaskStatus, len(issues))

	for _, issue := range issues {
		if issue.Number <= 0 {
			return TaskGraph{}, nil, fmt.Errorf("GitHub issue number must be positive")
		}
		labels := labelNames(issue.Labels)
		mapped = append(mapped, Issue{
			Number: issue.Number,
			Title:  issue.Title,
			Body:   issue.Body,
			Labels: labels,
		})
		statusByID[strconv.Itoa(issue.Number)] = taskStatusFromIssueState(issue.State)
	}

	graph, err := MapToTaskGraph(rootNumber, mapped)
	if err != nil {
		return TaskGraph{}, nil, fmt.Errorf("map GitHub issues to task graph: %w", err)
	}
	return graph, statusByID, nil
}

func dependencyIDsForIssue(issue githubIssuePayload, inScope []githubIssuePayload) []string {
	issueID := strconv.Itoa(issue.Number)
	validIDs := make(map[string]struct{}, len(inScope)+1)
	for _, candidate := range inScope {
		if candidate.Number <= 0 {
			continue
		}
		validIDs[strconv.Itoa(candidate.Number)] = struct{}{}
	}
	validIDs[issueID] = struct{}{}

	references := dependencyReferences(Issue{
		Number: issue.Number,
		Title:  issue.Title,
		Body:   issue.Body,
		Labels: labelNames(issue.Labels),
	})
	return normalizeDependencies(issueID, references, validIDs)
}

func labelNames(labels []githubLabelPayload) []string {
	if len(labels) == 0 {
		return nil
	}

	names := make([]string, 0, len(labels))
	for _, label := range labels {
		name := strings.TrimSpace(label.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func parseIssueNumber(raw string, fieldName string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", fieldName, value)
	}
	return number, nil
}

func dependenciesClosed(dependencies []string, statusByID map[string]contracts.TaskStatus) bool {
	for _, depID := range dependencies {
		status, ok := statusByID[depID]
		if !ok {
			continue
		}
		if status != contracts.TaskStatusClosed {
			return false
		}
	}
	return true
}

func taskSummaryFromNode(node Node) contracts.TaskSummary {
	priority := node.Priority
	return contracts.TaskSummary{
		ID:       node.ID,
		Title:    node.Title,
		Priority: &priority,
	}
}

func taskStatusFromIssueState(state string) contracts.TaskStatus {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "closed":
		return contracts.TaskStatusClosed
	default:
		return contracts.TaskStatusOpen
	}
}

func firstAPIError(body []byte) string {
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return "unknown error"
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Message) != "" {
		return strings.TrimSpace(payload.Message)
	}
	return bodyText
}

func probeRepository(ctx context.Context, client HTTPClient, endpoint string, owner string, repo string, token string) error {
	requestURL := strings.TrimRight(endpoint, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("cannot build probe request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("probe request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProbeResponseSize))
	if err != nil {
		return fmt.Errorf("cannot read probe response: %w", err)
	}

	var probe struct {
		FullName string `json:"full_name"`
		Message  string `json:"message"`
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &probe); err != nil {
			if resp.StatusCode >= http.StatusBadRequest {
				return fmt.Errorf("probe failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			return fmt.Errorf("cannot parse probe response: %w", err)
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("probe failed with status %d: %s", resp.StatusCode, firstProbeError(probe.Message, strings.TrimSpace(string(body))))
	}
	if strings.TrimSpace(probe.FullName) == "" {
		return errors.New("probe failed: repository identity missing in response")
	}

	expected := strings.ToLower(owner + "/" + repo)
	if strings.ToLower(strings.TrimSpace(probe.FullName)) != expected {
		return fmt.Errorf("probe failed: expected repository %q, got %q", expected, strings.TrimSpace(probe.FullName))
	}

	return nil
}

func firstProbeError(message string, fallback string) string {
	if strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "unknown error"
}
