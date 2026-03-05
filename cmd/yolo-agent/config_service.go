package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type trackerConfigService struct {
	readFile func(string) ([]byte, error)
}

func newTrackerConfigService() trackerConfigService {
	return trackerConfigService{
		readFile: os.ReadFile,
	}
}

func (s trackerConfigService) LoadModel(repoRoot string) (trackerProfilesModel, error) {
	configPath := filepath.Join(repoRoot, trackerConfigRelPath)
	return s.loadModelFromPath(configPath)
}

func (s trackerConfigService) ResolveAgentDefaults(repoRoot string) (yoloAgentConfigDefaults, error) {
	model, err := s.LoadModel(repoRoot)
	if err != nil {
		return yoloAgentConfigDefaults{}, err
	}
	catalog, err := loadCodingAgentsCatalog(repoRoot)
	if err != nil {
		return yoloAgentConfigDefaults{}, err
	}
	return resolveYoloAgentConfigDefaults(model.Agent, catalog)
}

func (s trackerConfigService) ResolveTrackerProfile(repoRoot string, selectedProfile string, trackerTypeOverride string, rootID string, getenv func(string) string) (resolvedTrackerProfile, error) {
	model, err := s.LoadModel(repoRoot)
	if err != nil {
		return resolvedTrackerProfile{}, err
	}

	profileName := strings.TrimSpace(selectedProfile)
	if profileName == "" {
		profileName = strings.TrimSpace(model.DefaultProfile)
	}
	if profileName == "" {
		profileName = defaultProfileName
	}

	profile, ok := model.Profiles[profileName]
	if !ok {
		return resolvedTrackerProfile{}, fmt.Errorf("tracker profile %q not found (available: %s)", profileName, strings.Join(sortedProfileNames(model.Profiles), ", "))
	}

	// Apply tracker type override from CLI flag if provided
	if trackerTypeOverride != "" {
		profile.Tracker.Type = trackerTypeOverride
	}

	validated, err := validateTrackerModel(profileName, profile.Tracker, rootID, getenv)
	if err != nil {
		return resolvedTrackerProfile{}, err
	}
	return resolvedTrackerProfile{
		Name:    profileName,
		Tracker: validated,
	}, nil
}

func (s trackerConfigService) loadModelFromPath(path string) (trackerProfilesModel, error) {
	content, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Extract repo root from config path for auto-detection
			// path is like: repoRoot/.yolo-runner/config.yaml
			// so repoRoot is parent of .yolo-runner directory
			yoloRunnerDir := filepath.Dir(path)
			repoRoot := filepath.Dir(yoloRunnerDir)
			return defaultTrackerProfilesModelForRepo(repoRoot), nil
		}
		return trackerProfilesModel{}, fmt.Errorf("cannot read config file at %s: %w", trackerConfigRelPath, err)
	}

	var model trackerProfilesModel
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&model); err != nil {
		return trackerProfilesModel{}, fmt.Errorf("cannot parse config file at %s: %w", trackerConfigRelPath, err)
	}

	if len(model.Profiles) == 0 && strings.TrimSpace(model.Tracker.Type) != "" {
		model.Profiles = map[string]trackerProfileDef{
			defaultProfileName: {Tracker: model.Tracker},
		}
	}

	if len(model.Profiles) == 0 {
		return trackerProfilesModel{}, fmt.Errorf("config file at %s must define at least one profile", trackerConfigRelPath)
	}
	return model, nil
}
