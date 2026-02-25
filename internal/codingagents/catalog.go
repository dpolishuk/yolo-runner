package codingagents

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egv/yolo-runner/v2/internal/distributed"
	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

const (
	builtinBackendDir    = "builtin"
	codingAgentConfigDir = ".yolo-runner"
	customBackendRelPath = "coding-agents"
)

type BackendDefinition struct {
	Name                string                   `yaml:"name" json:"name"`
	Adapter             string                   `yaml:"adapter" json:"adapter"`
	Binary              string                   `yaml:"binary" json:"binary"`
	Command             string                   `yaml:"command" json:"command"`
	Args                []string                 `yaml:"args" json:"args"`
	SupportsReview      bool                     `yaml:"supports_review" json:"supports_review"`
	SupportsStream      bool                     `yaml:"supports_stream" json:"supports_stream"`
	DistributedCaps     []distributed.Capability `yaml:"distributed_capabilities" json:"distributed_capabilities"`
	SupportedModels     []string                 `yaml:"supported_models" json:"supported_models"`
	RequiredCredentials []string                 `yaml:"required_credentials" json:"required_credentials"`
}

type BackendCapabilities struct {
	SupportsReview bool
	SupportsStream bool
}

type Catalog struct {
	backends map[string]BackendDefinition
}

func LoadCatalog(repoRoot string) (Catalog, error) {
	catalog := Catalog{backends: map[string]BackendDefinition{}}

	builtin, err := loadBuiltinBackends()
	if err != nil {
		return Catalog{}, err
	}
	for _, definition := range builtin {
		if err := catalog.add(definition); err != nil {
			return Catalog{}, err
		}
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return catalog, nil
	}

	customDir := filepath.Join(repoRoot, codingAgentConfigDir, customBackendRelPath)
	entries, err := os.ReadDir(customDir)
	if err != nil {
		if os.IsNotExist(err) {
			return catalog, nil
		}
		return Catalog{}, fmt.Errorf("cannot read custom coding agents from %q: %w", customDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		switch extension {
		case ".yaml", ".yml", ".json":
		default:
			continue
		}

		fullPath := filepath.Join(customDir, entry.Name())
		payload, err := os.ReadFile(fullPath)
		if err != nil {
			return Catalog{}, fmt.Errorf("read custom backend definition %q: %w", fullPath, err)
		}

		definition, err := parseBackendDefinition(payload, extension)
		if err != nil {
			return Catalog{}, fmt.Errorf("parse custom backend definition %q: %w", fullPath, err)
		}
		definition = normalizeBackendDefinition(definition)
		if err := validateBackendDefinition(definition); err != nil {
			return Catalog{}, fmt.Errorf("invalid custom backend definition %q: %w", fullPath, err)
		}
		if err := catalog.add(definition); err != nil {
			return Catalog{}, err
		}
	}

	return catalog, nil
}

func (c *Catalog) add(raw BackendDefinition) error {
	if c.backends == nil {
		c.backends = map[string]BackendDefinition{}
	}
	definition := normalizeBackendDefinition(raw)
	if strings.TrimSpace(definition.Name) == "" {
		return fmt.Errorf("backend name is required")
	}
	if err := validateBackendDefinition(definition); err != nil {
		return fmt.Errorf("invalid backend definition %q: %w", strings.TrimSpace(definition.Name), err)
	}
	c.backends[definition.Name] = definition
	return nil
}

func (c Catalog) Backend(name string) (BackendDefinition, bool) {
	if c.backends == nil {
		return BackendDefinition{}, false
	}
	backend, ok := c.backends[normalizeBackend(name)]
	return backend, ok
}

func (c Catalog) Names() []string {
	if len(c.backends) == 0 {
		return nil
	}
	values := make([]string, 0, len(c.backends))
	for name := range c.backends {
		values = append(values, name)
	}
	sort.Strings(values)
	return values
}

func (c Catalog) CapabilityProfile(name string) (BackendCapabilities, bool) {
	backend, ok := c.Backend(name)
	if !ok {
		return BackendCapabilities{}, false
	}
	return BackendCapabilities{SupportsReview: backend.SupportsReview, SupportsStream: backend.SupportsStream}, true
}

func (c Catalog) DistributedCapabilities(name string) ([]distributed.Capability, bool) {
	backend, ok := c.Backend(name)
	if !ok {
		return nil, false
	}
	if len(backend.DistributedCaps) == 0 {
		return nil, true
	}
	return append([]distributed.Capability(nil), backend.DistributedCaps...), true
}

func (c Catalog) ValidateBackendUsage(name string, model string, getenv func(string) string) error {
	backend, ok := c.Backend(name)
	if !ok {
		return fmt.Errorf("unsupported backend %q", name)
	}

	if strings.TrimSpace(model) != "" && !supportsModelPattern(backend.SupportedModels, model) {
		return fmt.Errorf("unsupported model %q for backend %q (supported: %s)", strings.TrimSpace(model), backend.Name, strings.Join(backend.SupportedModels, ", "))
	}

	if getenv == nil {
		getenv = os.Getenv
	}
	for _, envVar := range backend.RequiredCredentials {
		trimmedEnvVar := strings.TrimSpace(envVar)
		if trimmedEnvVar == "" {
			continue
		}
		if strings.TrimSpace(getenv(trimmedEnvVar)) == "" {
			return fmt.Errorf("missing auth token from %s for backend %q", trimmedEnvVar, backend.Name)
		}
	}
	return nil
}

func loadBuiltinBackends() ([]BackendDefinition, error) {
	entries, err := fs.ReadDir(builtinFS, builtinBackendDir)
	if err != nil {
		return nil, fmt.Errorf("read builtin backend definitions: %w", err)
	}
	out := make([]BackendDefinition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		switch extension {
		case ".yaml", ".yml":
		default:
			continue
		}
		payload, err := fs.ReadFile(builtinFS, filepath.ToSlash(filepath.Join(builtinBackendDir, entry.Name())))
		if err != nil {
			return nil, fmt.Errorf("read builtin backend definition %q: %w", entry.Name(), err)
		}
		definition, err := parseBackendDefinition(payload, extension)
		if err != nil {
			return nil, fmt.Errorf("parse builtin backend definition %q: %w", entry.Name(), err)
		}
		definition = normalizeBackendDefinition(definition)
		if err := validateBackendDefinition(definition); err != nil {
			return nil, fmt.Errorf("invalid builtin backend definition %q: %w", entry.Name(), err)
		}
		out = append(out, definition)
	}
	return out, nil
}

func parseBackendDefinition(payload []byte, extension string) (BackendDefinition, error) {
	definition := BackendDefinition{}
	content := strings.TrimSpace(string(payload))
	if content == "" {
		return BackendDefinition{}, fmt.Errorf("backend definition is empty")
	}
	switch strings.TrimSpace(strings.ToLower(extension)) {
	case ".json":
		if err := json.Unmarshal([]byte(content), &definition); err != nil {
			return BackendDefinition{}, err
		}
	default:
		if err := yaml.Unmarshal([]byte(content), &definition); err != nil {
			return BackendDefinition{}, err
		}
	}
	definition = normalizeBackendDefinition(definition)
	if definition.Name == "" {
		return BackendDefinition{}, fmt.Errorf("backend name is required")
	}
	if definition.Adapter == "" {
		definition.Adapter = "command"
	}
	if definition.Command != "" && definition.Binary == "" {
		definition.Binary = definition.Command
	}
	return definition, nil
}

func validateBackendDefinition(definition BackendDefinition) error {
	if definition.Name == "" {
		return fmt.Errorf("backend name is required")
	}
	if definition.Adapter == "" {
		return fmt.Errorf("backend adapter is required")
	}
	definition.Adapter = strings.ToLower(strings.TrimSpace(definition.Adapter))
	if definition.Adapter == "gemini" {
		definition.Adapter = "command"
		if definition.Binary == "" {
			definition.Binary = "gemini"
		}
	}
	switch definition.Adapter {
	case "opencode", "codex", "claude", "kimi", "command":
	default:
		return fmt.Errorf("unsupported adapter %q", definition.Adapter)
	}
	if definition.Adapter == "command" && strings.TrimSpace(definition.Binary) == "" {
		return fmt.Errorf("command adapter requires binary")
	}
	for _, raw := range definition.SupportedModels {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, err := filepath.Match(trimmed, "sample-text"); err != nil {
			return fmt.Errorf("invalid supported model pattern %q", trimmed)
		}
	}
	for _, capability := range definition.DistributedCaps {
		normalized, ok := supportedDistributedCapability(capability)
		if !ok {
			return fmt.Errorf("unsupported distributed capability %q", strings.TrimSpace(string(capability)))
		}
		_ = normalized
	}
	return nil
}

func normalizeBackendDefinition(definition BackendDefinition) BackendDefinition {
	definition.Name = normalizeBackend(definition.Name)
	definition.Adapter = strings.ToLower(strings.TrimSpace(definition.Adapter))
	definition.Binary = strings.TrimSpace(definition.Binary)
	definition.Command = strings.TrimSpace(definition.Command)
	if definition.Command != "" && definition.Binary == "" {
		definition.Binary = definition.Command
	}
	if definition.Adapter == "gemini" {
		definition.Adapter = "command"
		if definition.Binary == "" {
			definition.Binary = "gemini"
		}
	}

	definition.Args = normalizeStringSlice(definition.Args)
	definition.RequiredCredentials = normalizeStringSlice(definition.RequiredCredentials)
	definition.SupportedModels = normalizeStringSlice(definition.SupportedModels)
	definition.DistributedCaps = normalizeDistributedCaps(definition.DistributedCaps)
	return definition
}

func supportsModelPattern(patterns []string, model string) bool {
	if len(patterns) == 0 {
		return true
	}
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		return true
	}
	for _, pattern := range patterns {
		trimmedPattern := strings.TrimSpace(pattern)
		if trimmedPattern == "" {
			continue
		}
		matched, err := filepath.Match(trimmedPattern, trimmedModel)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDistributedCaps(caps []distributed.Capability) []distributed.Capability {
	if len(caps) == 0 {
		return nil
	}
	seen := map[distributed.Capability]struct{}{}
	out := make([]distributed.Capability, 0, len(caps))
	for _, raw := range caps {
		normalized, ok := supportedDistributedCapability(raw)
		if !ok {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func supportedDistributedCapability(value distributed.Capability) (distributed.Capability, bool) {
	normalized := distributed.Capability(strings.ToLower(strings.TrimSpace(string(value))))
	switch normalized {
	case distributed.CapabilityImplement, distributed.CapabilityReview, distributed.CapabilityRewriteTask, distributed.CapabilityLargerModel, distributed.CapabilityServiceProxy:
		return normalized, true
	default:
		return "", false
	}
}

func normalizeBackend(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	return value
}
