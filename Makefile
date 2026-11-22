.PHONY: build test race vet fmt engine-version

build:
	go build -o bin/boxd ./cmd/boxd

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd control internal kernel

engine-version:
	go test ./internal/driver/singbox -count=1
