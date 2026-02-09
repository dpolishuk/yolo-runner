package main

import "testing"

func TestCompatibilityNoticeMentionsNewCommands(t *testing.T) {
	notice := compatibilityNotice()
	if notice == "" {
		t.Fatalf("expected non-empty notice")
	}
	assertContainsText(t, notice, "yolo-agent")
	assertContainsText(t, notice, "yolo-task")
	assertContainsText(t, notice, "yolo-tui")
}

func assertContainsText(t *testing.T, text string, needle string) {
	t.Helper()
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return
		}
	}
	t.Fatalf("expected %q to include %q", text, needle)
}
