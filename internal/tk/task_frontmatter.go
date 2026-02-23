package tk

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type TicketFrontmatterConfig struct {
	Model      string
	Backend    string
	Skillset   string
	Tools      []string
	Mode       string
	HasTimeout bool
	Timeout    time.Duration
}

var ticketFrontmatterSupportedBackends = []string{"opencode", "codex", "claude", "kimi"}
var ticketFrontmatterSupportedModes = []string{"implement", "review"}

func ParseTicketFrontmatterConfig(raw string) (TicketFrontmatterConfig, error) {
	raw = strings.TrimSpace(raw)
	config := TicketFrontmatterConfig{}
	if raw == "" {
		return config, nil
	}

	values := make(map[string]any)
	if err := yaml.Unmarshal([]byte(raw), &values); err != nil {
		return config, fmt.Errorf("frontmatter must be valid YAML: %v", err)
	}

	issues := []string{}

	if model, ok := values["model"]; ok {
		value, ok := model.(string)
		if !ok {
			issues = append(issues, "model must be a string")
		} else if strings.TrimSpace(value) == "" {
			issues = append(issues, "model must be a non-empty string")
		} else {
			config.Model = strings.TrimSpace(value)
		}
	}

	if backend, ok := values["backend"]; ok {
		value, ok := backend.(string)
		if !ok {
			issues = append(issues, "backend must be a string")
		} else {
			value = strings.ToLower(strings.TrimSpace(value))
			if !contains(ticketFrontmatterSupportedBackends, value) {
				issues = append(issues, "backend must be one of: "+strings.Join(ticketFrontmatterSupportedBackends, ", "))
			} else {
				config.Backend = value
			}
		}
	}

	if skillset, ok := values["skillset"]; ok {
		value, ok := skillset.(string)
		if !ok {
			issues = append(issues, "skillset must be a string")
		} else if strings.TrimSpace(value) == "" {
			issues = append(issues, "skillset must be a non-empty string")
		} else {
			config.Skillset = strings.TrimSpace(value)
		}
	}

	if tools, ok := values["tools"]; ok {
		rawTools, ok := tools.([]any)
		if !ok {
			issues = append(issues, "tools must be an array")
		} else {
			for _, rawTool := range rawTools {
				tool, ok := rawTool.(string)
				if !ok {
					issues = append(issues, "tools must be an array of strings")
					break
				}
				trimmedTool := strings.TrimSpace(tool)
				if trimmedTool == "" {
					issues = append(issues, "tools entries must be non-empty strings")
					break
				}
				config.Tools = append(config.Tools, trimmedTool)
			}
		}
	}

	if timeout, ok := values["timeout"]; ok {
		value, ok := timeout.(string)
		if !ok {
			issues = append(issues, "timeout must be a duration string")
		} else {
			trimmedValue := strings.TrimSpace(value)
			if trimmedValue == "" {
				issues = append(issues, "timeout must be a duration string")
			} else {
				parsedTimeout, parseErr := time.ParseDuration(trimmedValue)
				if parseErr != nil {
					issues = append(issues, "timeout must be a valid duration (for example 30s, 5m)")
				} else if parsedTimeout < 0 {
					issues = append(issues, "timeout must be greater than or equal to 0")
				} else {
					config.Timeout = parsedTimeout
					config.HasTimeout = true
				}
			}
		}
	}

	if mode, ok := values["mode"]; ok {
		value, ok := mode.(string)
		if !ok {
			issues = append(issues, "mode must be a string")
		} else if !contains(ticketFrontmatterSupportedModes, strings.TrimSpace(strings.ToLower(value))) {
			issues = append(issues, "mode must be one of: "+strings.Join(ticketFrontmatterSupportedModes, ", "))
		} else {
			config.Mode = strings.TrimSpace(strings.ToLower(value))
		}
	}

	if len(issues) > 0 {
		return config, fmt.Errorf("frontmatter validation failed: %s", strings.Join(issues, "; "))
	}

	return config, nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

