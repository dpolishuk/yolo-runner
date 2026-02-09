test:
	go test ./...

smoke-agent-tui:
	go test ./cmd/yolo-agent ./cmd/yolo-tui

smoke-event-stream:
	$(MAKE) smoke-agent-tui

build:
	mkdir -p bin
	go build -o bin/yolo-runner ./cmd/yolo-runner
