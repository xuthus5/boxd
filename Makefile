BUILD_TAGS ?= with_utls,with_quic
MODULE := github.com/xuthus5/boxd
VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build test race vet fmt fmt-check engine-version ci

build:
	go build -trimpath -tags "$(BUILD_TAGS)" -ldflags "$(LDFLAGS)" -o bin/boxd ./cmd/boxd

test:
	go test -tags "$(BUILD_TAGS)" ./...

race:
	go test -race -tags "$(BUILD_TAGS)" ./...

vet:
	go vet -tags "$(BUILD_TAGS)" ./...

fmt:
	gofmt -w cmd control internal kernel

fmt-check:
	@unformatted="$$(gofmt -l cmd control internal kernel webfiles.go)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; echo "$$unformatted"; exit 1; \
	fi

engine-version:
	go test ./internal/driver/singbox -count=1

ci: fmt-check vet test build
