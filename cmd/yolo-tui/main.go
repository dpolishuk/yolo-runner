package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/anomalyco/yolo-runner/internal/contracts"
	"github.com/anomalyco/yolo-runner/internal/ui/monitor"
)

func main() {
	os.Exit(RunMain(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func RunMain(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("yolo-tui", flag.ContinueOnError)
	fs.SetOutput(errOut)
	eventsStdin := fs.Bool("events-stdin", true, "Read NDJSON events from stdin")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if !*eventsStdin {
		fmt.Fprintln(errOut, "--events-stdin must be enabled")
		return 1
	}
	if in == nil {
		fmt.Fprintln(errOut, "stdin reader is required")
		return 1
	}

	if err := renderFromReader(in, out, errOut); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

func renderFromReader(reader io.Reader, out io.Writer, errOut io.Writer) error {
	decoder := contracts.NewEventDecoder(reader)
	model := monitor.NewModel(nil)
	haveEvents := false
	decodeFailures := 0
	for {
		event, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			decodeFailures++
			haveEvents = true
			model.Apply(contracts.Event{Type: contracts.EventTypeRunnerWarning, Message: "decode_error: " + err.Error()})
			if _, writeErr := io.WriteString(out, model.View()); writeErr != nil {
				return writeErr
			}
			if errOut != nil {
				_, _ = io.WriteString(errOut, "event decode warning: "+err.Error()+"\n")
			}
			if decodeFailures >= 3 {
				return fmt.Errorf("failed to decode event stream after %d errors: %w", decodeFailures, err)
			}
			continue
		}
		decodeFailures = 0
		haveEvents = true
		model.Apply(event)
		if _, writeErr := io.WriteString(out, model.View()); writeErr != nil {
			return writeErr
		}
	}
	if !haveEvents {
		if _, writeErr := io.WriteString(out, model.View()); writeErr != nil {
			return writeErr
		}
	}
	return nil
}
