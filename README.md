# boxd

sing-box 单节点控制平面（control plane）。提供 Web 面板管理内核配置、订阅节点、路由/DNS 策略与运行观测。

> 当前处于 PoC 阶段，接口与数据模型可能随开发调整。

## 功能

- **仪表盘**：内核启停/重启、实时上下行流量、内存、最近日志
- **代理配置**：入站 / 出站结构化表单编辑（含 TLS/Reality、传输等），保留高级 JSON
- **流量策略**：路由规则与规则集、DNS 服务器与规则；规则名称/描述独立持久化
- **节点与订阅**：订阅分组卡片、导入单节点、节点/分组 TCP·HTTP·ICMP 测速
- **运行观测**：内核日志、应用日志、活跃连接
- **高级配置**：Endpoints、Experimental、完整内核 JSON
- **应用设置**：主题、中英文、账号密码与 JWT 轮换、测速地址、URLTest 全局默认、日志级别

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
2. **订阅 / 节点**：添加订阅 URL 或导入单节点；按需配置 URLTest（可继承全局默认）。
3. **入站 / 出站**：创建本地代理入站；出站可绑定订阅组 selector / urltest，或直连/阻断等。
4. **路由 / DNS**：用表单维护规则；可一键安装常用默认规则与规则集。
5. **仪表盘**：启动内核；观察流量与日志。
6. **设置**：主题、语言、最低日志级别、系统测速地址、内核自启等。

### 预置路由能力

路由页可一键安装常见规则（嗅探、劫持 DNS、绕过局域网/ICMP、屏蔽 QUIC/广告、中国域名/IP 分流等）。规则集默认包含 Loyalsoldier 文本规则集（本地转换）与 SagerNet 二进制规则集（远程缓存）。

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
| `BOXD_CONFIG` | `--config` | `/etc/sing-box/config.json` | 内核配置路径 |
| `BOXD_DATA_DIR` | `--data-dir` | `/var/lib/boxd` | 数据目录（库、备份、规则集缓存等） |
| `BOXD_USERNAME` | `--username` | `admin` | 登录用户名 |
| `BOXD_PASSWORD` | `--password` | `admin123` | 仅首次初始化密码；库中已有哈希后不覆盖 |
| `BOXD_LOG_LEVEL` | `--log-level` | `info` | 应用日志级别 |
| `BOXD_REFRESH_INTERVAL` | `--refresh-interval` | `60` | 订阅刷新间隔（分钟） |
| `BOXD_TLS_CERT` | `--tls-cert` | - | TLS 证书路径 |
| `BOXD_TLS_KEY` | `--tls-key` | - | TLS 私钥路径 |
| `BOXD_CORS_ALLOWED_ORIGINS` | - | - | CORS 允许源，逗号分隔 |

### 认证说明

- 密码：Argon2id 存入 bbolt；优先级为「库中哈希 → 首次 `BOXD_PASSWORD` → `admin123`」。
- JWT：首次启动自动生成密钥写入数据库；设置页可轮换，轮换后全部会话失效。
- 默认密码状态下面板会持续告警并引导修改。

## 部署

### systemd（推荐）

```bash
# 1. 构建
make build

# 2. 系统用户与目录
useradd --system --home /var/lib/boxd --shell /sbin/nologin boxd || true
install -d -o boxd -g boxd -m 0700 /var/lib/boxd
install -d -o root -g boxd -m 0750 /etc/boxd
install -d -o boxd -g boxd -m 0750 /etc/sing-box

# 3. 安装二进制与单元
install -o root -g boxd -m 0750 bin/boxd /usr/local/bin/boxd
install -m 0644 deploy/boxd.service /etc/systemd/system/boxd.service
install -o root -g boxd -m 0640 deploy/boxd.env.example /etc/boxd/boxd.env
# 编辑 /etc/boxd/boxd.env，至少设置 BOXD_PASSWORD

# 4. 启动
systemctl daemon-reload
systemctl enable --now boxd.service
systemctl status boxd.service
```

单元文件见 [`deploy/boxd.service`](deploy/boxd.service)，环境示例见 [`deploy/boxd.env.example`](deploy/boxd.env.example)。

权限约定：

- 二进制：`root:boxd` `0750`
- 数据目录 `/var/lib/boxd`：`0700`
- 配置文件：`0600` / `0640`（按实际共享需求）

### Docker

```bash
docker build -t boxd .
docker run --name boxd -p 9091:9091 \
  -e BOXD_PASSWORD='your-strong-password' \
  -v boxd-data:/var/lib/boxd \
  -v boxd-config:/etc/sing-box \
  boxd
```

### TLS

内置 TLS：

```bash
BOXD_TLS_CERT=/etc/boxd/tls/fullchain.pem \
BOXD_TLS_KEY=/etc/boxd/tls/privkey.pem \
/usr/local/bin/boxd
```

或由 Caddy / Nginx / Traefik 终止 TLS，上游仅监听 `127.0.0.1:9091`，并透传 WebSocket/SSE。

更完整的发布门禁与回滚见 [docs/boxd/release-checklist.md](docs/boxd/release-checklist.md) 与 [docs/operations.md](docs/operations.md)。

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
# 产出 release/boxd_v0.1.0_linux_amd64.tar.gz 及 sha256
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
