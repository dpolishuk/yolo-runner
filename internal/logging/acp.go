package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ACPRequestEntry struct {
	LoggingSchemaFields
	IssueID     string `json:"issue_id"`
	RequestType string `json:"request_type"`
	Decision    string `json:"decision"`
	Message     string `json:"message,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Context     string `json:"context,omitempty"`
}

func AppendACPRequest(logPath string, entry ACPRequestEntry) error {
	if entry.Component == "" {
		entry.Component = "opencode"
	}
	entry.LoggingSchemaFields = populateRequiredLogFields(entry.LoggingSchemaFields, entry.IssueID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(payload, '\n'))
	return err
}
