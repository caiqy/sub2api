.PHONY: build generate test test-unit test-integration test-e2e

VERSION ?= $(shell ./scripts/resolve-version.sh)
LDFLAGS ?= -s -w -X main.Version=$(VERSION)

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

generate:
	go generate ./ent
	go generate ./cmd/server

test:
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 ./...
	golangci-lint run ./...

test-unit:
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=unit ./...

test-integration:
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=integration ./...

test-e2e:
	./scripts/e2e-test.sh

test-e2e-local:
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=e2e -v -timeout=300s ./internal/integration/...
