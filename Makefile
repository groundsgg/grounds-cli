# GlitchTip DSN baked into release binaries. Public key only — safe
# to commit (Sentry's documented model: DSNs ship in client bundles).
# Override at runtime via GROUNDS_SENTRY_DSN env var, or disable with
# GROUNDS_SENTRY_DISABLE=1. Set SENTRY_DSN=… on the make line to point
# a build at a personal GlitchTip.
SENTRY_DSN ?= https://ea5b2e86-1c42-4a4c-a227-35c8d7652e2e@glitch.grounds.gg/1

LDFLAGS := -X github.com/groundsgg/grounds-cli/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
           -X github.com/groundsgg/grounds-cli/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) \
           -X github.com/groundsgg/grounds-cli/internal/version.BuildAt=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
           -X github.com/groundsgg/grounds-cli/internal/observability.DSN=$(SENTRY_DSN)

build:
	go build -ldflags "$(LDFLAGS)" -o ./bin/grounds ./cmd/grounds

test:
	go test ./... -race -cover

vet:
	go vet ./...

.PHONY: build test vet
