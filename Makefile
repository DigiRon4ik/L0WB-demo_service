.DEFAULT_GOAL := run
.PHONY: run lint

lint:
	@golangci-lint run

run: lint
	@go run cmd/demoservice/main.go

send:
	@go run cmd/send/main.go