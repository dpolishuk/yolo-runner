package main

import (
	"strings"
	"sync"

	"github.com/anomalyco/yolo-runner/internal/ui/tui"
)

type tuiLogWriter struct {
	program tuiProgram
	mu      sync.Mutex
	buffer  strings.Builder
}

func newTUILogWriter(program tuiProgram) *tuiLogWriter {
	return &tuiLogWriter{program: program}
}

func (w *tuiLogWriter) Write(p []byte) (int, error) {
	if w == nil {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer.Write(p)
	buffered := w.buffer.String()
	lines := strings.Split(buffered, "\n")
	for i := 0; i < len(lines)-1; i++ {
		w.emit(strings.TrimRight(lines[i], "\r"))
	}
	remaining := lines[len(lines)-1]
	w.buffer.Reset()
	if remaining != "" {
		w.buffer.WriteString(remaining)
	}
	return len(p), nil
}

func (w *tuiLogWriter) Flush() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := strings.TrimRight(w.buffer.String(), "\r")
	w.buffer.Reset()
	if remaining == "" {
		return
	}
	w.emit(remaining)
}

func (w *tuiLogWriter) emit(line string) {
	if w.program == nil {
		return
	}
	w.program.SendInput(tui.AppendLogMsg{Line: line})
}
