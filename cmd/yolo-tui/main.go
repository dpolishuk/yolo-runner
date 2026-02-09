package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/anomalyco/yolo-runner/internal/contracts"
	"github.com/anomalyco/yolo-runner/internal/ui/monitor"
)

func main() {
	os.Exit(RunMain(os.Args[1:], os.Stdout, os.Stderr))
}

func RunMain(args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("yolo-tui", flag.ContinueOnError)
	fs.SetOutput(errOut)
	events := fs.String("events", "", "Path to JSONL events file")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *events == "" {
		fmt.Fprintln(errOut, "--events is required")
		return 1
	}

	file, err := os.Open(*events)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer file.Close()

	model := monitor.NewModel(time.Now)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		event, parseErr := parseEvent(line)
		if parseErr != nil {
			fmt.Fprintln(errOut, parseErr)
			return 1
		}
		model.Apply(event)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	_, _ = io.WriteString(out, model.View())
	return 0
}

func parseEvent(line []byte) (contracts.Event, error) {
	var payload struct {
		Type      string            `json:"type"`
		TaskID    string            `json:"task_id"`
		TaskTitle string            `json:"task_title"`
		Message   string            `json:"message"`
		Metadata  map[string]string `json:"metadata"`
		TS        string            `json:"ts"`
	}
	if err := json.Unmarshal(line, &payload); err != nil {
		return contracts.Event{}, err
	}
	timestamp := time.Time{}
	if payload.TS != "" {
		parsed, err := time.Parse(time.RFC3339, payload.TS)
		if err != nil {
			return contracts.Event{}, err
		}
		timestamp = parsed
	}
	return contracts.Event{
		Type:      contracts.EventType(payload.Type),
		TaskID:    payload.TaskID,
		TaskTitle: payload.TaskTitle,
		Message:   payload.Message,
		Metadata:  payload.Metadata,
		Timestamp: timestamp,
	}, nil
}
