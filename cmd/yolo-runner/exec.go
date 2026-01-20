package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"yolo-runner/internal/opencode"
)

var commandOutput io.Writer = os.Stdout

var now = time.Now

func runCommand(args ...string) (string, error) {
	return runCommandWithOutput(commandOutput, args...)
}

func printCommand(out io.Writer, args []string) {
	if out == nil {
		return
	}
	fmt.Fprintln(out, "$ "+strings.Join(redactedCommand(args), " "))
}

func redactedCommand(args []string) []string {
	if len(args) >= 3 && args[0] == "opencode" && args[1] == "run" {
		redacted := append([]string{}, args...)
		redacted[2] = "<prompt redacted>"
		return redacted
	}
	return args
}

func printOutcome(out io.Writer, err error, elapsed time.Duration, command string) {
	if out == nil {
		return
	}
	status := "ok"
	exitCode := 0
	if err != nil {
		status = "failed"
		exitCode = exitCodeFromError(err)
	}
	fmt.Fprintf(out, "%s (exit=%d, elapsed=%s)\n", status, exitCode, formatElapsed(elapsed))
}

func formatElapsed(elapsed time.Duration) string {
	if elapsed < time.Millisecond {
		return "0ms"
	}
	return elapsed.Round(time.Millisecond).String()
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func runCommandWithOutput(out io.Writer, args ...string) (string, error) {
	start := now()
	printCommand(out, args)
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	elapsed := now().Sub(start)
	printOutcome(out, err, elapsed, strings.Join(args, " "))
	return string(output), err
}

type cmdProcess struct {
	cmd        *exec.Cmd
	stdoutFile *os.File
	stderrFile *os.File
	exitCode   int
	startTime  time.Time
	out        io.Writer
	command    string
}

func (process cmdProcess) Wait() error {
	err := process.cmd.Wait()
	if process.stdoutFile != nil {
		_ = process.stdoutFile.Close()
	}
	if process.stderrFile != nil {
		_ = process.stderrFile.Close()
	}
	printOutcome(process.out, err, now().Sub(process.startTime), process.command)
	return err
}

func (process cmdProcess) Kill() error {
	if process.cmd.Process == nil {
		if process.stdoutFile != nil {
			_ = process.stdoutFile.Close()
		}
		if process.stderrFile != nil {
			_ = process.stderrFile.Close()
		}
		return nil
	}
	err := process.cmd.Process.Kill()
	if process.stdoutFile != nil {
		_ = process.stdoutFile.Close()
	}
	if process.stderrFile != nil {
		_ = process.stderrFile.Close()
	}
	return err
}

func startCommandWithEnv(args []string, env map[string]string, stdoutPath string) (opencode.Process, error) {
	return startCommandWithEnvOutput(commandOutput, args, env, stdoutPath)
}

func startCommandWithEnvOutput(out io.Writer, args []string, env map[string]string, stdoutPath string) (opencode.Process, error) {
	start := now()
	printCommand(out, args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		printOutcome(out, err, now().Sub(start), strings.Join(args, " "))
		return nil, err
	}
	stderrPath := strings.TrimSuffix(stdoutPath, ".jsonl") + ".stderr.log"
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		_ = stdoutFile.Close()
		printOutcome(out, err, now().Sub(start), strings.Join(args, " "))
		return nil, err
	}
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	if err := cmd.Start(); err != nil {
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		printOutcome(out, err, now().Sub(start), strings.Join(args, " "))
		return nil, err
	}
	return cmdProcess{cmd: cmd, stdoutFile: stdoutFile, stderrFile: stderrFile, startTime: start, out: out, command: strings.Join(args, " ")}, nil
}
