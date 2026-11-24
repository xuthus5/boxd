BUILD_TAGS ?= with_utls,with_quic

.PHONY: build test race vet fmt engine-version

build:
	go build -tags "$(BUILD_TAGS)" -o bin/boxd ./cmd/boxd

test:
	go test -tags "$(BUILD_TAGS)" ./...

race:
	go test -race -tags "$(BUILD_TAGS)" ./...

vet:
	go vet -tags "$(BUILD_TAGS)" ./...

fmt:
	gofmt -w cmd control internal kernel

engine-version:
	go test ./internal/driver/singbox -count=1
