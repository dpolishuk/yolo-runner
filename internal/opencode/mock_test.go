package opencode

import (
	"testing"
)

func TestMockProcess(t *testing.T) {
	// Test that MockProcess works correctly
	mockProcess := &MockProcess{
		waitCh: make(chan error, 1),
		killCh: make(chan error, 1),
		stdin:  &nopWriteCloser{},
		stdout: &nopReadCloser{},
	}

	// Send value to waitCh so Wait doesn't block
	mockProcess.waitCh <- nil

	// Send value to killCh so Kill doesn't block
	mockProcess.killCh <- nil

	err := mockProcess.Wait()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Test Kill method
	err = mockProcess.Kill()
	if err != nil {
		t.Fatalf("expected no error from Kill, got: %v", err)
	}

	if !mockProcess.killCalled {
		t.Fatal("expected Kill to be called")
	}
}
