.PHONY: build test install

build:
	go build -o bin/codex-account ./cmd/codex-account

test:
	go test ./...

install:
	go install ./cmd/codex-account
