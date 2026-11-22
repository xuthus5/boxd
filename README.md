# boxd v2

boxd v2 是围绕 sing-box 重新实现的控制面与多宿主客户端基础工程。当前仓库完成了内核契约、单写入者状态机、第一版控制面业务、SQLite 存储、组合根和 UI 原型；sing-box v1.14.4 Driver 已接入真实配置解析、实例生命周期和日志桥接。

## 当前结构

```text
cmd/boxd/                          headless 宿主入口
kernel/                            稳定内核契约
control/                           稳定控制面契约
internal/kernel/manager/           与引擎无关的状态机
internal/kernel/fake/              契约测试用 Fake Engine
internal/kernel/events/            有界非阻塞事件分发
internal/kernel/logbuf/            最近日志环形缓冲
internal/controlplane/             control 契约实现
internal/adapters/outbound/singboxconfig/ 期望状态到 sing-box JSON 的编译器
internal/adapters/outbound/sqlite/ SQLite、迁移与仓储
internal/adapters/outbound/httpdownload/ 条件 HTTP 下载器
internal/adapters/outbound/subscriptionjson/ 订阅 JSON 解析器
internal/driver/singbox/           sing-box 唯一依赖边界与真实 Driver
internal/bootstrap/                唯一组合根
prototype/                         UI 结构原型
```

依赖方向：`cmd -> bootstrap -> controlplane / adapters / manager -> kernel`。公开包 `kernel` 和 `control` 不依赖 `internal`，sing-box 与 SQLite 驱动分别限制在各自适配器内。

## 已具备

- `Start / Stop / Restart / ApplyConfig` 单写入者状态机。
- 配置修订冲突检测、原子持久化、应用失败回滚和状态恢复。
- Fake Engine、契约测试和可替换 Factory。
- 有界事件总线、日志环、运行状态与配置事件。
- SQLite WAL、`busy_timeout=5000`、外键、版本化内嵌迁移。
- SQLite 配置历史、当前配置指针和设置仓储。
- Kernel、Config、Setup、Inbounds、Egress、Subscriptions、Nodes、Policies、Routes、DNS、RuleSets、Settings、Observability、Backup、Security 控制面用例。
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
- 订阅条件刷新、稳定节点 ID、原子节点替换和节点测速历史。
- 内置/自定义规则集定义、刷新状态和预设依赖校验。
- SQLite 一致快照、`quick_check` 校验和备份列表。
- 本地管理员 PBKDF2 口令与短期会话契约。

## 引擎版本策略

当前锁定 `github.com/sagernet/sing-box v1.14.4`，兼容线为 `v1.14.x`。`go.mod` 是精确版本的事实来源；Driver 在打开引擎前会从 Go build info 读取实际模块版本，只允许同一 minor 线，补丁版本升级不需要修改代码。

补丁升级流程、次版本升级门槛和回退方式见 `docs/sing-box-versioning.md`。

## 快速验证

```text
go test ./...
go test -race ./...
make engine-version
go build -o bin/boxd ./cmd/boxd
go run ./cmd/boxd -data-dir ./tmp-data
```

首次启动会幂等写入默认 Mixed/TUN、路由和 DNS 期望状态，并生成保存在 SQLite 中的 sing-box 配置。`-start` 会启动真实 sing-box；如果默认 1080 端口已被其他代理客户端占用，启动会返回端口绑定错误。sing-box 会在数据目录创建内部 `cache.db`，它只用于引擎缓存，不是 boxd 的业务存储。

连接观测、连接控制、Clash 模式、DNS 刷写和真实 Driver 的出站延迟测量仍通过 `kernel_feature_disabled` 显式返回未实现状态；控制面 API 和测速历史已就绪。

UI 原型入口见 `prototype/README.md`。

架构设计见 `docs/superpowers/specs/2026-11-21-boxd-v2-design.md`。

控制面 API、领域约束和配置映射见 `docs/control-plane.md`。
