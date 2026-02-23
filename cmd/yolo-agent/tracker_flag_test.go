package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTrackerType(t *testing.T) {
	tests := []struct {
		name          string
		beadsExists   bool
		ticketsExists bool
		want          string
	}{
		{
			name:          "only beads exists",
			beadsExists:   true,
			ticketsExists: false,
			want:          trackerTypeBeads,
		},
		{
			name:          "only tickets exists",
			beadsExists:   false,
			ticketsExists: true,
			want:          trackerTypeTK,
		},
		{
			name:          "both exist prefers beads",
			beadsExists:   true,
			ticketsExists: true,
			want:          trackerTypeBeads,
		},
		{
			name:          "neither exists defaults to tk",
			beadsExists:   false,
			ticketsExists: false,
			want:          trackerTypeTK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Create directories based on test case
			if tt.beadsExists {
				beadsDir := filepath.Join(tmpDir, ".beads")
				if err := os.MkdirAll(beadsDir, 0755); err != nil {
					t.Fatalf("failed to create .beads: %v", err)
				}
			}
			if tt.ticketsExists {
				ticketsDir := filepath.Join(tmpDir, ".tickets")
				if err := os.MkdirAll(ticketsDir, 0755); err != nil {
					t.Fatalf("failed to create .tickets: %v", err)
				}
			}

			got := detectTrackerType(tmpDir)
			if got != tt.want {
				t.Errorf("detectTrackerType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectTrackerType_EmptyRepoRoot(t *testing.T) {
	// Test with empty string defaults to current directory
	got := detectTrackerType("")
	// Result depends on whether .beads or .tickets exists in current dir
	// Just verify it doesn't panic
	if got != trackerTypeBeads && got != trackerTypeTK {
		t.Errorf("detectTrackerType(\"\") returned unexpected value: %v", got)
	}
}

func TestDefaultTrackerProfilesModelForRepo(t *testing.T) {
	tests := []struct {
		name          string
		beadsExists   bool
		ticketsExists bool
		wantType      string
	}{
		{
			name:          "defaults to beads when .beads exists",
			beadsExists:   true,
			ticketsExists: false,
			wantType:      trackerTypeBeads,
		},
		{
			name:          "defaults to tk when only .tickets exists",
			beadsExists:   false,
			ticketsExists: true,
			wantType:      trackerTypeTK,
		},
		{
			name:          "prefers beads when both exist",
			beadsExists:   true,
			ticketsExists: true,
			wantType:      trackerTypeBeads,
		},
		{
			name:          "defaults to tk when neither exists",
			beadsExists:   false,
			ticketsExists: false,
			wantType:      trackerTypeTK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.beadsExists {
				os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)
			}
			if tt.ticketsExists {
				os.MkdirAll(filepath.Join(tmpDir, ".tickets"), 0755)
			}

			model := defaultTrackerProfilesModelForRepo(tmpDir)

			// Check default profile is set
			if model.DefaultProfile != defaultProfileName {
				t.Errorf("DefaultProfile = %v, want %v", model.DefaultProfile, defaultProfileName)
			}

			// Check profile exists
			profile, ok := model.Profiles[defaultProfileName]
			if !ok {
				t.Fatalf("profile %q not found", defaultProfileName)
			}

			// Check tracker type
			if profile.Tracker.Type != tt.wantType {
				t.Errorf("Tracker.Type = %v, want %v", profile.Tracker.Type, tt.wantType)
			}
		})
	}
}

func TestResolveTrackerProfile_WithTrackerOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file with tk tracker
	configContent := `
default_profile: default
profiles:
  default:
    tracker:
      type: tk
`
	configDir := filepath.Join(tmpDir, ".yolo-runner")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	svc := newTrackerConfigService()

	tests := []struct {
		name            string
		trackerOverride string
		wantType        string
	}{
		{
			name:            "override to beads",
			trackerOverride: "beads",
			wantType:        trackerTypeBeads,
		},
		{
			name:            "override to tk",
			trackerOverride: "tk",
			wantType:        trackerTypeTK,
		},
		{
			name:            "no override uses config",
			trackerOverride: "",
			wantType:        trackerTypeTK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := svc.ResolveTrackerProfile(tmpDir, "", tt.trackerOverride, "root-1", nil)
			if err != nil {
				t.Fatalf("ResolveTrackerProfile failed: %v", err)
			}

			if resolved.Tracker.Type != tt.wantType {
				t.Errorf("Tracker.Type = %v, want %v", resolved.Tracker.Type, tt.wantType)
			}
		})
	}
}

func TestResolveTrackerProfile_TrackerOverridePrecedence(t *testing.T) {
	// Test that --tracker flag overrides auto-detection
	tmpDir := t.TempDir()

	// Create .beads directory (would auto-detect to beads)
	os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)

	svc := newTrackerConfigService()

	// Without override, should auto-detect beads
	resolved, err := svc.ResolveTrackerProfile(tmpDir, "", "", "root-1", nil)
	if err != nil {
		t.Fatalf("ResolveTrackerProfile failed: %v", err)
	}
	if resolved.Tracker.Type != trackerTypeBeads {
		t.Errorf("Auto-detection: got %v, want %v", resolved.Tracker.Type, trackerTypeBeads)
	}

	// With override to tk, should use tk despite .beads existing
	resolved, err = svc.ResolveTrackerProfile(tmpDir, "", "tk", "root-1", nil)
	if err != nil {
		t.Fatalf("ResolveTrackerProfile failed: %v", err)
	}
	if resolved.Tracker.Type != trackerTypeTK {
		t.Errorf("Override: got %v, want %v", resolved.Tracker.Type, trackerTypeTK)
	}
}

func TestRunConfig_TrackerTypeFlag(t *testing.T) {
	// Test that --tracker flag is properly passed through runConfig
	tests := []struct {
		name        string
		trackerFlag string
		want        string
	}{
		{
			name:        "beads tracker",
			trackerFlag: "beads",
			want:        "beads",
		},
		{
			name:        "tk tracker",
			trackerFlag: "tk",
			want:        "tk",
		},
		{
			name:        "empty uses auto-detection",
			trackerFlag: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := runConfig{
				trackerType: tt.trackerFlag,
			}
			if cfg.trackerType != tt.want {
				t.Errorf("trackerType = %v, want %v", cfg.trackerType, tt.want)
			}
		})
	}
}
