package main

import (
	"fmt"
	"strings"
)

type CommandCall struct {
	Name string
	Args []string
}

type FakeRunner struct {
	calls []CommandCall
	stubs map[string][]byte
}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		stubs: make(map[string][]byte),
	}
}

func (f *FakeRunner) Script(name string, args []string, output []byte) {
	f.stubs[stubKey(name, args)] = output
}

func (f *FakeRunner) Run(name string, args ...string) ([]byte, error) {
	call := CommandCall{Name: name, Args: append([]string(nil), args...)}
	f.calls = append(f.calls, call)
	key := stubKey(name, args)
	output, ok := f.stubs[key]
	if !ok {
		return nil, fmt.Errorf("missing stub for command %s %s", name, strings.Join(args, " "))
	}
	return output, nil
}

func (f *FakeRunner) Calls() []CommandCall {
	return append([]CommandCall(nil), f.calls...)
}

func stubKey(name string, args []string) string {
	return fmt.Sprintf("%s\x00%s", name, strings.Join(args, "\x00"))
}
