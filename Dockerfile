# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETARCH

RUN target_arch="${TARGETARCH:-$(go env GOARCH)}"; \
    CGO_ENABLED=0 GOOS=linux GOARCH="${target_arch}" \
    go build -trimpath -tags "with_utls,with_quic" \
      -ldflags "-s -w \
        -X github.com/xuthus5/boxd/internal/version.Version=${VERSION} \
        -X github.com/xuthus5/boxd/internal/version.Commit=${COMMIT} \
        -X github.com/xuthus5/boxd/internal/version.Date=${BUILD_DATE}" \
      -o /out/boxd ./cmd/boxd

FROM gcr.io/distroless/static-debian12:latest

ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="boxd" \
      org.opencontainers.image.description="boxd control plane and sing-box data plane" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"

COPY --from=build /out/boxd /usr/local/bin/boxd

# TUN and route management require root plus NET_ADMIN. The default Mixed
# inbound can run without either when the container is not used for TUN.
USER 0:0
WORKDIR /var/lib/boxd
VOLUME ["/var/lib/boxd"]
EXPOSE 1080 7662
STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/boxd"]
CMD ["-data-dir", "/var/lib/boxd", "-web", "-listen", "[::]:7662", "-start"]
