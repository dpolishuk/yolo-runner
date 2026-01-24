package main

import (
	"bufio"
	"io"
	"strings"
)

func cleanupConfirmPrompt(summary string, input io.Reader, output io.Writer) (bool, error) {
	if output == nil {
		output = io.Discard
	}
	if input == nil {
		input = strings.NewReader("")
	}
	prompt := strings.TrimSpace(summary)
	if idx := strings.LastIndex(summary, "\n"); idx != -1 {
		prompt = strings.TrimSpace(summary[idx+1:])
	}
	if prompt == "" {
		prompt = "Discard these changes? [y/N]"
	}
	_, _ = io.WriteString(output, prompt+" ")

	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer == "y" || answer == "yes" {
		return true, nil
	}
	return false, nil
}
