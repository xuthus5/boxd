.PHONY: dev build build-desktop clean check-go check-ui check-embedded-ui

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
KERNEL_VERSION ?= 1.13.18
BUILD_TAGS := embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api

check-go:
	@./scripts/check-go.sh

check-ui:
	@cd ui && npm run check

check-embedded-ui:
	@./scripts/check-embedded-ui.sh

dev:
	@echo "Starting boxd..."
	@cd ui && npm run dev &
	@sleep 2
	@CGO_ENABLED=0 go run ./cmd/boxd/

build:
	@echo "Building frontend..."
	@cd ui && npm run build
	@find ui/dist -type d -exec chmod 0700 {} +
	@find ui/dist -type f -exec chmod 0600 {} +
	@echo "Copying frontend dist to cmd/boxd/ui/dist/ for Go embed..."
	@install -d -m 0700 cmd/boxd/ui
	@rm -rf cmd/boxd/ui/dist
	@cp -r ui/dist cmd/boxd/ui/dist
	@find cmd/boxd/ui -type d -exec chmod 0700 {} +
	@find cmd/boxd/ui -type f -exec chmod 0600 {} +
	@echo "Building binary..."
	@install -d -m 0700 bin
	@CGO_ENABLED=0 go build -tags "$(BUILD_TAGS)" -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=$(VERSION) -X github.com/sagernet/sing-box/constant.Version=$(KERNEL_VERSION)" -o bin/boxd ./cmd/boxd/
	@chmod 0700 bin/boxd
	@echo "Cleaning up embed copy..."
	@rm -rf cmd/boxd/ui
	@echo "Built bin/boxd"

build-desktop:
	@./scripts/build-desktop.sh $(VERSION)

clean:
	@rm -rf bin/ ui/dist/ cmd/boxd/ui/ desktop/bin/ desktop/ui/ desktop/desktop
	@echo "Cleaned"
