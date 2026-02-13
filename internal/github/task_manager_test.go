package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anomalyco/yolo-runner/internal/contracts"
)

func TestNewTaskManagerRequiresOwner(t *testing.T) {
	_, err := NewTaskManager(Config{Repo: "yolo-runner", Token: "ghp_test"})
	if err == nil {
		t.Fatalf("expected missing owner to fail")
	}
	if !strings.Contains(err.Error(), "owner") {
		t.Fatalf("expected owner validation error, got %q", err.Error())
	}
}

func TestNewTaskManagerRequiresRepo(t *testing.T) {
	_, err := NewTaskManager(Config{Owner: "anomalyco", Token: "ghp_test"})
	if err == nil {
		t.Fatalf("expected missing repository to fail")
	}
	if !strings.Contains(err.Error(), "repository") {
		t.Fatalf("expected repository validation error, got %q", err.Error())
	}
}

func TestNewTaskManagerRequiresToken(t *testing.T) {
	_, err := NewTaskManager(Config{Owner: "anomalyco", Repo: "yolo-runner"})
	if err == nil {
		t.Fatalf("expected missing token to fail")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Fatalf("expected token validation error, got %q", err.Error())
	}
}

func TestNewTaskManagerProbesConfiguredRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/anomalyco/yolo-runner" {
			t.Fatalf("expected probe path /repos/anomalyco/yolo-runner, got %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ghp_test" {
			t.Fatalf("expected bearer authorization, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"full_name":"anomalyco/yolo-runner"}`))
	}))
	t.Cleanup(server.Close)

	manager, err := NewTaskManager(Config{
		Owner:       "anomalyco",
		Repo:        "yolo-runner",
		Token:       "ghp_test",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("expected valid auth probe, got %v", err)
	}
	if manager == nil {
		t.Fatalf("expected non-nil task manager")
	}
}

func TestNewTaskManagerWrapsProbeAuthErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	t.Cleanup(server.Close)

	_, err := NewTaskManager(Config{
		Owner:       "anomalyco",
		Repo:        "yolo-runner",
		Token:       "ghp_invalid",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	})
	if err == nil {
		t.Fatalf("expected auth probe failure")
	}
	if !strings.Contains(err.Error(), "github auth validation failed") {
		t.Fatalf("expected wrapped auth failure, got %q", err.Error())
	}
	if !strings.Contains(strings.ToLower(err.Error()), "bad credentials") {
		t.Fatalf("expected probe details to be preserved, got %q", err.Error())
	}
}

func TestTaskManagerNextTasksFiltersUnsatisfiedDependenciesAndSortsByPriority(t *testing.T) {
	t.Parallel()

	manager := newGitHubTestManager(t, func(t *testing.T, r *http.Request, w http.ResponseWriter) {
		t.Helper()
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/repos/anomalyco/yolo-runner/issues" {
			t.Fatalf("expected issues path, got %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ghp_test" {
			t.Fatalf("expected bearer authorization, got %q", got)
		}
		if got := r.URL.Query().Get("state"); got != "all" {
			t.Fatalf("expected state=all, got %q", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("expected per_page=100, got %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("expected page=1, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
  {"number":1,"title":"Roadmap","body":"Root issue","state":"open","labels":[{"name":"p1"}]},
  {"number":2,"title":"Dependency done","body":"","state":"closed","labels":[]},
  {"number":3,"title":"Ready high","body":"","state":"open","labels":[{"name":"p0"}]},
  {"number":4,"title":"Ready low","body":"","state":"open","labels":[{"name":"priority:2"},{"name":"depends-on:#2"}]},
  {"number":5,"title":"Blocked task","body":"blocked-by: #1","state":"open","labels":[]},
  {"number":40,"title":"Automation PR","body":"","state":"open","labels":[{"name":"p0"}],"pull_request":{"url":"https://api.github.com/repos/anomalyco/yolo-runner/pulls/40"}}
]`))
	})

	tasks, err := manager.NextTasks(context.Background(), "1")
	if err != nil {
		t.Fatalf("NextTasks returned error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 runnable tasks, got %#v", tasks)
	}
	if tasks[0].ID != "3" || tasks[0].Title != "Ready high" {
		t.Fatalf("expected first task 3, got %#v", tasks[0])
	}
	if tasks[0].Priority == nil || *tasks[0].Priority != 0 {
		t.Fatalf("expected first task priority 0, got %#v", tasks[0].Priority)
	}
	if tasks[1].ID != "4" || tasks[1].Title != "Ready low" {
		t.Fatalf("expected second task 4, got %#v", tasks[1])
	}
	if tasks[1].Priority == nil || *tasks[1].Priority != 2 {
		t.Fatalf("expected second task priority 2, got %#v", tasks[1].Priority)
	}
}

func TestTaskManagerNextTasksReturnsOpenLeafParentIssueWhenNoChildren(t *testing.T) {
	t.Parallel()

	manager := newGitHubTestManager(t, func(t *testing.T, r *http.Request, w http.ResponseWriter) {
		t.Helper()
		if r.URL.Path != "/repos/anomalyco/yolo-runner/issues" {
			t.Fatalf("expected issues path, got %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
  {"number":99,"title":"Leaf issue","body":"single issue root","state":"open","labels":[{"name":"priority:1"}]}
]`))
	})

	tasks, err := manager.NextTasks(context.Background(), "99")
	if err != nil {
		t.Fatalf("NextTasks returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one leaf task, got %#v", tasks)
	}
	if tasks[0].ID != "99" || tasks[0].Title != "Leaf issue" {
		t.Fatalf("unexpected leaf summary: %#v", tasks[0])
	}
	if tasks[0].Priority == nil || *tasks[0].Priority != 1 {
		t.Fatalf("expected normalized priority 1, got %#v", tasks[0].Priority)
	}
}

func TestTaskManagerGetTaskMapsIssueDetailsAndDependencyMetadata(t *testing.T) {
	t.Parallel()

	manager := newGitHubTestManager(t, func(t *testing.T, r *http.Request, w http.ResponseWriter) {
		t.Helper()
		switch r.URL.Path {
		case "/repos/anomalyco/yolo-runner/issues":
			if got := r.URL.Query().Get("state"); got != "all" {
				t.Fatalf("expected state=all for dependency lookup, got %q", got)
			}
			_, _ = w.Write([]byte(`[
  {"number":1,"title":"Root","body":"","state":"open","labels":[]},
  {"number":2,"title":"Dep A","body":"","state":"closed","labels":[]},
  {"number":3,"title":"Dep B","body":"","state":"open","labels":[]},
  {"number":8,"title":"Implement read path","body":"Depends on #3\n- blocked by #2\nblocked-by: #8\nblocked-by: #999","state":"open","labels":[{"name":"depends-on:#2"},{"name":"blocked-by:#3, #3, #999"}]}
]`))
		case "/repos/anomalyco/yolo-runner/issues/8":
			_, _ = w.Write([]byte(`{
  "number": 8,
  "title": "Implement read path",
  "body": "Depends on #3\n- blocked by #2\nblocked-by: #8\nblocked-by: #999",
  "state": "open",
  "labels": [{"name":"depends-on:#2"},{"name":"blocked-by:#3, #3, #999"}]
}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	})

	task, err := manager.GetTask(context.Background(), "8")
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if task.ID != "8" {
		t.Fatalf("expected task ID 8, got %q", task.ID)
	}
	if task.Title != "Implement read path" {
		t.Fatalf("expected title %q, got %q", "Implement read path", task.Title)
	}
	if task.Description != "Depends on #3\n- blocked by #2\nblocked-by: #8\nblocked-by: #999" {
		t.Fatalf("unexpected task description: %q", task.Description)
	}
	if task.Status != contracts.TaskStatusOpen {
		t.Fatalf("expected task status %q, got %q", contracts.TaskStatusOpen, task.Status)
	}
	if task.ParentID != "" {
		t.Fatalf("expected empty parent ID for GitHub issue task, got %q", task.ParentID)
	}
	if deps := task.Metadata["dependencies"]; deps != "2,3" {
		t.Fatalf("expected normalized dependency metadata %q, got %q", "2,3", deps)
	}
}

func newGitHubTestManager(t *testing.T, handler func(t *testing.T, r *http.Request, w http.ResponseWriter)) *TaskManager {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		if r.URL.Path == "/repos/anomalyco/yolo-runner" {
			_, _ = w.Write([]byte(`{"full_name":"anomalyco/yolo-runner"}`))
			return
		}
		handler(t, r, w)
	}))
	t.Cleanup(server.Close)

	manager, err := NewTaskManager(Config{
		Owner:       "anomalyco",
		Repo:        "yolo-runner",
		Token:       "ghp_test",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("build test task manager: %v", err)
	}
	return manager
}
