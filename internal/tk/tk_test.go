package tk

import (
	"strings"
	"testing"
)

func TestReadyUsesTkReadyAndQuery(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{
		"tk query": `{"id":"root","status":"open","type":"epic","priority":0}` + "\n" +
			`{"id":"root.1","status":"open","type":"task","priority":1,"parent":"root"}` + "\n" +
			`{"id":"other.1","status":"open","type":"task","priority":0,"parent":"other"}`,
		"tk ready": "root.1 [open] - Implement core contract\nother.1 [open] - Should be filtered out\n",
	}}

	a := New(r)
	issue, err := a.Ready("root")
	if err != nil {
		t.Fatalf("ready failed: %v", err)
	}

	if issue.ID != "root.1" {
		t.Fatalf("expected root.1, got %q", issue.ID)
	}
	if !r.called("tk ready") || !r.called("tk query") {
		t.Fatalf("expected tk ready and tk query to be called, got %v", r.calls)
	}
	if r.calledPrefix("tk list") {
		t.Fatalf("tk list should not be used for readiness")
	}
}

func TestUpdateStatusWithReasonUsesAddNote(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{}}
	a := New(r)

	if err := a.UpdateStatusWithReason("root.1", "blocked", "timeout happened"); err != nil {
		t.Fatalf("update status with reason failed: %v", err)
	}

	if !r.called("tk status root.1 blocked") {
		if !r.called("tk status root.1 open") {
			t.Fatalf("expected tk status/open call, got %v", r.calls)
		}
	}
	if !r.called("tk add-note root.1 blocked: timeout happened") {
		t.Fatalf("expected tk add-note call, got %v", r.calls)
	}
}

func TestShowUsesTkShowAndQuery(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{
		"tk query":       `{"id":"root.1","title":"Task title","description":"Task description","status":"open","type":"task","priority":2}`,
		"tk show root.1": "---\nid: root.1\n---\n# Task title\n",
	}}
	a := New(r)

	bead, err := a.Show("root.1")
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if bead.ID != "root.1" || bead.Title != "Task title" {
		t.Fatalf("unexpected bead: %#v", bead)
	}
	if !r.called("tk show root.1") || !r.called("tk query") {
		t.Fatalf("expected tk show and tk query calls, got %v", r.calls)
	}
}

type fakeRunner struct {
	responses map[string]string
	calls     []string
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	joined := strings.Join(args, " ")
	f.calls = append(f.calls, joined)
	if out, ok := f.responses[joined]; ok {
		return out, nil
	}
	if out, ok := f.responses[args[0]+" "+args[1]]; ok {
		return out, nil
	}
	return "", nil
}

func (f *fakeRunner) called(cmd string) bool {
	for _, call := range f.calls {
		if call == cmd {
			return true
		}
	}
	return false
}

func (f *fakeRunner) calledPrefix(prefix string) bool {
	for _, call := range f.calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}
	return false
}
