package distributed

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultExecutorConfigWatchInterval = 1 * time.Second
)

var executorConfigPlaceholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

var executorConfigAllowedKeys = map[string]struct{}{
	"name":     {},
	"type":     {},
	"backend":  {},
	"pipeline": {},
}

var executorConfigStageNames = map[string]struct{}{
	"quality_gate": {},
	"execute":      {},
	"qc_gate":      {},
	"complete":     {},
}

var executorConfigAllowedTransitionActions = map[string]struct{}{
	"next":     {},
	"retry":    {},
	"fail":     {},
	"complete": {},
}

var executorConfigConditionPattern = regexp.MustCompile(`^\s*(?:true|false|tests_failed|review_failed|quality_score\s*(?:==|!=|>=|<=|>|<)\s*(?:[0-9]+(?:\.[0-9]+)?|threshold))\s*$`)

type ExecutorConfig struct {
	Name     string                         `json:"name" yaml:"name"`
	Type     string                         `json:"type" yaml:"type"`
	Backend  string                         `json:"backend" yaml:"backend"`
	Pipeline map[string]ExecutorConfigStage `json:"pipeline" yaml:"pipeline"`
}

type ExecutorConfigStage struct {
	Tools       []string                  `json:"tools" yaml:"tools"`
	Retry       ExecutorConfigRetry       `json:"retry" yaml:"retry"`
	Transitions ExecutorConfigTransitions `json:"transitions" yaml:"transitions"`
}

type ExecutorConfigRetry struct {
	MaxAttempts    int `json:"max_attempts" yaml:"max_attempts"`
	InitialDelayMs int `json:"initial_delay_ms" yaml:"initial_delay_ms"`
	BackoffMs      int `json:"backoff_ms" yaml:"backoff_ms"`
	MaxDelayMs     int `json:"max_delay_ms" yaml:"max_delay_ms"`
}

type ExecutorConfigTransitions struct {
	OnSuccess ExecutorConfigTransition `json:"on_success" yaml:"on_success"`
	OnFailure ExecutorConfigTransition `json:"on_failure" yaml:"on_failure"`
}

type ExecutorConfigTransition struct {
	Action    string `json:"action" yaml:"action"`
	NextStage string `json:"next_stage,omitempty" yaml:"next_stage,omitempty"`
	Condition string `json:"condition" yaml:"condition"`
}

type ExecutorConfigWatchEvent struct {
	Path   string
	Config ExecutorConfig
	Err    error
}

type executorConfigLoader struct {
	readFile func(string) ([]byte, error)
	getenv   func(string) string
}

// NewExecutorConfigLoader returns a default executor configuration loader.
func NewExecutorConfigLoader() executorConfigLoader {
	return executorConfigLoader{
		readFile: os.ReadFile,
		getenv:   os.Getenv,
	}
}

func defaultLoadExecutorConfig(path string) (ExecutorConfig, error) {
	return NewExecutorConfigLoader().Load(path)
}

// LoadExecutorConfig loads and validates an executor configuration from disk.
func LoadExecutorConfig(path string) (ExecutorConfig, error) {
	return defaultLoadExecutorConfig(path)
}

func (l executorConfigLoader) Load(path string) (ExecutorConfig, error) {
	content, err := l.readFile(path)
	if err != nil {
		return ExecutorConfig{}, fmt.Errorf("cannot read executor config at %q: %w", path, err)
	}
	return loadExecutorConfigFromBytes(path, content, l.getenv)
}

// WatchExecutorConfig watches a configuration file for changes and invokes onChange whenever
// the file content changes. The first load (successful or not) is emitted immediately.
func WatchExecutorConfig(
	ctx context.Context,
	path string,
	interval time.Duration,
	onChange func(ExecutorConfigWatchEvent),
	getenv func(string) string,
) error {
	loader := NewExecutorConfigLoader()
	if getenv != nil {
		loader.getenv = getenv
	}
	return loader.Watch(ctx, path, interval, onChange)
}

// Watch reads configuration changes in a polling loop and sends updates on each file
// content change. The callback can inspect Err to detect invalid or unreadable config.
func (l executorConfigLoader) Watch(
	ctx context.Context,
	path string,
	interval time.Duration,
	onChange func(ExecutorConfigWatchEvent),
) error {
	if onChange == nil {
		return fmt.Errorf("executor config watch callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("executor config path is required")
	}
	if interval <= 0 {
		interval = defaultExecutorConfigWatchInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastChecksum [20]byte
	var haveLastChecksum bool
	var lastErr error

	loadAndEmit := func() {
		content, err := l.readFile(path)
		if err != nil {
			eventErr := fmt.Errorf("cannot read executor config at %q: %w", path, err)
			if lastErr != nil && eventErr.Error() == lastErr.Error() {
				return
			}
			lastErr = eventErr
			onChange(ExecutorConfigWatchEvent{Path: path, Err: eventErr})
			return
		}
		checksum := sha1.Sum(content)
		if haveLastChecksum && checksum == lastChecksum {
			return
		}
		lastChecksum = checksum
		haveLastChecksum = true

		cfg, err := loadExecutorConfigFromBytes(path, content, l.getenv)
		lastErr = nil
		onChange(ExecutorConfigWatchEvent{Path: path, Config: cfg, Err: err})
	}

	loadAndEmit()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			loadAndEmit()
		}
	}
}

func loadExecutorConfigFromBytes(path string, content []byte, getenv func(string) string) (ExecutorConfig, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return ExecutorConfig{}, fmt.Errorf("cannot parse executor config at %q: %w", path, err)
	}

	var substitutionErrs []string
	if getenv != nil {
		substituted, err := substituteExecutorConfigEnv(raw, getenv)
		if err != nil {
			substitutionErrs = append(substitutionErrs, err.Error())
			validationIssues := validateExecutorConfig(path, raw)
			validationIssues = append(validationIssues, substitutionErrs...)
			if len(validationIssues) > 0 {
				return ExecutorConfig{}, fmt.Errorf("invalid executor config at %q: %s", path, strings.Join(validationIssues, "; "))
			}
		}
		var ok bool
		raw, ok = substituted.(map[string]any)
		if !ok {
			return ExecutorConfig{}, fmt.Errorf("invalid executor config at %q: top-level config must be an object", path)
		}
	}
	raw = normalizeExecutorConfigValue(raw).(map[string]any)

	validationIssues := validateExecutorConfig(path, raw)
	validationIssues = append(validationIssues, substitutionErrs...)
	if len(validationIssues) > 0 {
		return ExecutorConfig{}, fmt.Errorf("invalid executor config at %q: %s", path, strings.Join(validationIssues, "; "))
	}

	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return ExecutorConfig{}, fmt.Errorf("marshal normalized executor config for %q: %w", path, err)
	}
	var config ExecutorConfig
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return ExecutorConfig{}, fmt.Errorf("map executor config from %q into schema: %w", path, err)
	}
	return config, nil
}

func normalizeExecutorConfigValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeExecutorConfigValue(nested)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, nested := range typed {
			text := fmt.Sprintf("%v", key)
			normalized[text] = normalizeExecutorConfigValue(nested)
		}
		return normalized
	case []any:
		out := make([]any, len(typed))
		for idx := range typed {
			out[idx] = normalizeExecutorConfigValue(typed[idx])
		}
		return out
	default:
		return value
	}
}

func substituteExecutorConfigEnv(value any, getenv func(string) string) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			substituted, err := substituteExecutorConfigEnv(nested, getenv)
			if err != nil {
				return nil, err
			}
			out[key] = substituted
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for idx := range typed {
			substituted, err := substituteExecutorConfigEnv(typed[idx], getenv)
			if err != nil {
				return nil, err
			}
			out[idx] = substituted
		}
		return out, nil
	case string:
		missingVars := map[string]struct{}{}
		subbed := executorConfigPlaceholderPattern.ReplaceAllStringFunc(typed, func(match string) string {
			sub := executorConfigPlaceholderPattern.FindStringSubmatch(match)
			if len(sub) != 2 {
				return match
			}
			envName := sub[1]
			envValue := strings.TrimSpace(getenv(envName))
			if envValue == "" {
				missingVars[envName] = struct{}{}
				return ""
			}
			return envValue
		})
		if len(missingVars) > 0 {
			var missing []string
			for envName := range missingVars {
				missing = append(missing, envName)
			}
			return nil, fmt.Errorf("missing environment variables in executor config: %s", strings.Join(missing, ", "))
		}
		if strings.Contains(subbed, "${") {
			return nil, fmt.Errorf("failed to substitute all variables in value %q", typed)
		}
		return subbed, nil
	default:
		return value, nil
	}
}

func validateExecutorConfig(path string, config map[string]any) []string {
	var issues []string
	for key := range config {
		if _, ok := executorConfigAllowedKeys[key]; !ok {
			issues = append(issues, fmt.Sprintf("top-level key %q is not allowed", key))
		}
	}
	for _, field := range []string{"name", "type", "backend", "pipeline"} {
		if _, ok := config[field]; !ok {
			issues = append(issues, fmt.Sprintf("field %q is required", field))
		}
	}

	name, ok := config["name"].(string)
	if ok {
		if strings.TrimSpace(name) == "" {
			issues = append(issues, "field name must be non-empty")
		}
	}
	configType, ok := config["type"].(string)
	if ok {
		if strings.TrimSpace(configType) == "" {
			issues = append(issues, "field type must be non-empty")
		}
	}
	backend, ok := config["backend"].(string)
	if ok {
		if strings.TrimSpace(backend) == "" {
			issues = append(issues, "field backend must be non-empty")
		}
	}

	pipelineRaw, ok := config["pipeline"].(map[string]any)
	if !ok {
		if _, exists := config["pipeline"]; exists {
			issues = append(issues, "field pipeline must be an object")
		}
		return issues
	}
	for stage := range pipelineRaw {
		if _, ok := executorConfigStageNames[stage]; !ok {
			issues = append(issues, fmt.Sprintf("pipeline includes unknown stage %q", stage))
		}
	}
	for _, stageName := range []string{"quality_gate", "execute", "qc_gate", "complete"} {
		stage, exists := pipelineRaw[stageName]
		if !exists {
			issues = append(issues, fmt.Sprintf("pipeline missing required stage %q", stageName))
			continue
		}
		issues = append(issues, validateExecutorStage(fmt.Sprintf("pipeline.%s", stageName), stage)...)
	}
	if len(pipelineRaw) != len(executorConfigStageNames) {
		issues = append(issues, "pipeline must contain exactly quality_gate, execute, qc_gate, and complete stages")
	}
	return issues
}

func validateExecutorStage(field string, value any) []string {
	var issues []string
	stage, ok := value.(map[string]any)
	if !ok {
		issues = append(issues, fmt.Sprintf("%s must be an object", field))
		return issues
	}
	for key := range stage {
		switch key {
		case "tools", "retry", "transitions":
		default:
			issues = append(issues, fmt.Sprintf("%s has unknown field %q", field, key))
		}
	}

	toolsRaw, ok := stage["tools"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.tools is required", field))
	} else {
		tools, ok := toolsRaw.([]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s.tools must be a list", field))
		} else if len(tools) == 0 {
			issues = append(issues, fmt.Sprintf("%s.tools must include at least one tool", field))
		} else {
			for i, toolRaw := range tools {
				tool, ok := toolRaw.(string)
				if !ok || strings.TrimSpace(tool) == "" {
					issues = append(issues, fmt.Sprintf("%s.tools[%d] must be a non-empty string", field, i))
				}
			}
		}
	}

	retryRaw, ok := stage["retry"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.retry is required", field))
	} else {
		retry, ok := retryRaw.(map[string]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s.retry must be an object", field))
		} else {
			issues = append(issues, validateExecutorRetry(field, retry)...)
		}
	}

	transitionsRaw, ok := stage["transitions"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.transitions is required", field))
	} else {
		transitions, ok := transitionsRaw.(map[string]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s.transitions must be an object", field))
		} else {
			issues = append(issues, validateExecutorTransitions(field, transitions)...)
		}
	}
	return issues
}

func validateExecutorRetry(path string, value map[string]any) []string {
	var issues []string
	allowed := map[string]struct{}{
		"max_attempts":     {},
		"initial_delay_ms": {},
		"backoff_ms":       {},
		"max_delay_ms":     {},
	}
	for key := range value {
		if _, ok := allowed[key]; !ok {
			issues = append(issues, fmt.Sprintf("%s.retry has unknown field %q", path, key))
		}
	}
	maxAttemptsRaw, ok := value["max_attempts"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.retry.max_attempts is required", path))
		return issues
	}
	maxAttempts, ok := asInt(maxAttemptsRaw)
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.retry.max_attempts must be an integer", path))
	} else if maxAttempts < 1 {
		issues = append(issues, fmt.Sprintf("%s.retry.max_attempts must be >= 1", path))
	}

	if initialDelayRaw, ok := value["initial_delay_ms"]; ok {
		if initialDelay, ok := asInt(initialDelayRaw); !ok || initialDelay < 0 {
			issues = append(issues, fmt.Sprintf("%s.retry.initial_delay_ms must be an integer >= 0", path))
		}
	}
	if backoffRaw, ok := value["backoff_ms"]; ok {
		if backoff, ok := asInt(backoffRaw); !ok || backoff < 0 {
			issues = append(issues, fmt.Sprintf("%s.retry.backoff_ms must be an integer >= 0", path))
		}
	}
	if maxDelayRaw, ok := value["max_delay_ms"]; ok {
		if maxDelay, ok := asInt(maxDelayRaw); !ok || maxDelay < 0 {
			issues = append(issues, fmt.Sprintf("%s.retry.max_delay_ms must be an integer >= 0", path))
		}
	}
	return issues
}

func validateExecutorTransitions(path string, value map[string]any) []string {
	var issues []string
	successRaw, ok := value["on_success"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.transitions.on_success is required", path))
	} else {
		issues = append(issues, validateExecutorTransition(path+".transitions.on_success", successRaw)...)
	}
	failureRaw, ok := value["on_failure"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.transitions.on_failure is required", path))
	} else {
		issues = append(issues, validateExecutorTransition(path+".transitions.on_failure", failureRaw)...)
	}
	for key := range value {
		if key != "on_success" && key != "on_failure" {
			issues = append(issues, fmt.Sprintf("%s.transitions has unknown field %q", path, key))
		}
	}
	return issues
}

func validateExecutorTransition(path string, value any) []string {
	var issues []string
	transition, ok := value.(map[string]any)
	if !ok {
		issues = append(issues, fmt.Sprintf("%s must be an object", path))
		return issues
	}
	for key := range transition {
		switch key {
		case "action", "next_stage", "condition":
		default:
			issues = append(issues, fmt.Sprintf("%s has unknown field %q", path, key))
		}
	}

	actionRaw, ok := transition["action"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.action is required", path))
		return issues
	}
	action, ok := actionRaw.(string)
	if !ok || strings.TrimSpace(action) == "" {
		issues = append(issues, fmt.Sprintf("%s.action must be a non-empty string", path))
		return issues
	}
	if _, ok := executorConfigAllowedTransitionActions[action]; !ok {
		issues = append(issues, fmt.Sprintf("%s.action %q is invalid", path, action))
	}

	conditionRaw, ok := transition["condition"]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s.condition is required", path))
	} else {
		condition, ok := conditionRaw.(string)
		if !ok || !executorConfigConditionPattern.MatchString(condition) {
			issues = append(issues, fmt.Sprintf("%s.condition %q is invalid", path, conditionRaw))
		}
	}

	nextStageRaw, hasNext := transition["next_stage"]
	nextStage, _ := nextStageRaw.(string)
	if action == "next" {
		if !hasNext || strings.TrimSpace(nextStage) == "" {
			issues = append(issues, fmt.Sprintf("%s.next_stage is required when action is next", path))
		} else if _, ok := executorConfigStageNames[nextStage]; !ok {
			issues = append(issues, fmt.Sprintf("%s.next_stage %q is invalid", path, nextStage))
		}
	} else if hasNext && strings.TrimSpace(nextStage) != "" {
		issues = append(issues, fmt.Sprintf("%s.next_stage is only valid when action is next", path))
	}
	return issues
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		if math.Trunc(float64(typed)) != float64(typed) {
			return 0, false
		}
		return int(typed), true
	case float64:
		if math.Trunc(typed) != typed {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}
