# boxd v2

boxd v2 是围绕 sing-box 重新实现的控制面基础工程。当前仓库完成了内核契约、单写入者状态机、控制面业务、SQLite 存储、组合根，以及内置的 HTTP/SSE Web 控制台；sing-box v1.14.4 Driver 已接入真实配置解析、实例生命周期与运行时能力。

## 当前结构

```text
cmd/boxd/                          headless / Web 宿主入口
kernel/                            稳定内核契约
control/                           稳定控制面契约
prototype/                         内嵌 Web 控制台静态资源
webfiles.go                        Web 资源 embed 入口
internal/kernel/manager/           与引擎无关的状态机
internal/kernel/fake/              契约测试用 Fake Engine
internal/kernel/events/            有界非阻塞事件分发
internal/kernel/logbuf/            最近日志环形缓冲
internal/controlplane/             control 契约实现
internal/adapters/outbound/singboxconfig/ 期望状态到 sing-box JSON 的编译器
internal/adapters/outbound/sqlite/ SQLite、迁移与仓储
internal/adapters/outbound/httpdownload/ 条件 HTTP 下载器
internal/adapters/outbound/subscriptionjson/ 订阅解析器（JSON 与分享链接）
internal/adapters/outbound/nodelink/ 节点分享链接编解码器
internal/adapters/inbound/httpapi/ REST、SSE、认证与静态 Web 控制台
internal/driver/singbox/           sing-box 唯一依赖边界与真实 Driver
internal/bootstrap/                唯一组合根
```

依赖方向：`cmd -> bootstrap -> controlplane / adapters / manager -> kernel`。公开包 `kernel` 和 `control` 不依赖 `internal`，sing-box 与 SQLite 驱动分别限制在各自适配器内。

## 已具备

- `Start / Stop / Restart / ApplyConfig` 单写入者状态机。
- 配置修订冲突检测、原子持久化、应用失败回滚和状态恢复。
- Fake Engine、契约测试和可替换 Factory。
- 有界事件总线、日志环、运行状态与配置事件。
- SQLite WAL、`busy_timeout=5000`、外键、版本化内嵌迁移。
- SQLite 配置历史、当前配置指针和设置仓储。
- Kernel、Config、Setup、Inbounds、Egress、Subscriptions、Nodes、Policies、Routes、DNS、RuleSets、Settings、Observability、Security、Audit 控制面用例。
- 期望状态快照、领域校验、原子落库和统一配置应用/回滚。
- Mixed `[::]:1080` 双栈入口与 TUN 入口模型。
- Node、Selector/URLTest Group、最多 4 跳 Chain 的代理编排。
- rules/global/direct 一键路由模式、常用预设和有序规则管理。
- 国内/国外/默认三槽位 DNS，支持 UDP、DoH、DoT 和固定直连 bootstrap。
- sing-box v1.14.x 配置编译、严格解析和真实内核启动测试。
- 唯一 bootstrap 装配、优雅关闭和 headless 命令。
- sing-box JSON 配置解析与校验、Box 创建、Start/Stop/Restart。
- 运行中配置采用重启式应用，失败时由内核管理器回滚数据库和运行状态。
- sing-box 日志桥接到内核日志缓冲区与事件总线。
- Selector/URLTest 出站分组查询和 Selector 切换。
- 连接观测、连接关闭、Clash 兼容模式、DNS 缓存刷写和真实出站延迟测量。
- 节点与入口流量按当前连接的 sing-box 真实字节增量聚合，并将采样间隔内关闭的连接计入本次运行统计。
- 路由规则命中数映射自 sing-box 实际匹配序号；DNS 槽位支持真实 UDP/DoH/DoT 往返探测。
- 订阅条件刷新、后台定时调度、稳定节点 ID、原子节点替换和节点测速历史。
- 订阅支持 sing-box JSON、Base64 JSON、明文分享链接列表和 Base64 分享链接列表。
- 节点支持单个编辑，以及单节点/多节点 URL Test、TCP Ping 和 ICMP Ping 测速。
- Shadowsocks、VMess、VLESS、Trojan、Hysteria2、SOCKS、HTTP 分享链接导入与导出。
- 内置/自定义规则集定义、下载字节数、刷新状态和预设依赖校验。
- 控制面变更、内核生命周期、认证和手动审计事件查询。
- 本地管理员 PBKDF2 口令与短期会话契约。
- 同源 Web 控制台、REST API 与 SSE 流式事件，默认监听 `[::]:7662`。
- 应用日志与 sing-box 内核日志在同一个日志页中流式查看。
- 首次初始化管理员口令；支持 `HttpOnly; SameSite=Strict` Cookie 和 Bearer Token 会话。
- 订阅、节点列表、规则集和连接凭据在 Web 视图中脱敏；节点编辑器在管理员显式打开时才加载协议参数。不提供 Web 终端、Shell 或任意命令执行。

## 引擎版本策略

当前锁定 `github.com/sagernet/sing-box v1.14.4`，兼容线为 `v1.14.x`。`go.mod` 是精确版本的事实来源；Driver 在打开引擎前会从 Go build info 读取实际模块版本，只允许同一 minor 线，补丁版本升级不需要修改代码。

补丁升级流程、次版本升级门槛和回退方式见 `docs/sing-box-versioning.md`。

## 快速验证

```text
go test -tags with_utls,with_quic ./...
go test -race -tags with_utls,with_quic ./...
make engine-version
go build -tags with_utls,with_quic -o bin/boxd ./cmd/boxd
go run -tags with_utls,with_quic ./cmd/boxd -data-dir ./tmp-data -web -listen [::]:7662
```

生产构建必须包含 `with_utls`，否则 VLESS Reality 等依赖 uTLS 的节点会在内核启动阶段失败；`with_quic` 用于 Hysteria2 等 QUIC 协议。`make build` 的默认构建标签为 `with_utls,with_quic`。

启动后访问 `http://127.0.0.1:7662/`。首次访问需要设置管理员口令并登录；之后可通过账号设置页退出会话。`-web` 启用浏览器控制台和 HTTP API，`-listen` 指定监听地址，默认是 `[::]:7662`。如果只想运行控制面而不暴露端口，可省略 `-web`。

首次启动会幂等写入默认 Mixed/TUN、路由和 DNS 期望状态，并生成保存在 SQLite 中的 sing-box 配置。`-start` 会启动真实 sing-box；如果默认 1080 端口已被其他代理客户端占用，启动会返回端口绑定错误。sing-box 会在数据目录创建内部 `cache.db`，它只用于引擎缓存，不是 boxd 的业务存储。

本阶段只接入控制面及以下的 Web 管理能力，不接入 Wails、MyGo、桌面宿主等上层应用；也不包含备份/恢复能力和 Web 终端。远程访问时请优先使用 TLS 反向代理或可信内网，并在首次部署后立即完成管理员口令初始化。

## 发布与部署

GitHub Actions 提供以下流程：

- `ci.yml`：对提交和 Pull Request 执行 `gofmt`、`go vet`、普通测试、竞态测试、构建验证和部署脚本 ShellCheck。
- `release.yaml`：推送 `v*` 标签时构建并发布正式版本，例如 `v0.1.0`。
- `nightly.yaml`：`main` 分支更新或手动触发时，滚动更新 `nightly` 预发布版本。
- `container.yaml`：构建多架构容器镜像并推送到 GHCR；Pull Request 只构建验证，不推送。

每次发布包含 Linux `amd64`、`arm64` 的 `boxd` 裸二进制、完整部署压缩包、`boxd-manifest.json` 和 `checksums.txt`。部署包内含 `boxd`、`boxd.service`、`install.sh`、`uninstall.sh` 与项目文档，Web 控制台已通过 Go embed 内置，无需额外静态文件。

在 systemd Linux 主机上解压对应架构的压缩包后执行：

```bash
tar -xzf boxd-linux-amd64.tar.gz
cd boxd-*-linux-amd64
sudo ./install.sh
```

安装脚本会创建 `/var/lib/boxd`、安装 `/usr/local/bin/boxd` 和 `/etc/systemd/system/boxd.service`，启用并启动服务。默认监听 `[::]:7662` 并在启动时拉起内核；使用 `sudo ./install.sh --no-start` 可只安装不启动。卸载时默认保留 `/var/lib/boxd`，仅在显式执行 `sudo ./uninstall.sh --purge` 时删除业务数据。

### 容器镜像

容器镜像推送到 GitHub Container Registry，地址固定为 `ghcr.io/<github-owner>/<github-repo>`。正式版本发布 `0.1.0`、`0.1`、`0`、`v0.1.0` 和稳定版 `latest`；`main` 分支发布 `nightly` 与 `nightly-<commit>`。发布 manifest 的 `container` 字段也会记录镜像地址和标签。

```bash
docker pull ghcr.io/<owner>/<repo>:latest
docker run -d --name boxd --restart unless-stopped \
  -p 7662:7662 \
  -p 1080:1080 \
  -v boxd-data:/var/lib/boxd \
  ghcr.io/<owner>/<repo>:latest
```

容器默认同时启动 Web 控制台与 sing-box 内核。健康检查地址为 `GET /api/v1/health`。需要启用 TUN 时使用 host 网络，并授予网络与设备权限：

```bash
docker run -d --name boxd --restart unless-stopped \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  --device /dev/net/tun \
  -v boxd-data:/var/lib/boxd \
  ghcr.io/<owner>/<repo>:latest
```

镜像默认以 root 运行，这是 TUN 和系统路由管理所需；不使用 TUN 时仍应避免将 `7662` 直接暴露到不可信公网。首次发布后可在 GitHub Package 设置中调整镜像可见性。

发布工作流需要仓库的 **Settings → Actions → General → Workflow permissions** 允许 `Read and write permissions`，以便通过 `GITHUB_TOKEN` 创建 Release。

架构设计见 `docs/superpowers/specs/2026-11-21-boxd-v2-design.md`。

控制面 API、领域约束和配置映射见 `docs/control-plane.md`。
