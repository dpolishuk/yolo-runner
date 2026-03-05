package main

import (
	"testing"
)

func TestValidateTrackerModel_Beads(t *testing.T) {
	tests := []struct {
		name       string
		model      trackerModel
		rootID     string
		getenv     func(string) string
		wantErr    bool
		errMessage string
	}{
		{
			name: "beads type with valid scope",
			model: trackerModel{
				Type: "beads",
				Beads: &beadsTrackerModel{
					Scope: beadsScopeModel{
						Root: "epic-123",
					},
				},
			},
			rootID:  "epic-123",
			getenv:  nil,
			wantErr: false,
		},
		{
			name: "beads type with no beads config",
			model: trackerModel{
				Type: "beads",
			},
			rootID:  "",
			getenv:  nil,
			wantErr: false,
		},
		{
			name: "beads type with root outside scope",
			model: trackerModel{
				Type: "beads",
				Beads: &beadsTrackerModel{
					Scope: beadsScopeModel{
						Root: "epic-123",
					},
				},
			},
			rootID:     "other-epic",
			getenv:     nil,
			wantErr:    true,
			errMessage: `root "other-epic" is outside beads scope "epic-123"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateTrackerModel("test-profile", tt.model, tt.rootID, tt.getenv)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTrackerModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMessage != "" && err != nil {
				if !contains(err.Error(), tt.errMessage) {
					t.Errorf("validateTrackerModel() error = %v, want error containing %q", err, tt.errMessage)
				}
			}
		})
	}
}

func TestBuildTaskManagerForTracker_Beads(t *testing.T) {
	// Test that buildTaskManagerForTracker can handle beads type
	profile := resolvedTrackerProfile{
		Name: "test-beads-profile",
		Tracker: trackerModel{
			Type:  "beads",
			Beads: &beadsTrackerModel{},
		},
	}

	// This should not return "unsupported tracker type" error
	// It should attempt to create a beads task manager
	_, err := buildTaskManagerForTracker("/tmp/nonexistent", profile)
	// We expect it to fail because /tmp/nonexistent is not a valid beads repo
	// But it should NOT fail with "unsupported tracker type"
	if err != nil && contains(err.Error(), "not supported yet") {
		t.Errorf("buildTaskManagerForTracker() should support beads type, got: %v", err)
	}
	// The error is expected to be about beads capability probe failing
	// or path not existing, not about unsupported type
}

func TestBuildStorageBackendForTracker_Beads(t *testing.T) {
	// Test that buildStorageBackendForTracker can handle beads type
	profile := resolvedTrackerProfile{
		Name: "test-beads-profile",
		Tracker: trackerModel{
			Type:  "beads",
			Beads: &beadsTrackerModel{},
		},
	}

	// This should not return "unsupported tracker type" error
	_, err := buildStorageBackendForTracker("/tmp/nonexistent", profile)
	if err != nil && contains(err.Error(), "not supported yet") {
		t.Errorf("buildStorageBackendForTracker() should support beads type, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
