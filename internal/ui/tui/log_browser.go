package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LogBrowser renders a simple two-level list of available task logs with selected
// log content preview.
type LogBrowser struct {
	root              string
	taskIDs           []string
	logFilesByTaskID  map[string][]string
	selectedTaskIndex int
	selectedFileIndex int
	selectedContent   string
}

// NewLogBrowser discovers available task log files under the given root directory.
func NewLogBrowser(logRoot string) (*LogBrowser, error) {
	if strings.TrimSpace(logRoot) == "" {
		return &LogBrowser{logFilesByTaskID: map[string][]string{}}, nil
	}

	if info, err := os.Stat(logRoot); err != nil {
		if os.IsNotExist(err) {
			return &LogBrowser{logFilesByTaskID: map[string][]string{}}, nil
		}
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("log browser root is not a directory: %s", logRoot)
	}

	browser := &LogBrowser{
		root:             logRoot,
		logFilesByTaskID: map[string][]string{},
	}

	if err := filepath.WalkDir(logRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !isSupportedLogFile(name) {
			return nil
		}

		taskID := taskIDFromLogFile(name)
		if taskID == "" {
			return nil
		}
		browser.logFilesByTaskID[taskID] = append(browser.logFilesByTaskID[taskID], path)
		return nil
	}); err != nil {
		return nil, err
	}

	browser.taskIDs = make([]string, 0, len(browser.logFilesByTaskID))
	for taskID := range browser.logFilesByTaskID {
		browser.taskIDs = append(browser.taskIDs, taskID)
	}
	sort.Strings(browser.taskIDs)

	for _, taskID := range browser.taskIDs {
		files := browser.logFilesByTaskID[taskID]
		sort.SliceStable(files, func(i, j int) bool {
			return filepath.Base(files[i]) < filepath.Base(files[j])
		})
		browser.logFilesByTaskID[taskID] = files
	}

	if len(browser.taskIDs) > 0 {
		browser.selectedTaskIndex = 0
		browser.selectedFileIndex = 0
		browser.refreshSelectedLogContent()
	}

	return browser, nil
}

func isSupportedLogFile(name string) bool {
	return strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".log")
}

func taskIDFromLogFile(name string) string {
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimSuffix(name, "\\")
	if strings.HasSuffix(name, ".stderr.log") {
		return strings.TrimSuffix(name, ".stderr.log")
	}
	return strings.TrimSuffix(strings.TrimSuffix(name, ".jsonl"), ".log")
}

func (b *LogBrowser) refreshSelectedLogContent() {
	if b == nil {
		return
	}
	path := b.CurrentLogFile()
	if path == "" {
		b.selectedContent = ""
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		b.selectedContent = fmt.Sprintf("failed to read log file %s: %v", path, err)
		return
	}
	b.selectedContent = string(content)
}

func (b *LogBrowser) clampSelection() {
	if b == nil {
		return
	}
	if len(b.taskIDs) == 0 {
		b.selectedTaskIndex = -1
		b.selectedFileIndex = -1
		b.selectedContent = ""
		return
	}
	if b.selectedTaskIndex < 0 {
		b.selectedTaskIndex = 0
	}
	if b.selectedTaskIndex >= len(b.taskIDs) {
		b.selectedTaskIndex = len(b.taskIDs) - 1
	}
	files := b.logFilesByTaskID[b.taskIDs[b.selectedTaskIndex]]
	if len(files) == 0 {
		b.selectedFileIndex = -1
		b.selectedContent = ""
		return
	}
	if b.selectedFileIndex < 0 {
		b.selectedFileIndex = 0
	}
	if b.selectedFileIndex >= len(files) {
		b.selectedFileIndex = len(files) - 1
	}
}

// SelectTask sets the active task by index and refreshes the selected file/content.
func (b *LogBrowser) SelectTask(index int) {
	if b == nil {
		return
	}
	b.selectedTaskIndex = index
	b.selectedFileIndex = 0
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// SelectLogFile sets the active log file index for the selected task.
func (b *LogBrowser) SelectLogFile(index int) {
	if b == nil {
		return
	}
	b.selectedFileIndex = index
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// NextTask advances the selected task in the list, clamped to the final item.
func (b *LogBrowser) NextTask() {
	if b == nil || len(b.taskIDs) == 0 {
		return
	}
	if b.selectedTaskIndex < 0 {
		b.selectedTaskIndex = 0
		b.selectedFileIndex = 0
		b.refreshSelectedLogContent()
		return
	}
	b.selectedTaskIndex++
	b.selectedFileIndex = 0
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// PrevTask moves the selected task backward in the list.
func (b *LogBrowser) PrevTask() {
	if b == nil || len(b.taskIDs) == 0 {
		return
	}
	if b.selectedTaskIndex < 0 {
		b.selectedTaskIndex = 0
		b.selectedFileIndex = 0
		b.refreshSelectedLogContent()
		return
	}
	if b.selectedTaskIndex == 0 {
		return
	}
	b.selectedTaskIndex--
	b.selectedFileIndex = 0
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// NextLogFile selects the next log file for the active task.
func (b *LogBrowser) NextLogFile() {
	if b == nil || len(b.taskIDs) == 0 {
		return
	}
	if b.selectedTaskIndex < 0 || b.selectedTaskIndex >= len(b.taskIDs) {
		b.clampSelection()
	}
	files := b.logFilesByTaskID[b.taskIDs[b.selectedTaskIndex]]
	if len(files) == 0 {
		return
	}
	b.selectedFileIndex++
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// PrevLogFile selects the previous log file for the active task.
func (b *LogBrowser) PrevLogFile() {
	if b == nil || len(b.taskIDs) == 0 {
		return
	}
	if b.selectedTaskIndex < 0 || b.selectedTaskIndex >= len(b.taskIDs) {
		b.clampSelection()
	}
	files := b.logFilesByTaskID[b.taskIDs[b.selectedTaskIndex]]
	if len(files) == 0 {
		return
	}
	if b.selectedFileIndex > 0 {
		b.selectedFileIndex--
	}
	b.clampSelection()
	b.refreshSelectedLogContent()
}

// CurrentTask returns the currently selected task id.
func (b *LogBrowser) CurrentTask() string {
	if b == nil || len(b.taskIDs) == 0 {
		return ""
	}
	if b.selectedTaskIndex < 0 || b.selectedTaskIndex >= len(b.taskIDs) {
		return ""
	}
	return b.taskIDs[b.selectedTaskIndex]
}

// CurrentLogFile returns the absolute path for the currently selected log file.
func (b *LogBrowser) CurrentLogFile() string {
	if b == nil || len(b.taskIDs) == 0 {
		return ""
	}
	if b.selectedTaskIndex < 0 || b.selectedTaskIndex >= len(b.taskIDs) {
		return ""
	}
	files := b.logFilesByTaskID[b.taskIDs[b.selectedTaskIndex]]
	if b.selectedFileIndex < 0 || b.selectedFileIndex >= len(files) {
		return ""
	}
	return files[b.selectedFileIndex]
}

// View renders the current task list, file list, and selected log file content.
func (b *LogBrowser) View() string {
	if b == nil {
		return ""
	}
	if len(b.taskIDs) == 0 {
		return "No task logs found\n"
	}

	lines := []string{"Tasks and Log Files:"}
	for i, taskID := range b.taskIDs {
		taskPrefix := "  "
		if i == b.selectedTaskIndex {
			taskPrefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s", taskPrefix, taskID))

		files := b.logFilesByTaskID[taskID]
		if len(files) == 0 {
			lines = append(lines, "    (no log files)")
			continue
		}
		for j, file := range files {
			filePrefix := "    "
			if i == b.selectedTaskIndex && j == b.selectedFileIndex {
				filePrefix = "    > "
			}
			lines = append(lines, fmt.Sprintf("%s%s", filePrefix, filepath.Base(file)))
		}
	}

	lines = append(lines, "", "Selected Log:")
	if b.selectedContent == "" {
		lines = append(lines, "<empty>")
	} else {
		lines = append(lines, strings.TrimRight(b.selectedContent, "\n"))
	}

	return strings.Join(lines, "\n") + "\n"
}
