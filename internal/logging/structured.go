package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type logLevel int

const (
	logLevelDebug logLevel = iota
	logLevelInfo
	logLevelWarn
	logLevelError
)

type StructuredLogger struct {
	w        io.Writer
	minLevel logLevel
	defaults LoggingSchemaFields
}

// NewStructuredLogger returns a logger that writes structured JSON lines to w.
func NewStructuredLogger(w io.Writer, minLevel string, defaults LoggingSchemaFields) *StructuredLogger {
	if w == nil {
		return &StructuredLogger{w: nil, minLevel: parseLevelOrDefault(minLevel), defaults: populateRequiredLogFields(defaults, defaults.TaskID)}
	}
	return &StructuredLogger{w: w, minLevel: parseLevelOrDefault(minLevel), defaults: populateRequiredLogFields(defaults, defaults.TaskID)}
}

// Log writes a single structured JSON line when level passes the configured threshold.
func (l *StructuredLogger) Log(level string, fields map[string]interface{}) error {
	if l == nil || l.w == nil {
		return nil
	}

	entryLevel := normalizeLogLevel(level)
	entrySeverity, ok := parseLogLevel(entryLevel)
	if !ok {
		return fmt.Errorf("invalid log level %q", level)
	}

	if entrySeverity < l.minLevel {
		return nil
	}

	entry := map[string]interface{}{}
	for key, value := range fields {
		entry[key] = value
	}

	entry["timestamp"] = l.defaults.Timestamp
	entry["level"] = entryLevel
	entry["component"] = chooseField(entry["component"], l.defaults.Component)
	entry["task_id"] = chooseField(entry["task_id"], l.defaults.TaskID)
	entry["run_id"] = chooseField(entry["run_id"], l.defaults.RunID)

	if ts, ok := entry["timestamp"].(string); !ok || strings.TrimSpace(ts) == "" {
		entry["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.w.Write(append(payload, '\n'))
	return err
}

func parseLevelOrDefault(raw string) logLevel {
	parsed, ok := parseLogLevel(normalizeLogLevel(raw))
	if !ok {
		return logLevelInfo
	}
	return parsed
}

func normalizeLogLevel(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func parseLogLevel(raw string) (logLevel, bool) {
	switch raw {
	case "debug":
		return logLevelDebug, true
	case "info":
		return logLevelInfo, true
	case "warn":
		return logLevelWarn, true
	case "warning":
		return logLevelWarn, true
	case "error":
		return logLevelError, true
	default:
		return 0, false
	}
}

func chooseField(raw interface{}, fallback string) string {
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
