package main

import "testing"

func TestNormalizeDistributedRoleForRunnerDefaultsToLocal(t *testing.T) {
	role, err := normalizeDistributedRoleForRunner("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != runnerRoleLocal {
		t.Fatalf("expected default role %q, got %q", runnerRoleLocal, role)
	}
}

func TestNormalizeDistributedRoleForRunnerUsesEnv(t *testing.T) {
	role, err := normalizeDistributedRoleForRunner("", runnerRoleWorker)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != runnerRoleWorker {
		t.Fatalf("expected env role %q, got %q", runnerRoleWorker, role)
	}
}

func TestNormalizeDistributedRoleForRunnerRejectsInvalidRole(t *testing.T) {
	_, err := normalizeDistributedRoleForRunner("unknown", "")
	if err == nil {
		t.Fatalf("expected invalid role error")
	}
}

func TestNormalizeDistributedBusBackendForRunnerDefaultsToRedis(t *testing.T) {
	backend, err := normalizeDistributedBusBackendForRunner("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend != runnerDistributedBusRedis {
		t.Fatalf("expected redis backend, got %q", backend)
	}
}

func TestNormalizeDistributedBusBackendForRunnerRejectsInvalidBackend(t *testing.T) {
	_, err := normalizeDistributedBusBackendForRunner("kafka")
	if err == nil {
		t.Fatalf("expected invalid backend error")
	}
}
