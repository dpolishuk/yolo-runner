package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestFakeRunnerRecordsCommands(t *testing.T) {
	runner := NewFakeRunner()
	runner.Script("bd", []string{"ready"}, []byte("[]"))
	runner.Script("git", []string{"status", "--porcelain"}, []byte(""))

	if _, err := runner.Run("bd", "ready"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if _, err := runner.Run("git", "status", "--porcelain"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	expected := []CommandCall{
		{Name: "bd", Args: []string{"ready"}},
		{Name: "git", Args: []string{"status", "--porcelain"}},
	}

	if !reflect.DeepEqual(runner.Calls(), expected) {
		t.Fatalf("Calls mismatch: expected %#v, got %#v", expected, runner.Calls())
	}
}

func TestFakeRunnerReturnsScriptedOutput(t *testing.T) {
	runner := NewFakeRunner()
	expected := []byte("ok")
	runner.Script("git", []string{"status"}, expected)

	output, err := runner.Run("git", "status")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !reflect.DeepEqual(output, expected) {
		t.Fatalf("Output mismatch: expected %q, got %q", expected, output)
	}
}

func TestFakeRunnerMissingStubFails(t *testing.T) {
	runner := NewFakeRunner()

	_, err := runner.Run("git", "status")
	if err == nil {
		t.Fatal("Expected missing stub error")
	}

	message := err.Error()
	if !strings.Contains(message, "missing stub") {
		t.Fatalf("Expected missing stub error, got %q", message)
	}
	if !strings.Contains(message, "git") {
		t.Fatalf("Expected error to mention command name, got %q", message)
	}
}
