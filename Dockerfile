# UI build
FROM node:24-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json* ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# Go build
FROM golang:1.26.6-alpine AS go-builder
WORKDIR /app
ARG VERSION=dev
ARG KERNEL_VERSION=1.13.21
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/dist ./cmd/boxd/ui/dist
RUN CGO_ENABLED=0 go build \
    -tags "embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
    -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${VERSION} -X github.com/sagernet/sing-box/constant.Version=${KERNEL_VERSION}" \
    -o /bin/boxd ./cmd/boxd/

# Runtime
FROM alpine:3.24
ARG VERSION=dev
LABEL org.opencontainers.image.source="https://github.com/xuthus5/boxd" \
      org.opencontainers.image.revision="${VERSION}"
RUN apk add --no-cache ca-certificates iptables iproute2 wget && \
    mkdir -p /var/lib/boxd /etc/sing-box && \
    chmod 0700 /var/lib/boxd /etc/sing-box
WORKDIR /app
ENV BOXD_CONTAINER=true
COPY --from=go-builder --chmod=0700 /bin/boxd /app/boxd
COPY --chmod=0700 deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
# ICMP 测速需要原始 socket：容器默认以 root 运行，需在 docker run 时
# 追加 --cap-add NET_RAW（见 README「Docker」章节）。
EXPOSE 9091 1080/tcp 1080/udp
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-proxy -q -O /dev/null http://127.0.0.1:9091/readyz || exit 1
ENTRYPOINT ["/app/docker-entrypoint.sh"]
