BUILD_TAGS ?= with_utls,with_quic
MODULE := github.com/xuthus5/boxd
GO_PACKAGES := $(shell go list ./... | grep -v '/node_modules/')
VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build build-desktop build-desktop-all test race vet fmt fmt-check engine-version ci

build:
	go build -trimpath -tags "$(BUILD_TAGS)" -ldflags "$(LDFLAGS)" -o bin/boxd ./cmd/boxd

build-desktop:
	go build -trimpath -tags "$(BUILD_TAGS)" -ldflags "$(LDFLAGS)" -o bin/boxd-desktop ./cmd/boxd-desktop

build-desktop-all:
	@mkdir -p dist/desktop
	@set -eu; \
	for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		os=$${target%/*}; arch=$${target#*/}; ext=""; platform_ldflags=""; \
		if [ "$${os}" = "windows" ]; then ext=".exe"; platform_ldflags="-H=windowsgui"; fi; \
		echo "building desktop $$os/$$arch"; \
		CGO_ENABLED=0 GOOS="$${os}" GOARCH="$${arch}" \
			go build -trimpath -tags "$(BUILD_TAGS)" -ldflags "$(LDFLAGS) $${platform_ldflags}" \
			-o "dist/desktop/boxd-desktop-$${os}-$${arch}$${ext}" ./cmd/boxd-desktop; \
	done

test:
	go test -tags "$(BUILD_TAGS)" $(GO_PACKAGES)

race:
	go test -race -tags "$(BUILD_TAGS)" $(GO_PACKAGES)

vet:
	go vet -tags "$(BUILD_TAGS)" $(GO_PACKAGES)

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
