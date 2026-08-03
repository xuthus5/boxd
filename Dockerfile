# UI build
FROM node:20-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json* ./
RUN npm ci || npm install
COPY ui/ ./
RUN npm run build

# Go build
FROM golang:1.26.5-alpine AS go-builder
WORKDIR /app
ARG VERSION=dev
ARG KERNEL_VERSION=1.13.14
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/dist ./cmd/boxd/ui/dist
RUN CGO_ENABLED=0 go build \
    -tags "embed_ui with_gvisor with_quic with_dhcp with_wireguard with_utls with_acme with_clash_api" \
    -ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${VERSION} -X github.com/sagernet/sing-box/constant.Version=v${KERNEL_VERSION}" \
    -o /bin/boxd ./cmd/boxd/

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates iptables iproute2 wget && \
    mkdir -p /var/lib/boxd /etc/sing-box && \
    chmod 0700 /var/lib/boxd
WORKDIR /app
COPY --from=go-builder /bin/boxd /app/boxd
COPY --chmod=0700 deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
# ICMP 测速需要原始 socket：容器默认以 root 运行，需在 docker run 时
# 追加 --cap-add NET_RAW（见 README「Docker」章节）。
EXPOSE 9091
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:9091/healthz || exit 1
ENTRYPOINT ["/app/docker-entrypoint.sh"]
