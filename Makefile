LDFLAGS := -X github.com/groundsgg/grounds-cli/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
           -X github.com/groundsgg/grounds-cli/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) \
           -X github.com/groundsgg/grounds-cli/internal/version.BuildAt=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "$(LDFLAGS)" -o ./bin/grounds ./cmd/grounds

test:
	go test ./... -race -cover

vet:
	go vet ./...

.PHONY: build test vet
