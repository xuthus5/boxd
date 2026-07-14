.PHONY: dev build clean check-go check-ui check-embedded-ui

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
KERNEL_VERSION ?= 1.13.14
BUILD_TAGS := embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api

check-go:
	@./scripts/check-go.sh

check-ui:
	@cd ui && npm run check

check-embedded-ui:
	@./scripts/check-embedded-ui.sh

dev:
	@echo "Starting boxui..."
	@cd ui && npm run dev &
	@sleep 2
	@go run ./cmd/boxui/

build:
	@echo "Building frontend..."
	@cd ui && npm run build
	@find ui/dist -type d -exec chmod 0700 {} +
	@find ui/dist -type f -exec chmod 0600 {} +
	@echo "Copying frontend dist to cmd/boxui/ui/dist/ for Go embed..."
	@install -d -m 0700 cmd/boxui/ui
	@rm -rf cmd/boxui/ui/dist
	@cp -r ui/dist cmd/boxui/ui/dist
	@find cmd/boxui/ui -type d -exec chmod 0700 {} +
	@find cmd/boxui/ui -type f -exec chmod 0600 {} +
	@echo "Building binary..."
	@install -d -m 0700 bin
	@go build -tags "$(BUILD_TAGS)" -ldflags "-X github.com/xuthus5/boxui/internal/core.Version=$(VERSION) -X github.com/sagernet/sing-box/constant.Version=$(KERNEL_VERSION)" -o bin/boxui ./cmd/boxui/
	@chmod 0700 bin/boxui
	@echo "Cleaning up embed copy..."
	@rm -rf cmd/boxui/ui
	@echo "Built bin/boxui"

clean:
	@rm -rf bin/ ui/dist/ cmd/boxui/ui/
	@echo "Cleaned"
