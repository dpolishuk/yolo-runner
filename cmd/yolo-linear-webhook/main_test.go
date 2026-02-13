package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunMainParsesFlagsAndInvokesRun(t *testing.T) {
	called := false
	var got runConfig
	run := func(_ context.Context, cfg runConfig) error {
		called = true
		got = cfg
		return nil
	}

	code := RunMain([]string{"--listen", ":9123", "--path", "/hooks/linear", "--queue-path", "runner-logs/linear.jobs.jsonl", "--queue-buffer", "256", "--shutdown-timeout", "7s"}, run)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !called {
		t.Fatal("expected run function to be called")
	}
	if got.listenAddr != ":9123" {
		t.Fatalf("expected listen address :9123, got %q", got.listenAddr)
	}
	if got.webhookPath != "/hooks/linear" {
		t.Fatalf("expected webhook path /hooks/linear, got %q", got.webhookPath)
	}
	if got.queuePath != "runner-logs/linear.jobs.jsonl" {
		t.Fatalf("expected queue path to be parsed, got %q", got.queuePath)
	}
	if got.queueBuffer != 256 {
		t.Fatalf("expected queue buffer=256, got %d", got.queueBuffer)
	}
	if got.shutdownTimeout != 7*time.Second {
		t.Fatalf("expected shutdown timeout=7s, got %s", got.shutdownTimeout)
	}
}

func TestRunMainRejectsInvalidQueueBuffer(t *testing.T) {
	called := false
	code := RunMain([]string{"--queue-buffer", "0"}, func(context.Context, runConfig) error {
		called = true
		return nil
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if called {
		t.Fatal("expected run not to be called")
	}
}

func TestRunMainRejectsInvalidShutdownTimeout(t *testing.T) {
	called := false
	code := RunMain([]string{"--shutdown-timeout", "0s"}, func(context.Context, runConfig) error {
		called = true
		return nil
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if called {
		t.Fatal("expected run not to be called")
	}
}

func TestRunMainReturnsErrorWhenRunFails(t *testing.T) {
	code := RunMain([]string{}, func(context.Context, runConfig) error {
		return errors.New("boom")
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}
