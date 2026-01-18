package main

import (
	"reflect"
	"testing"
)

func TestBeadsAdapterShowParsesFields(t *testing.T) {
	runner := NewFakeRunner()
	payload := `[
		{
			"id": "task-1",
			"title": "Do the thing",
			"description": "Detailed",
			"acceptance_criteria": "Must work"
		}
	]`
	runner.Script("bd", []string{"show", "task-1", "--json"}, []byte(payload))

	adapter := NewBeadsAdapter(runner)
	issue, err := adapter.Show("task-1")
	if err != nil {
		t.Fatalf("Show returned error: %v", err)
	}

	expected := Issue{
		ID:                 "task-1",
		Title:              "Do the thing",
		Description:        "Detailed",
		AcceptanceCriteria: "Must work",
	}

	if !reflect.DeepEqual(issue, expected) {
		t.Fatalf("Show mismatch: expected %#v, got %#v", expected, issue)
	}
}

func TestBeadsAdapterShowEmptyArrayFails(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"show", "task-1", "--json"}, []byte("[]"))

	adapter := NewBeadsAdapter(runner)
	_, err := adapter.Show("task-1")
	if err == nil {
		t.Fatal("Expected error for empty show response")
	}
}

func TestBeadsAdapterShowInvalidJSONFails(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"show", "task-1", "--json"}, []byte("nope"))

	adapter := NewBeadsAdapter(runner)
	_, err := adapter.Show("task-1")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestBeadsAdapterReadyChildrenParsesItems(t *testing.T) {
	runner := NewFakeRunner()
	payload := `[
		{
			"id": "child-1",
			"issue_type": "task",
			"status": "open",
			"priority": 1,
			"title": "Ignore"
		},
		{
			"id": "child-2",
			"issue_type": "epic",
			"status": "blocked",
			"priority": 3
		}
	]`
	runner.Script("bd", []string{"ready", "--parent", "root-1", "--json"}, []byte(payload))

	adapter := NewBeadsAdapter(runner)
	items, err := adapter.ReadyChildren("root-1")
	if err != nil {
		t.Fatalf("ReadyChildren returned error: %v", err)
	}

	expected := []ReadyChild{
		{ID: "child-1", IssueType: "task", Status: "open", Priority: 1},
		{ID: "child-2", IssueType: "epic", Status: "blocked", Priority: 3},
	}

	if !reflect.DeepEqual(items, expected) {
		t.Fatalf("ReadyChildren mismatch: expected %#v, got %#v", expected, items)
	}
}

func TestBeadsAdapterReadyChildrenInvalidJSONFails(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"ready", "--parent", "root-1", "--json"}, []byte("nope"))

	adapter := NewBeadsAdapter(runner)
	_, err := adapter.ReadyChildren("root-1")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestBeadsAdapterUpdateStatusRunsCommand(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"update", "task-1", "--status", "in_progress"}, []byte(""))

	adapter := NewBeadsAdapter(runner)
	if err := adapter.UpdateStatus("task-1", "in_progress"); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	expected := []CommandCall{{Name: "bd", Args: []string{"update", "task-1", "--status", "in_progress"}}}
	if !reflect.DeepEqual(runner.Calls(), expected) {
		t.Fatalf("Calls mismatch: expected %#v, got %#v", expected, runner.Calls())
	}
}

func TestBeadsAdapterCloseRunsCommand(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"close", "task-1"}, []byte(""))

	adapter := NewBeadsAdapter(runner)
	if err := adapter.Close("task-1"); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	expected := []CommandCall{{Name: "bd", Args: []string{"close", "task-1"}}}
	if !reflect.DeepEqual(runner.Calls(), expected) {
		t.Fatalf("Calls mismatch: expected %#v, got %#v", expected, runner.Calls())
	}
}

func TestBeadsAdapterSyncRunsCommand(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"sync"}, []byte(""))

	adapter := NewBeadsAdapter(runner)
	if err := adapter.Sync(); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	expected := []CommandCall{{Name: "bd", Args: []string{"sync"}}}
	if !reflect.DeepEqual(runner.Calls(), expected) {
		t.Fatalf("Calls mismatch: expected %#v, got %#v", expected, runner.Calls())
	}
}
