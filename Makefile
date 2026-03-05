test:
	go test ./...

smoke-agent-tui:
	go test ./cmd/yolo-agent ./cmd/yolo-tui
	$(MAKE) smoke-config-commands

smoke-config-commands:
	go test ./cmd/yolo-agent -run 'TestE2E_ConfigCommands_(InitThenValidateHappyPath|ValidateMissingFileFallsBackToDefaults|ValidateInvalidValuesReportsDeterministicDiagnostics|ValidateMissingAuthEnvReportsRemediation)$$' -count=1

smoke-event-stream:
	$(MAKE) smoke-agent-tui

release-gate-e8:
	go test ./cmd/yolo-agent -run 'TestE2E_(CodexTKConcurrency2LandsViaMergeQueue|ClaudeConflictRetryPathFinalizesWithLandingOrBlockedTriage|KimiLinearProfileProcessesAndClosesIssue|GitHubProfileProcessesAndClosesIssue)$$' -count=1
	go test ./internal/docs -run 'Test(MakefileDefinesE8ReleaseGateChecklistTarget|ReadmeDocumentsE8ReleaseGateChecklist|MigrationDocumentsE8ReleaseGateMigrationInstructions)$$' -count=1

RELEASE_WITH_GATE ?= 0

release-v2.4.0:
	@if [ -n "$$(git status --short)" ]; then \
		echo "release preflight failed: working tree is not clean"; \
		git status --short; \
		exit 1; \
	fi
	go test ./...
	go build ./...
	@if [ "$(RELEASE_WITH_GATE)" = "1" ]; then make release-gate-e8; fi
	@echo "Tagging:"
	@echo "git tag -a v2.4.0 -m \"Release v2.4.0\""
	@echo "git push origin v2.4.0"

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
MODULE   := github.com/egv/yolo-runner/v2/internal/version
LDFLAGS  := -s -w -X $(MODULE).Version=$(VERSION)
GOFLAGS  := -trimpath -ldflags '$(LDFLAGS)'

BINARIES := yolo-runner yolo-agent yolo-task yolo-tui yolo-linear-webhook yolo-linear-worker

build:
	mkdir -p bin
	$(foreach bin,$(BINARIES),go build $(GOFLAGS) -o bin/$(bin) ./cmd/$(bin);)

PREFIX ?= /usr/local

install: build
	install -d $(PREFIX)/bin
	$(foreach bin,$(BINARIES),install -m 755 bin/$(bin) $(PREFIX)/bin/$(bin);)
	@echo "installed $(BINARIES) to $(PREFIX)/bin"

uninstall:
	$(foreach bin,$(BINARIES),rm -f $(PREFIX)/bin/$(bin);)
	@echo "removed $(BINARIES) from $(PREFIX)/bin"
