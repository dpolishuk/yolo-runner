package main

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatLinearSessionActionableErrorClassifiesWrappedProcessorFailures(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		category  string
		contains  string
		notReason string
	}{
		{
			name:      "runtime wrapped by queued job context",
			err:       errors.New("process queued job \"job-1\": run linear session job: opencode stall while waiting for output"),
			category:  "runtime",
			contains:  "opencode stall",
			notReason: "Category: webhook",
		},
		{
			name:      "auth wrapped by queued job context",
			err:       errors.New("process queued job \"job-2\": emit linear response activity: agent activity mutation http 401: unauthorized"),
			category:  "auth",
			contains:  "401",
			notReason: "Category: webhook",
		},
		{
			name:      "graphql wrapped by queued job context",
			err:       errors.New("process queued job \"job-3\": transition delegated Linear issue \"iss-1\" to started: query delegated issue workflow state: graphql errors: Cannot query field \"foo\""),
			category:  "graphql",
			contains:  "graphql errors",
			notReason: "Category: webhook",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := FormatLinearSessionActionableError(tc.err)
			if !strings.Contains(formatted, "Category: "+tc.category) {
				t.Fatalf("expected category %q, got %q", tc.category, formatted)
			}
			if !strings.Contains(formatted, "Cause: ") || !strings.Contains(formatted, tc.contains) {
				t.Fatalf("expected root cause details in formatted message, got %q", formatted)
			}
			if !strings.Contains(formatted, "Next step: ") {
				t.Fatalf("expected remediation guidance, got %q", formatted)
			}
			if strings.Contains(formatted, tc.notReason) {
				t.Fatalf("expected non-webhook category for wrapped error, got %q", formatted)
			}
		})
	}
}
