# boxd

[English](README.md) | [简体中文](README.zh-CN.md)

sing-box 单节点控制平面（control plane）。提供 Web 面板管理内核配置、订阅节点、路由/DNS 策略与运行观测。

> 当前处于 PoC 阶段，接口与数据模型可能随开发调整。

## 功能

- **配置安全**：dry-run、保存与历史恢复会在写盘或重启前阻断路径级 sing-box 语义/拓扑错误；成功应用保留有界本地快照、路径级恢复差异、校验、重启与失败自动回滚
- **仪表盘**：内核启停/重启、面板就绪状态与失败重试、快速上手清单、sing-box 语义配置诊断（出站/DNS 跨层启动拓扑、特性摘要、1.14 弃用迁移预警与编辑器深链）、规则集文件/缓存新鲜度、按新鲜度更新内置规则集、逐次自动更新诊断与批量刷新、运行健康（失败订阅/不稳节点/配置应用失败告警）、配置应用时间线、全局出口/Clash 模式（延迟诊断）、实时流量/最近日志、内存
- **代理配置**：入站 / 出站结构化表单、列表搜索、复制 tag、故障闭环 deep-link（连接/日志/节点），高级 JSON 路径级差异、编辑对话框 dry-run 校验（失败可跳转路径）
- **流量策略**：路由/DNS 规则搜索、取反快捷开关、sing-box 1.13 接口感知地址匹配、直接逻辑子规则结构化编辑与深层 JSON 兜底、Inline headless rule 结构化编辑、移动端更紧凑的规则卡片、DNS 服务器搜索；规则名称/描述独立持久化；规则集更新错误码诊断；应用前 dry-run 校验；规则/服务器路径深链打开对话框，支持对话框内 dry-run 校验
- **节点与订阅**：按订阅间隔后台自动刷新、Clash YAML / base64 订阅、16 MiB 下载上限、仅允许公网 HTTP(S) 订阅源并拦截本机/私网地址与不安全重定向、失败优先与重试、刷新/配置同步诊断、流量/到期、订阅卡片 deep-link 到节点/日志、移动端更紧凑的工具栏/卡片、节点搜索、延迟色阶、运行时分组、测速（错误码提示与可复制诊断）
- **运行观测**：内核/应用日志带时间戳、复制/导出、代理友好的 SSE 心跳与手动重连；活跃连接支持实时上下行速率、搜索/排序/导出、乐观关闭、分组视图与关闭错误诊断
- **高级配置**：Endpoints（含 WireGuard Peer 可视化编辑）、X.509 证书信任库、sing-box CCM/OCM/DERP/Resolved/SSM API 服务、内核日志配置、NTP 时间同步、Experimental、完整内核 JSON（路径级 diff）、本地 JSON 导入/导出、只读引用与 DNS/出站自举拓扑预检、配置历史审计/恢复；raw/endpoints/certificate/services/log/NTP/experimental dry-run 校验；保存错误跳转
- **认证**：登录失败展示错误码/可操作提示与一键复制诊断
- **页面加载诊断**：仪表盘/代理/策略/节点订阅/设置/高级配置等页面的查询失败统一展示错误码、可操作提示、复制诊断与重试
- **应用设置**：主题、中英文与日志级别入库持久化；账号密码与 JWT 轮换、脏状态保存门禁、测速地址、URLTest 全局默认、脱敏诊断支持包与面板备份导出

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go、chi、bbolt、sing-box（静态链接） |
| 前端 | React 19、TypeScript、Vite、shadcn/ui、Tailwind CSS |
| 认证 | JWT（HS256），密码 Argon2id |

## 快速开始

### 要求

- Go 1.26+
- Node.js 20+
- Linux（生产推荐）；本地开发可用当前环境

### 一键构建并运行

```bash
make build
BOXD_PASSWORD='your-strong-password' \
  BOXD_DATA_DIR=./data \
  BOXD_CONFIG=./data/config.json \
  ./bin/boxd
```

浏览器打开 `http://127.0.0.1:9091`，默认用户名 `admin`。首次使用默认密码时会强制进入设置页完成密码轮换。

### 开发模式（前后端分离）

```bash
# 终端 1：前端 Vite（默认 http://127.0.0.1:5173）
cd ui && npm ci && npm run dev

# 终端 2：后端 API（默认 [::]:9091）
export BOXD_PASSWORD='dev-password'
export BOXD_DATA_DIR="$PWD/data"
export BOXD_CONFIG="$PWD/data/config.json"
export BOXD_CORS_ALLOWED_ORIGINS='http://127.0.0.1:5173,http://localhost:5173'
go run ./cmd/boxd/
```

也可使用 `make dev`（会后台启动前端并运行后端，适合快速试跑）。

## 使用说明

1. 登录面板，轮换管理员密码。
2. **订阅 / 节点**：添加公网 HTTP(S) 订阅 URL，或导入 VMess、VLESS、Trojan、Shadowsocks/SIP002、SSR、Hysteria/Hysteria2、TUIC、AnyTLS、ShadowTLS 链接。本机/私网源和不安全重定向会被拦截。订阅会按各自间隔后台刷新，全局间隔作为旧数据回退。按需配置 URLTest（可继承全局默认）。订阅下载限制为 16 MiB，刷新或配置同步失败会展示可操作错误码。
3. **入站 / 出站**：可一键安装 mixed（1080）与 TUN 模板，或自建入站；出站可绑定订阅组 selector / urltest，或直连/阻断等。
4. **路由 / DNS / 证书 / Services / 内核日志 / NTP / Experimental**：用表单维护规则、信任库、内核辅助服务、日志输出和时间同步；可一键安装默认规则与 Clash API。
5. **仪表盘**：先确认面板就绪状态，再启动内核；切换全局出口与 Clash 模式；观察流量与日志。
6. **设置**：主题、语言、最低日志级别（写入数据库）、系统测速地址、内核自启、脱敏诊断支持包与备份导出等。

### 预置路由能力

路由页可一键安装常见规则（嗅探、劫持 DNS、绕过局域网/ICMP、屏蔽 QUIC/广告、中国域名/IP 分流等）。规则集默认包含 Loyalsoldier 文本规则集（本地转换）与 SagerNet 二进制规则集（远程缓存）；手动下载仅允许公网 HTTP(S) 地址，且限制为 16 MiB。

### 备份与恢复

```bash
./bin/boxd --backup /var/backups/boxd/boxd-$(date +%F).tar.gz
systemctl stop boxd.service   # 若以服务运行
./bin/boxd --restore /var/backups/boxd/boxd-YYYY-MM-DD.tar.gz \
  --data-dir /var/lib/boxd --config /etc/sing-box/config.json
systemctl start boxd.service
```

归档内数据库文件名为 `boxd.db`。

## 配置

| 环境变量 | Flag | 默认值 | 说明 |
| --- | --- | --- | --- |
| `BOXD_LISTEN` | `--listen` | `[::]:9091` | 监听地址（优先于 `BOXD_PORT`） |
| `BOXD_PORT` | - | `9091` | 仅端口时使用 |
| `BOXD_CONFIG` | `--config` | `/etc/sing-box/config.json`（Linux）；`%ProgramData%\sing-box\config.json`（Windows） | 内核配置路径 |
| `BOXD_DATA_DIR` | `--data-dir` | `/var/lib/boxd`（Linux）；`%ProgramData%\boxd`（Windows） | 数据目录（库、备份、规则集缓存等） |
| `BOXD_USERNAME` | `--username` | `admin` | 登录用户名 |
| `BOXD_PASSWORD` | `--password` | `admin123` | 仅首次初始化密码；库中已有哈希后不覆盖 |
| `BOXD_LOG_LEVEL` | `--log-level` | `info` | 应用日志级别 |
| `BOXD_REFRESH_INTERVAL` | `--refresh-interval` | `60` | 旧订阅的回退刷新间隔（分钟） |
| `BOXD_TLS_CERT` | `--tls-cert` | - | TLS 证书路径 |
| `BOXD_TLS_KEY` | `--tls-key` | - | TLS 私钥路径 |
| `BOXD_CORS_ALLOWED_ORIGINS` | - | - | CORS 允许源，逗号分隔 |

### 认证说明

- 密码：Argon2id 存入 bbolt；优先级为「库中哈希 → 首次 `BOXD_PASSWORD` → `admin123`」。
- JWT：首次启动自动生成密钥写入数据库；设置页可轮换，轮换后全部会话失效。
- 默认密码状态下面板会持续告警并引导修改。

## 从二进制归档安装

Nightly 与正式版归档同时提供 `linux/amd64` 与 `linux/arm64`。常见内容：

- `boxd` — 二进制
- `boxd.service` — systemd 单元
- `boxd.env.example` — 可选环境变量模板（仓库中存在时才会打进包）
- 文档 / 许可证

boxd 进程**不会**自动加载 `.env` 文件。配置来源只有：

1. 进程环境变量（`BOXD_*`）
2. CLI 参数
3. 在 systemd 下由 `boxd.service` 的 `EnvironmentFile=-/etc/boxd/boxd.env` 注入（前缀 `-` 表示文件可不存在）

### systemd 安装（推荐）

```bash
# 1. 解压
tar -xzf boxd_v0.1.0_linux_amd64.tar.gz -C /tmp/boxd-release   # or linux_arm64
cd /tmp/boxd-release

# 2. 系统用户与目录
useradd --system --home /var/lib/boxd --shell /sbin/nologin boxd || true
install -d -o boxd -g boxd -m 0700 /var/lib/boxd
install -d -o root -g boxd -m 0750 /etc/boxd
install -d -o boxd -g boxd -m 0750 /etc/sing-box

# 3. 二进制 + 单元 + 环境文件
install -o root -g boxd -m 0750 ./boxd /usr/local/bin/boxd
install -m 0644 ./boxd.service /etc/systemd/system/boxd.service
if [[ -f ./boxd.env.example ]]; then
  install -o root -g boxd -m 0640 ./boxd.env.example /etc/boxd/boxd.env
  # 至少设置 BOXD_PASSWORD，并检查其他项
  ${EDITOR:-vi} /etc/boxd/boxd.env
else
  printf '%s\n' 'BOXD_PASSWORD=your-strong-password' > /etc/boxd/boxd.env
  chown root:boxd /etc/boxd/boxd.env
  chmod 0640 /etc/boxd/boxd.env
fi

# 4. 启动
systemctl daemon-reload
systemctl enable --now boxd.service
systemctl status boxd.service
```

浏览器访问 `http://127.0.0.1:9091`（或你的监听地址）。默认用户名 `admin`。

### 不使用 systemd 直接运行

```bash
export BOXD_PASSWORD='your-strong-password'
export BOXD_DATA_DIR=/var/lib/boxd
export BOXD_CONFIG=/etc/sing-box/config.json
/usr/local/bin/boxd
```

或使用参数：

```bash
/usr/local/bin/boxd \
  --data-dir /var/lib/boxd \
  --config /etc/sing-box/config.json \
  --password 'your-strong-password'
```

### ICMP 测速需要网络权限

ICMP 测速会打开原始套接字，需要 `CAP_NET_RAW` 能力。

- **systemd 安装**：`boxd.service` 已内置 `AmbientCapabilities=CAP_NET_RAW`，直接 `systemctl start boxd.service` 即可。若自建 unit，需自行加入 `AmbientCapabilities=CAP_NET_RAW` 与 `CapabilityBoundingSet=... CAP_NET_RAW`。
- **不使用 systemd 直接运行**：以 root 运行，或为服务用户授予该能力。否则 ICMP 测速会报 `icmp raw socket requires CAP_NET_RAW`（TCP/HTTP 测速不受影响）。

## Docker

公开镜像（CI 推送后）：

```bash
# main 的 nightly
docker pull ghcr.io/xuthus5/boxd:nightly

# 正式版
docker pull ghcr.io/xuthus5/boxd:latest
docker pull ghcr.io/xuthus5/boxd:v0.1.0
```

### Docker 运行

```bash
docker run -d --name boxd --restart unless-stopped \
  -p 9091:9091 \
  -e BOXD_PASSWORD='your-strong-password' \
  -e BOXD_LISTEN='[::]:9091' \
  -v boxd-data:/var/lib/boxd \
  -v boxd-config:/etc/sing-box \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  ghcr.io/xuthus5/boxd:latest
```

说明：

- 通过 `-e BOXD_*` 或 `--env-file` 传入配置；容器内进程不会自行读取 `/etc/boxd/boxd.env`。
- 请持久化 `/var/lib/boxd`（数据库、缓存）与 `/etc/sing-box`（内核配置）。
- 使用 TUN 等高级网络能力时建议加 `NET_ADMIN`。
- ICMP 测速需要 `NET_RAW`（原始套接字）。容器内进程以 root 运行，能力生效；不加则禁用 ICMP 测速。
- 健康检查访问容器内 `http://127.0.0.1:9091/healthz`。

### 本地构建镜像

```bash
docker build --build-arg VERSION=dev -t boxd:local .
docker run --rm -p 9091:9091 -e BOXD_PASSWORD='dev-password' boxd:local
```

## 部署

二进制安装请优先看 [从二进制归档安装](#从二进制归档安装)。
容器运行请优先看 [Docker](#docker)。

### TLS

内置 TLS：

```bash
BOXD_TLS_CERT=/etc/boxd/tls/fullchain.pem \
BOXD_TLS_KEY=/etc/boxd/tls/privkey.pem \
/usr/local/bin/boxd
```

或由 Caddy / Nginx / Traefik 终止 TLS，上游仅监听 `127.0.0.1:9091`，并透传 WebSocket/SSE。

更完整的发布门禁与回滚见 [docs/boxd/release-checklist.md](docs/boxd/release-checklist.md) 与 [docs/operations.md](docs/operations.md)。


## CI 产物与 Docker 镜像

| 触发条件 | 二进制 | Docker（GHCR） |
| --- | --- | --- |
| Pull Request | 仅质量门禁 | 构建冒烟（不推送） |
| 推送到 `main` | 滚动 **nightly** GitHub Release + Actions 产物（`boxd_nightly_linux_amd64/arm64.tar.gz`、`boxd_nightly_windows_amd64/arm64.tar.gz`、`boxd_nightly_darwin_amd64/arm64.tar.gz`；桌面：`boxd-desktop-windows`、`boxd-desktop-macos`） | 多架构 `ghcr.io/<owner>/boxd:nightly`（`linux/amd64`、`linux/arm64`） |
| 打 `v*` tag | 正式 GitHub Release + SBOM（`linux/amd64`、`linux/arm64`） | 多架构 `ghcr.io/<owner>/boxd:<tag>`、`:<version>`、`:latest` |

headless 二进制发布覆盖 **Linux**（`linux/amd64`、`linux/arm64`）、**Windows**（`windows/amd64`、`windows/arm64`）与 **macOS**（`darwin/amd64`、`darwin/arm64`），以 `.tar.gz` + sha256 校验和发布。桌面应用发布覆盖 **Linux**（deb/rpm/AppImage）、**Windows**（zip）与 **macOS**（zip，DMG 尽力生成）。

示例（替换 owner）：

```bash
# Nightly 镜像
docker pull ghcr.io/xuthus5/boxd:nightly

# 正式版镜像
docker pull ghcr.io/xuthus5/boxd:latest
docker pull ghcr.io/xuthus5/boxd:v0.1.0
```

Nightly 二进制发布在滚动 GitHub Release 标签 `nightly` 下，同时作为 workflow artifact 上传（保留 14 天）。正式版通过推送 `v*` tag 产出。

## 本地开发

### 日常命令

```bash
make build              # 前端构建 + 嵌入 + 产出 bin/boxd
make dev                # 简易联调
make clean              # 清理 bin 与 dist
make check-go           # Go 测试、race、覆盖率 ≥90%、lint、govulncheck
make check-ui           # 前端 typecheck/lint/coverage/build
make check-embedded-ui  # 嵌入资源完整性
```

前端目录：

```bash
cd ui
npm ci
npm run dev        # 开发服务器
npm run check      # 类型 / lint / 覆盖率 / 构建
npm run e2e:install && npm run e2e   # Playwright（Mock，不连生产 9091）
```

后端：

```bash
go run ./cmd/boxd/
go test ./...
golangci-lint run ./...
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./internal
goimports-reviser -rm-unused -set-alias -project-name github.com/xuthus5/boxd -recursive ./cmd
```

### 发布包

```bash
./scripts/package-release.sh v0.1.0 release
# 产出 release/boxd_v0.1.0_linux_{amd64,arm64}.tar.gz 及对应 sha256
```

推送 `v*` tag 后，GitHub Release workflow 会跑完整质量门禁并上传归档、SBOM。

### 运行时抽检

```bash
BOXD_PASSWORD='your-password' ./scripts/e2e-live.sh
BOXD_PASSWORD='your-password' ./scripts/soak-runtime.sh
```

## 许可证

boxd **自有代码**采用 [Apache License 2.0](LICENSE)。

正式发布的二进制**静态链接**了 GPL-3.0 的 sing-box / sing，分发二进制时还需同时遵守 [第三方声明](THIRD_PARTY_NOTICES.md) 中的 GPL-3.0 义务（含对应源码与构建信息）。本说明不构成法律意见。
