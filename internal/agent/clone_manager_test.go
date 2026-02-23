package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGitCloneManagerClonesRepoPerTaskAndCleansUp(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	readmePath := filepath.Join(repoRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "init")

	manager := NewGitCloneManager(t.TempDir())
	clonePath, err := manager.CloneForTask(context.Background(), "t-1", repoRoot)
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}
	if clonePath == repoRoot {
		t.Fatalf("expected isolated clone path, got source path %q", clonePath)
	}
	if _, err := os.Stat(filepath.Join(clonePath, ".git")); err != nil {
		t.Fatalf("expected git metadata in clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clonePath, "README.md")); err != nil {
		t.Fatalf("expected tracked file in clone: %v", err)
	}

	if err := manager.Cleanup("t-1"); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		t.Fatalf("expected clone path removed, got err=%v", err)
	}
}

func TestGitCloneManagerSetsCloneOriginToSourceUpstream(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}

	remoteRoot := t.TempDir()
	remotePath := filepath.Join(remoteRoot, "remote.git")
	runGit(t, remoteRoot, "init", "--bare", remotePath)

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	readmePath := filepath.Join(repoRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "init")
	runGit(t, repoRoot, "remote", "add", "origin", remotePath)

	manager := NewGitCloneManager(t.TempDir())
	clonePath, err := manager.CloneForTask(context.Background(), "t-remote", repoRoot)
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}
	defer func() { _ = manager.Cleanup("t-remote") }()

	originURL := strings.TrimSpace(runGitOutput(t, clonePath, "remote", "get-url", "origin"))
	if originURL != remotePath {
		t.Fatalf("expected clone origin=%q, got %q", remotePath, originURL)
	}
}

func TestGitCloneManagerCreatesIsolatedParallelClones(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	readmePath := filepath.Join(repoRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "init")

	baseDir := t.TempDir()
	manager := NewGitCloneManager(baseDir)

	type cloneResult struct {
		taskID string
		path   string
		err    error
	}

	const taskCount = 4
	results := make(chan cloneResult, taskCount)
	var wg sync.WaitGroup
	for i := 1; i <= taskCount; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("task-%d", i)
		go func(taskID string) {
			defer wg.Done()
			path, err := manager.CloneForTask(context.Background(), taskID, repoRoot)
			results <- cloneResult{taskID: taskID, path: path, err: err}
		}(taskID)
	}
	wg.Wait()
	close(results)

	clonePaths := map[string]string{}
	for result := range results {
		if result.err != nil {
			t.Fatalf("parallel clone failed: %v", result.err)
		}
		if result.path == repoRoot {
			t.Fatalf("expected isolated clone path, got source path %q", result.path)
		}
		if _, ok := clonePaths[result.path]; ok {
			t.Fatalf("expected unique clone path per task, got shared path %q", result.path)
		}
		clonePaths[result.path] = result.taskID

		if _, err := os.Stat(filepath.Join(result.path, ".git")); err != nil {
			t.Fatalf("expected git metadata in clone for task %q: %v", result.taskID, err)
		}
	}
	if len(clonePaths) != taskCount {
		t.Fatalf("expected %d clone paths, got %d", taskCount, len(clonePaths))
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("read clone base dir: %v", err)
	}
	if len(entries) != taskCount {
		t.Fatalf("expected %d clone directories, got %d", taskCount, len(entries))
	}

	for _, taskID := range []string{"task-1", "task-2", "task-3", "task-4"} {
		if err := manager.Cleanup(taskID); err != nil {
			t.Fatalf("cleanup for %q failed: %v", taskID, err)
		}
	}
	for path := range clonePaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected clone path removed for %q, got err=%v", path, err)
		}
	}
	entries, err = os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("read clone base dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected clone base dir to be empty after cleanup, got %d entries", len(entries))
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
	return string(out)
}
