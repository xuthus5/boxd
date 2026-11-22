# boxd v2 内核与控制面架构设计

- 日期：2026-11-21
- 状态：草案，等待评审
- 范围：全新实现，不迁移旧项目代码，不沿用旧项目目录结构
- 目标：以 sing-box 为最底层引擎，建立稳定内核契约、控制面应用层和多客户端适配体系

## 1. 核心结论

boxd v2 按 greenfield 方式实现。sing-box 是最底层的网络引擎，不是业务内核抽象本身。系统由四部分组成：

1. **Engine**：sing-box，提供连接、路由、DNS、TUN、出站等实际网络能力。
2. **Kernel Contract**：boxd 自己定义的稳定内核契约，隔离 sing-box 细节。
3. **Control Plane**：配置、订阅、节点、策略、规则集、设置、观测等业务编排。
4. **Adapters**：入站客户端协议适配、出站引擎适配、存储与平台能力适配。

依赖方向固定为：

```text
Hosts -> Inbound Adapters -> Control Plane -> Kernel Contract
                                             ^
                                             |
                                  sing-box Driver
                                             |
                                        sing-box Engine
```

控制面只依赖契约，不直接依赖 sing-box。sing-box Driver 是唯一允许导入 sing-box 的区域。

## 2. 目标与非目标

### 2.1 目标

- 内核生命周期、配置操作、运行时控制和观测能力有明确契约。
- 控制面业务可以脱离真实 sing-box 使用 Fake Engine 完整测试。
- Web、Wails、MyGo 等宿主共享同一套应用能力，不复制初始化逻辑。
- 同一个内核既支持进程内嵌入，也支持未来独立守护进程部署。
- sing-box 升级被限制在 Driver 内，不向控制面和客户端泄漏类型。
- 配置写入、内核重启、订阅同步等操作具备事务语义和回滚能力。
- 事件、日志、流量和连接观测有统一模型。
- 所有关键包具备清晰的公开/内部边界。

### 2.2 非目标

- 第一阶段不承诺同时支持多个非 sing-box 引擎，只为未来扩展保留契约空间。
- 第一阶段不实现跨机器分布式控制面。
- 第一阶段不实现多租户。
- 第一阶段不让客户端直接依赖 Driver。
- 不为了“抽象”而复制 sing-box 的所有配置字段和协议类型。

## 3. 术语

| 术语 | 含义 |
| --- | --- |
| Engine | 实际网络引擎实例，目前为 sing-box |
| Driver | 把某个 Engine 适配成 boxd Kernel Contract 的实现 |
| Kernel Manager | 与引擎无关的状态机、并发控制和操作编排 |
| Config Document | 交给 Engine 校验和应用的配置文档，以字节和修订号表示 |
| Control Plane | 控制面应用层，组织业务用例和领域状态 |
| Inbound Adapter | HTTP、Wails、MyGo、CLI 等入站协议适配器 |
| Driven Adapter | sing-box、存储、下载器、平台能力等出站适配器 |
| Host | 进程入口，例如 headless 服务、Wails 桌面端、MyGo 桌面端 |
| Revision | 配置或领域状态的单调修订标识 |
| Desired State | 控制面持久化的用户意图，例如订阅、节点、设置 |

## 4. 架构原则

### 4.1 契约优先

先定义行为、状态机和错误语义，再实现 Driver 和控制面。契约测试先于具体实现。

### 4.2 引擎隔离

只有 `internal/driver/singbox` 可以导入 `github.com/sagernet/sing-box`。其余包不得出现 `box`、`adapter`、`option` 等 sing-box 类型。

### 4.3 单一写入者

同一数据目录和同一 Engine 实例只有一个内核状态机负责 `Start`、`Stop`、`Restart`、`ApplyConfig`。所有入站适配器必须通过控制面进入该状态机。

### 4.4 控制面不依赖传输

控制面不感知 HTTP、Wails、MyGo、SSE 或任何前端模型。传输层只做命令/查询转换和事件编码。

### 4.5 适配器不做业务

入站适配器不合并配置、不计算订阅状态、不决定重载策略；出站适配器不包含业务规则。

### 4.6 组合根唯一

依赖装配只发生在 `internal/bootstrap` 和 `cmd/*` 中。宿主不得自行构造 Engine、Driver、数据库或控制面服务。

### 4.7 显式错误和可操作诊断

错误必须包含稳定错误码、操作、可重试性、底层原因和用户可操作提示。禁止用字符串匹配驱动控制流。

### 4.8 事件不阻塞内核

日志、流量、连接和状态事件通过有界队列分发。订阅者过慢时按明确策略丢弃旧事件或断开订阅，不得阻塞网络引擎。

## 5. 逻辑层级

```text
┌──────────────────────── L5 宿主与表现层 Hosts ────────────────────────┐
│ Web UI │ Wails UI │ MyGo UI │ CLI │ 未来移动端/第三方客户端           │
└──────────────────────────────┬────────────────────────────────────────┘
                               │ 调用协议或应用门面
┌──────────────────────────────▼────────────────────────────────────────┐
│ L4 入站适配器 Inbound Adapters                                        │
│ HTTP/JSON + SSE │ Wails Bindings │ MyGo Bindings │ CLI │ IPC/RPC      │
└──────────────────────────────┬────────────────────────────────────────┘
                               │ Commands / Queries / Events
┌──────────────────────────────▼────────────────────────────────────────┐
│ L3 控制面应用层 Control Plane                                         │
│ Kernel Use Cases │ Config │ Subscription │ Node │ Policy │ RuleSet   │
│ Settings │ Observability │ Setup │ Backup │ Auth(optional)            │
└───────────────┬───────────────────────────────────┬───────────────────┘
                │ Kernel Port                       │ Outbound Ports
┌───────────────▼──────────────────┐   ┌────────────▼──────────────────┐
│ L2 内核契约 Kernel Contract       │   │ L2 外围端口                    │
│ Lifecycle │ Config │ Runtime      │   │ Repository │ FileSystem       │
│ Connection │ Observer │ Features  │   │ Downloader │ Secret │ Clock   │
└───────────────┬──────────────────┘   └────────────┬──────────────────┘
                │ 由 Driver 实现                     │ 由基础设施实现
┌───────────────▼──────────────────┐   ┌────────────▼──────────────────┐
│ L1 出站适配器 Driven Adapters     │   │ L1 基础设施适配器              │
│ sing-box Driver                  │   │ SQLite / OS / HTTP / Platform │
└───────────────┬──────────────────┘   └───────────────────────────────┘
                │ 封装
┌───────────────▼───────────────────────────────────────────────────────┐
│ L0 sing-box Engine                                                    │
│ 连接管理 │ 路由 │ DNS │ TUN │ 出站 │ Clash API │ 网络栈                │
└───────────────────────────────────────────────────────────────────────┘
```

## 6. 包结构

初始采用单一 Go module，避免过早拆模块带来的版本和发布负担。`kernel` 和 `control` 作为稳定公开包，其余实现放在 `internal`。

```text
boxd_v2/
├── kernel/                         # 公开内核契约，不依赖 sing-box
│   ├── engine.go                   # Engine / EngineFactory
│   ├── lifecycle.go                # Start / Stop / Restart / Status
│   ├── config.go                   # ConfigDocument / Validate / Apply
│   ├── runtime.go                  # 出站组、模式、延迟、DNS 刷新
│   ├── connection.go               # 连接查询与关闭
│   ├── observer.go                 # 日志、流量、事件
│   ├── feature.go                  # 引擎特性能力契约
│   ├── types.go                    # 纯 boxd 自有类型
│   └── errors.go                   # 稳定错误模型
│
├── control/                        # 公开控制面应用契约
│   ├── app.go                      # Application Facade
│   ├── commands.go                 # 命令类型
│   ├── queries.go                  # 查询类型
│   ├── events.go                   # 应用事件
│   └── dto/                        # 客户端可见 DTO
│
├── internal/
│   ├── kernel/
│   │   └── manager/                # 与引擎无关的状态机和串行编排
│   │
│   ├── driver/
│   │   └── singbox/                # 唯一允许导入 sing-box 的区域
│   │       ├── engine.go
│   │       ├── config.go
│   │       ├── diagnostics.go
│   │       ├── runtime.go
│   │       ├── traffic.go
│   │       ├── capabilities.go
│   │       └── defaults/
│   │
│   ├── controlplane/               # control 契约实现
│   │   ├── app.go
│   │   ├── kernel/
│   │   ├── config/
│   │   ├── subscription/
│   │   ├── node/
│   │   ├── policy/
│   │   ├── ruleset/
│   │   ├── settings/
│   │   ├── observability/
│   │   ├── setup/
│   │   ├── backup/
│   │   └── security/
│   │
│   ├── adapters/
│   │   ├── inbound/
│   │   │   ├── http/
│   │   │   ├── wails/
│   │   │   ├── mygo/
│   │   │   └── cli/
│   │   └── outbound/
│   │       ├── sqlite/
│   │       │   ├── migrations/
│   │       │   └── repository/
│   │       ├── httpdownload/
│   │       ├── secret/
│   │       └── platform/
│   │           ├── windows/
│   │           ├── linux/
│   │           └── darwin/
│   │
│   └── bootstrap/                  # 唯一组合根
│
├── cmd/
│   ├── boxd/                       # headless / Web 宿主
│   ├── boxd-wails/                 # Wails 宿主
│   └── boxd-mygo/                  # MyGo 宿主
│
└── ui/                             # 共享前端
```

## 7. 内核契约

### 7.1 契约设计原则

- 按能力拆分接口，避免一个巨型接口。
- 契约只使用 boxd 自有类型和标准库类型。
- `ConfigDocument` 对控制面是不透明配置，不在契约中暴露 sing-box 字段。
- 引擎特性差异通过能力接口表达，不通过类型断言泄漏驱动细节。
- 所有操作接受 `context.Context`。
- 状态查询与状态变更分离。
- 事件订阅与查询分离。

### 7.2 核心类型草案

```go
package kernel

import (
    context
    time
)

type EngineKind string

const (
    EngineSingBox EngineKind = sing-box
)

type State string

const (
    StateStopped  State = stopped
    StateStarting State = starting
    StateRunning  State = running
    StateStopping State = stopping
    StateFailed   State = failed
)

type Status struct {
    Kind      EngineKind
    State     State
    Version   string
    StartedAt time.Time
    Revision  string
    LastError *Error
}

type ConfigDocument struct {
    Format   string
    Body     []byte
    Revision string
}

type Diagnostic struct {
    Severity string
    Code     string
    Path     string
    Message  string
    Hint     string
}

type ValidationResult struct {
    Valid       bool
    Diagnostics []Diagnostic
    Normalized  []byte
}

type ApplyOptions struct {
    Source           string
    ExpectedRevision string
    RestartIfRunning bool
}

type ApplyResult struct {
    Revision  string
    Restarted bool
    Warnings  []Diagnostic
}

type Capabilities struct {
    HotReload          bool
    ClashMode          bool
    DNSFlush           bool
    ConnectionControl  bool
    TrafficObservation bool
    ConfigDiagnostics  bool
}
```

### 7.3 能力接口

```go
package kernel

type EngineInfo interface {
    Kind() EngineKind
    Capabilities() Capabilities
}

type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Restart(ctx context.Context) error
    Status() Status
}

type ConfigController interface {
    ReadConfig(ctx context.Context) (ConfigDocument, error)
    ValidateConfig(ctx context.Context, doc ConfigDocument) (ValidationResult, error)
    ApplyConfig(ctx context.Context, doc ConfigDocument, opts ApplyOptions) (ApplyResult, error)
}

type RuntimeController interface {
    OutboundGroups(ctx context.Context) ([]OutboundGroup, error)
    SelectOutbound(ctx context.Context, group, outbound string) error
    OutboundDelay(ctx context.Context, tag, url string) (time.Duration, error)
    ClashMode(ctx context.Context) (ClashMode, error)
    SetClashMode(ctx context.Context, mode string) error
    FlushDNS(ctx context.Context) error
}

type ConnectionController interface {
    ListConnections(ctx context.Context) ([]Connection, error)
    CloseConnection(ctx context.Context, id int64) error
    CloseConnections(ctx context.Context, filter ConnectionFilter) (int, error)
}

type Observer interface {
    RecentLogs(limit int) []LogEntry
    TrafficSnapshot() TrafficSnapshot
    Subscribe(ctx context.Context, types ...EventType) (<-chan Event, error)
}
```

### 7.4 Engine 与 Factory

```go
package kernel

type Engine interface {
    EngineInfo
    Lifecycle
    ConfigController
    RuntimeController
    ConnectionController
    Observer
    Close() error
}

type OpenOptions struct {
    Name       string
    WorkDir    string
    Config     ConfigDocument
    LogWriter  LogSink
    EventSink  EventSink
}

type Factory interface {
    Kind() EngineKind
    Open(ctx context.Context, opts OpenOptions) (Engine, error)
}
```

### 7.5 Kernel Manager

`Kernel Manager` 位于 `internal/kernel/manager`，与具体引擎无关，负责：

- 单实例状态机。
- 所有变更操作的串行化。
- 操作超时和取消。
- 配置应用失败后的回滚。
- 状态事件发布。
- 防止停止状态下执行运行时操作。
- 防止并发的 Start/Stop/Reload。
- 维护最后成功修订号和最后错误。

控制面只依赖 Manager，不直接操作 Driver。Manager 对上层暴露 `kernel.Manager` 契约：

```go
package kernel

type Manager interface {
    EngineInfo
    Lifecycle
    ConfigController
    RuntimeController
    ConnectionController
    Observer
}
```

### 7.6 引擎特性接口

不同 Engine 的配置语义不同。为了保证控制面不导入 sing-box 类型，又保留配置诊断、默认配置、节点链接解析等能力，使用可选能力接口：

```go
package kernel

type ConfigDiagnosticsFeature interface {
    DiagnoseConfig(ctx context.Context, doc ConfigDocument) ([]Diagnostic, error)
}

type DefaultsFeature interface {
    InstallDefaults(ctx context.Context, doc ConfigDocument, section string) (ConfigDocument, any, error)
}

type NodeCodecFeature interface {
    DecodeNodeLink(ctx context.Context, link string) (NodeSpec, error)
    EncodeNodeLink(ctx context.Context, spec NodeSpec) (string, error)
}
```

控制面通过 `Capabilities` 判断能力是否可用，通过窄接口调用，不通过 `switch driver.(type)` 判断具体驱动。

### 7.7 错误模型

```go
package kernel

type ErrorCode string

const (
    ErrCodeNotRunning       ErrorCode = kernel_not_running
    ErrCodeInvalidState     ErrorCode = kernel_invalid_state
    ErrCodeConflict         ErrorCode = kernel_conflict
    ErrCodeInvalidConfig    ErrorCode = kernel_invalid_config
    ErrCodeApplyFailed      ErrorCode = kernel_apply_failed
    ErrCodeFeatureDisabled  ErrorCode = kernel_feature_disabled
    ErrCodeTimeout          ErrorCode = kernel_timeout
    ErrCodeCanceled         ErrorCode = kernel_canceled
    ErrCodeInternal         ErrorCode = kernel_internal
)

type Error struct {
    Code      ErrorCode
    Op        string
    Message   string
    Hint      string
    Retryable bool
    Temporary bool
    Err       error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }
```

错误规则：

- 调用方使用 `errors.As` 判断 `*kernel.Error`，禁止匹配错误字符串。
- 驱动必须把底层错误包装成本模型，同时保留 `Unwrap` 链。
- 传输适配器负责把错误码映射为 HTTP 状态码、Wails 错误或 CLI 退出码。
- `Retryable` 与 `Temporary` 必须分开表达，避免把配置错误误判为可重试。
- 诊断信息不得包含密钥和完整订阅 URL。

### 7.8 Manager 的配置持久化端口

Manager 需要负责配置修订和回滚，但不直接依赖具体文件格式或存储实现。定义一个最小端口：

```go
package kernel

type ConfigStore interface {
    Load(ctx context.Context) (ConfigDocument, error)
    SaveAtomic(ctx context.Context, doc ConfigDocument) error
    History(ctx context.Context, limit int) ([]ConfigRevision, error)
}

type ConfigRevision struct {
    Revision  string
    Source    string
    CreatedAt time.Time
    Summary   string
}
```

文件存储和数据库历史可以分别实现该端口。Driver 只负责 Engine 语义校验和运行控制，不决定历史策略。

### 7.9 基础 DTO 清单

以下类型由 `kernel` 包定义，不允许直接复用 sing-box 类型：

- `OutboundGroup`：`Type`、`Tag`、`Selected`、`Members`。
- `ClashMode`：`Current`、`Available`。
- `Connection`：`ID`、`Inbound`、`Network`、`Source`、`Target`、`Outbound`、`Rule`、`Process`、`Upload`、`Download`、`StartedAt`。
- `ConnectionFilter`：`IDs`、`Outbound`、`Rule`、`Process`。
- `LogEntry`：`Time`、`Level`、`Source`、`Message`、`Fields`。
- `TrafficSnapshot`：`UploadTotal`、`DownloadTotal`、`UploadRate`、`DownloadRate`、`At`。
- `TrafficConnection`：`Connection` 加实时上下行速率。
- `Event`：`Type`、`Sequence`、`Time`、`Payload`。
- `NodeSpec`：控制面节点意图，不包含引擎配置结构。
- `EngineDescriptor`：`Kind`、`Version`、`Capabilities`。

## 8. 内核状态机

### 8.1 状态定义

| 状态 | 含义 |
| --- | --- |
| `stopped` | Engine 未运行，可以启动或写入配置 |
| `starting` | 正在打开或启动 Engine |
| `running` | Engine 正常运行，可以执行运行时操作 |
| `stopping` | 正在停止 Engine |
| `failed` | 启动失败或运行期致命错误，需要显式重试或停止 |

### 8.2 状态转换

| 当前状态 | 操作 | 目标状态 | 说明 |
| --- | --- | --- | --- |
| stopped | Start | starting -> running | 失败则 failed |
| failed | Start | starting -> running | 允许重试 |
| running | Stop | stopping -> stopped | 幂等 |
| stopped | Stop | stopped | 直接成功 |
| running | Restart | stopping -> starting -> running | 串行执行 |
| running | ApplyConfig | running | 重启或热重载由 Driver 能力决定 |
| stopped | ApplyConfig | stopped | 只校验并持久化，不启动 |

同一时刻只允许一个变更操作。查询操作不进入变更队列，但必须读取原子快照。

### 8.3 配置应用流程

```text
读取当前 Revision
    -> 校验 ExpectedRevision
    -> 调用 Driver ValidateConfig
    -> 保存 previous Document
    -> 原子持久化 candidate Document
    -> 如果运行中且需要重启：Stop -> Start
    -> 成功：更新 Revision，发布事件
    -> 失败：恢复 previous Document，按策略重启旧配置，发布失败事件
```

规则：

- 任何配置写入必须先验证。
- 配置持久化必须原子写入。
- 运行中应用失败必须回滚文件和运行状态。
- 停止状态应用只写文件，不自动启动。
- `ExpectedRevision` 不匹配时返回 `ErrConflict`，避免并发覆盖。
- 不支持热重载的引擎统一使用重启策略；契约保留 `HotReload` 能力位。

### 8.4 并发与超时

- 所有状态变更通过单一 `operationMu` 串行化。
- 查询使用不可变快照，不持有长时间锁。
- 默认启动超时 30 秒，停止超时 10 秒，配置应用超时 60 秒，可由配置覆盖。
- 取消请求不得让状态机停留在中间状态；必须得到最终 `stopped` 或 `failed`。
- 禁止在持锁期间调用外部网络或执行长时间 I/O。

## 9. 观测与事件模型

### 9.1 事件类型

| 类型 | 内容 |
| --- | --- |
| `kernel.state` | 状态变化、修订号、错误 |
| `kernel.log` | 内核日志 |
| `kernel.traffic` | 上下行累计流量和速率 |
| `kernel.connection` | 连接建立、更新、关闭 |
| `config.applied` | 配置成功应用或回滚 |
| `control.changed` | 订阅、节点、设置等领域状态变化 |

### 9.2 分发规则

- 每个事件带单调递增 `Sequence`。
- `EventSink` 使用有界队列。
- 慢消费者按事件类型执行策略：
  - 日志和流量：丢弃最旧事件，并发送 `gap` 标记。
  - 连接和状态：优先保留，不静默丢失。
- 订阅者断开必须释放资源，不泄漏 goroutine。
- 事件分发不得在 Engine 回调中同步写网络。

### 9.3 日志

- 内核日志和应用日志使用统一结构：时间、级别、来源、消息、字段。
- 日志写入有界环形缓冲，供最近日志查询。
- 敏感字段必须脱敏，包括密码、私钥、订阅完整 URL、JWT、令牌。

## 10. 控制面应用层

### 10.1 模块职责

| 模块 | 职责 | 主要依赖 |
| --- | --- | --- |
| Kernel | 内核生命周期、配置应用、状态编排 | Kernel Manager |
| Config | 配置读取、校验、应用、历史、回滚 | Kernel Manager、Repository |
| Subscription | 订阅 CRUD、刷新、同步、流量与到期信息 | Repository、Downloader |
| Node | 节点导入、更新、选择、测速 | Repository、Kernel Runtime、NodeCodec |
| Policy | 路由、DNS、出站策略管理 | Config、Kernel Features |
| RuleSet | 规则集安装、更新、状态 | Repository、Downloader、Config |
| Settings | 应用设置、运行偏好 | Repository |
| Observability | 日志、流量、连接、事件查询 | Kernel Observer |
| Setup | 首次安装、默认配置、环境能力探测 | Config、Platform |
| Backup | 备份、恢复、支持包 | Repository、FileSystem |
| Security | 凭据、会话、访问控制 | Repository、SecretProvider |

### 10.2 Application Facade

`control.Application` 是客户端的应用门面。它不实现业务，只把调用转给模块用例：

```go
package control

type Application interface {
    Kernel() KernelAPI
    Config() ConfigAPI
    Subscriptions() SubscriptionAPI
    Nodes() NodeAPI
    Policies() PolicyAPI
    RuleSets() RuleSetAPI
    Settings() SettingsAPI
    Observability() ObservabilityAPI
    Setup() SetupAPI
    Backup() BackupAPI
    Security() SecurityAPI
}
```

每个子 API 由命令和查询组成。入站适配器只依赖这些接口，不依赖具体实现包。

### 10.3 控制面端口

控制面通过端口访问外部世界：

```go
package control

type Repository interface {
    Settings() SettingsRepository
    Subscriptions() SubscriptionRepository
    Nodes() NodeRepository
    ConfigHistory() ConfigHistoryRepository
    TestResults() TestResultRepository
}

type Downloader interface {
    Get(ctx context.Context, req DownloadRequest) (DownloadResult, error)
}

type Platform interface {
    Capabilities(ctx context.Context) (PlatformCapabilities, error)
    DataDir() string
    EnsurePrivilege(ctx context.Context, feature string) error
}
```

具体端口可以按模块拆分，避免超大接口。上述仅表达依赖边界。

### 10.4 配置所有权

配置有两种模式：

| 模式 | 规则 |
| --- | --- |
| Managed | 控制面拥有配置，订阅、节点、策略等根据 Desired State 生成或合并配置 |
| Manual | 用户直接维护原始配置，控制面只校验和应用，不自动同步受管模块 |

规则：

- 默认使用 Managed 模式。
- 显式切换到 Manual 模式需要确认，并在界面展示当前模式。
- Manual 模式下订阅和节点仍可维护，但不会自动改配置。
- 切回 Managed 模式时，先生成候选配置并完整校验，再允许应用。
- 配置修订号用于检测并发修改。
- 所有配置历史记录来源、模式、操作者和结果。

### 10.5 第一版入口与代理编排能力

#### 入站入口

第一版只提供客户端本机接入能力：

| 类型 | 说明 | 第一版范围 |
| --- | --- | --- |
| `mixed` | HTTP 与 SOCKS 共用同一监听端口 | 必须支持 |
| `tun` | 系统级透明代理入口 | 必须支持，可选启用 |

Mixed 默认配置：

- 监听地址：`[::]`
- 监听端口：`1080`
- 双栈监听：同时接受 IPv4 和 IPv6 连接
- 默认启用
- UDP 可用
- 默认不要求用户名密码

双栈要求：

- 在 Windows、Linux、macOS 上实际验证 IPv4 和 IPv6 均可接入。
- 如果操作系统或 Go 运行时默认启用 `IPV6_V6ONLY`，Driver 或平台适配层必须显式关闭，避免只监听 IPv6。
- 配置诊断需要检测 Mixed 端口冲突和双栈绑定失败。
- 日志和状态页应显示实际监听地址与协议族。

安全约束：

- `[::]:1080` 默认对所有网络接口开放，不等同于仅本机代理。
- 首次启动和入口编辑页必须明确提示“当前入口可能被局域网访问”。
- 默认认证关闭属于产品决定，但这意味着局域网内可访问该代理；后续如需要默认认证，应作为独立决策调整。
- 防火墙、容器端口映射和系统网络策略不在内核层自动修改。

入站卡片只展示：

- 名称
- 类型
- 监听地址与端口
- 启用状态
- 运行状态
- 默认出口路径

高级选项默认折叠，包括认证、UDP、允许局域网、TUN 栈、MTU、auto-route 和系统路由相关参数。第一版不提供 shadowsocks、VMess、Trojan 等服务端入站协议。

#### 出口编排

第一版提供三层出口抽象：

| 抽象 | 说明 |
| --- | --- |
| Node | 单个代理节点 |
| Group | Selector 或 URLTest 节点分组 |
| Chain | 有序多跳代理路径 |

代理链方向定义为从本机到目标：

```text
Chain = [Hop1, Hop2, Hop3]
路径  = 本机 -> Hop1 -> Hop2 -> Hop3 -> 目标
```

生成 sing-box 配置时，后一跳通过前一跳的 `detour` 拨号。用户界面不暴露 `detour` 字段。

第一版约束：

- 链最多 4 跳。
- 每跳只能是 Node 或 Group。
- 禁止重复节点。
- 禁止环路。
- `direct` 和 `reject` 不能作为链中的出口跳。
- Group 为空时禁止保存。
- UDP 入口需要校验整条链路是否支持 UDP。
- 单节点路径不需要显式创建 Chain。

路由绑定：

- 每个入站入口可以绑定默认出口。
- 路由规则可以按域名、IP、进程、端口等条件绑定出口。
- 未匹配流量使用全局默认出口。
- 入口、路由规则和出口路径通过稳定 ID 引用，不通过显示名称引用。

#### 第一版界面模型

只保留三个核心区域：

- 入口：管理 Mixed 和 TUN。
- 出口编排：管理节点、分组和代理链。
- 路由：绑定默认出口和规则出口。

代理链编辑器使用有序列表或可视化路径，支持拖拽排序、链路预览和端到端延迟测试，不提供原始出站 JSON 编辑。

DNS 不进入主导航。DNS 属于基础网络设置，放在设置页面中维护。

#### 第一版领域模型

```go
type InboundProfile struct {
    ID      string
    Name    string
    Kind    InboundKind // mixed, tun
    Enabled bool
    Listen  ListenAddress
    RouteID string
}

type EgressPath struct {
    ID   string
    Name string
    Kind EgressKind // node, group, chain
}

type ProxyChain struct {
    ID   string
    Name string
    Hops []ProxyHop
}

type ProxyHop struct {
    Kind  EgressKind // node, group
    RefID string
}
```

### 10.6 DNS 设置

第一版 DNS 不设计独立页面，也不放在左侧主导航。DNS 配置归入设置页面的“网络 / DNS”分组。

信息架构：

```text
设置
├── 通用
│   ├── 主题
│   └── 语言
├── 网络
│   ├── DNS
│   └── 其他网络设置
├── 路由默认值
└── 高级
```

DNS 设置只保留三个固定槽位：

| 槽位 | 用途 |
| --- | --- |
| 国内 DNS | 解析国内域名和 IP，默认直连 |
| 国外 DNS | 解析非国内域名，默认跟随路由出口 |
| 默认 DNS | 无法分类、槽位禁用或解析失败时兜底 |

每个槽位只展示：

- 是否启用
- 服务商预设
- 协议：UDP、DoH、DoT
- 地址或 URL
- 查询出口：直连、跟随路由默认出口、指定代理链
- 超时

全局 DNS 选项：

- IP 策略：自动、IPv4 优先、IPv6 优先
- DNS 缓存开关，默认开启
- 解析测试

界面要求：

- 设置页默认只显示三个槽位的摘要和启用开关。
- 点击槽位后展开编辑，不同时展开全部字段。
- 路由页只显示当前 DNS 摘要和“去设置修改”入口，不提供 DNS 编辑。
- 不提供自定义 DNS 规则、规则排序、FakeIP 参数或原始 JSON 编辑。
- 解析测试显示命中的槽位、服务器、耗时、解析结果和错误。

行为要求：

- DNS 设置变更产生新的配置修订，并通过统一配置应用流程生效。
- 国内、国外和默认 DNS 的变更与路由模式切换保持原子性，不允许只更新部分 DNS 配置。
- 代理节点域名的 bootstrap 解析固定直连，不依赖国外 DNS 或代理链。
- DNS 设置页面必须提示基础 DNS 泄漏风险，但不需要展示底层实现细节。

领域模型：

```go
type DNSProfile struct {
    Domestic DNSServerConfig
    Foreign  DNSServerConfig
    Default  DNSServerConfig
    Strategy DNSStrategy // auto, prefer_ipv4, prefer_ipv6
    Cache    bool
}

type DNSServerConfig struct {
    Enabled  bool
    Provider DNSProvider // alidns, tencent, cloudflare, google, quad9, custom
    Protocol DNSProtocol // udp, doh, dot
    Address  string
    Detour   DNSDetour   // direct, route_default, egress
    EgressID string
    Timeout  time.Duration
}
```

推荐默认值：

- 国内：阿里 DoH，直连。
- 国外：Cloudflare DoH，跟随路由默认出口。
- 默认：跟随国外 DNS。
- 策略：IPv4 优先。
- 缓存：开启。

## 11. 适配器

### 11.1 入站适配器

| 适配器 | 职责 |
| --- | --- |
| HTTP | REST 命令/查询、SSE 事件、鉴权、CORS、TLS |
| Wails | Wails Service 方法到控制面命令/查询的映射，Wails Event 到事件推送 |
| MyGo | MyGo 绑定到控制面命令/查询的映射，原生事件分发 |
| CLI | 参数解析、命令调用、文本输出、退出码 |

入站适配器禁止：

- 直接操作 Kernel Manager。
- 直接读写配置文件。
- 直接访问 Repository。
- 在适配器内实现订阅同步、配置合并或重试策略。

### 11.2 出站适配器

| 适配器 | 实现端口 |
| --- | --- |
| sing-box Driver | Engine、Config、Runtime、Observer、Features |
| SQLite Repository | 设置、订阅、节点、历史、测试结果 |
| HTTP Downloader | 订阅和规则集下载 |
| Platform | 文件系统、权限、网络能力、系统自启 |
| Secret Provider | 凭据、JWT、加密密钥 |

### 11.3 宿主与组合根

所有宿主共享 `internal/bootstrap`：

- 解析运行配置。
- 初始化日志。
- 打开 Repository。
- 构造 EngineFactory 和 Kernel Manager。
- 构造 Control Plane。
- 构造入站适配器。
- 启动后台任务。
- 注册关闭顺序。

关闭顺序必须为：

```text
入站适配器停止接收请求
    -> 后台任务停止
    -> Control Plane 刷新必要状态
    -> Kernel Manager Stop
    -> Engine Close
    -> Repository Close
```

## 12. 存储设计

### 12.1 技术选型

第一阶段使用 SQLite，驱动固定为 `modernc.org/sqlite`。

选择原因：

- 纯 Go 实现，不依赖 CGO，适合 Windows、Linux、macOS 和桌面宿主交叉构建。
- 单文件数据库，符合单机控制面部署形态。
- 支持事务、索引、外键和标准 SQL，便于后续查询和迁移。
- 通过 `database/sql` 使用，不引入 ORM，避免隐式行为和数据模型耦合。
- Repository 端口隔离具体数据库，未来仍可替换为 PostgreSQL 或其他存储。

约束：

- 业务代码只使用 `database/sql` 和 Repository 接口，不直接持有 `*sql.DB`。
- SQLite 驱动只允许在 `internal/adapters/outbound/sqlite` 中导入。
- 不拼接 SQL；所有带参数查询使用占位符。
- 所有查询和写入接受 `context.Context`。
- 不使用 ORM。

### 12.2 数据库文件与连接参数

默认数据库文件：

```text
<data-dir>/boxd.sqlite
```

推荐连接参数：

- `journal_mode=WAL`
- `busy_timeout=5000`
- `foreign_keys=ON`
- `synchronous=NORMAL`
- 限制最大连接数，写操作保持单写者语义
- 关闭连接前执行 checkpoint 或等待 WAL 自动回收

打开数据库后应显式执行 PRAGMA，并通过测试验证这些参数在连接池中生效。

### 12.3 Schema 草案

初始表结构：

| 表 | 用途 | 关键字段 |
| --- | --- | --- |
| `schema_migrations` | 迁移版本 | `version`、`applied_at` |
| `settings` | 应用设置 | `key`、`value`、`updated_at` |
| `subscriptions` | 订阅源 | `id`、`name`、`url`、`interval_seconds`、`enabled`、`created_at`、`updated_at` |
| `nodes` | 节点及选中状态 | `id`、`subscription_id`、`tag`、`protocol`、`config_json`、`selected`、`created_at`、`updated_at` |
| `test_results` | 测速结果 | `node_id`、`kind`、`latency_ms`、`error`、`tested_at` |
| `config_history` | 配置修订历史 | `id`、`revision`、`source`、`mode`、`summary`、`body`、`created_at` |
| `config_metadata` | 当前配置元数据 | `key`、`value` |
| `rule_set_status` | 规则集状态 | `name`、`source`、`etag`、`sha256`、`updated_at` |
| `audit_events` | 审计事件 | `id`、`action`、`actor`、`source`、`detail_json`、`created_at` |

建议索引：

- `nodes(subscription_id)`
- `test_results(node_id, tested_at DESC)`
- `config_history(created_at DESC)`
- `audit_events(created_at DESC)`
- `subscriptions(enabled, updated_at)`

数据表示规则：

- 时间统一存 UTC Unix 毫秒整数，不使用本地时区字符串。
- 布尔值使用 `0/1` 整数。
- 结构化扩展字段使用 JSON TEXT，只用于非核心查询字段。
- 配置正文和历史快照使用 BLOB 或 TEXT，按迁移版本决定。

### 12.4 事务与并发

- 所有多表写入必须在单个事务内完成。
- 配置历史、当前配置元数据和审计事件在同一事务内更新。
- 订阅刷新和节点替换使用事务，避免中途失败留下部分数据。
- 写事务保持短小，不在事务中执行网络下载或内核操作。
- SQLite 写操作串行化；需要“先检查再写入”的场景使用事务或唯一约束。
- `database is locked` 必须映射为可重试错误，并受 `busy_timeout` 和上层重试策略约束。

### 12.5 迁移

- 使用版本化 SQL 迁移，文件名如 `0001_initial.sql`。
- 迁移文件通过 `embed` 编入二进制。
- 启动时在独占数据库锁下执行迁移。
- 每个迁移在事务内执行，记录 `schema_migrations.version`。
- 禁止修改已发布的迁移文件；只新增迁移。
- 迁移失败时停止服务启动，并返回可操作诊断。

### 12.6 安全与备份

- 数据目录权限 `0700`，数据库文件权限 `0600`。
- WAL 和 SHM 文件同样按敏感数据处理。
- 订阅 URL 等敏感信息在展示和日志中脱敏。
- 备份前执行 WAL checkpoint，确保主数据库文件完整。
- 备份文件默认加密或至少限制为当前用户可读。
- 恢复流程必须先校验数据库完整性和 schema 版本。

## 13. 安全设计

- 所有外部输入必须校验，包括配置 JSON、订阅 URL、节点链接、规则集 URL。
- 订阅下载限制大小、超时、重定向次数和协议。
- 阻止访问本机、私网和元数据地址，除非用户显式允许。
- Web 模式启用认证、限速和会话过期。
- 远程模式必须使用本地回环或 TLS。
- 本地 IPC/RPC 使用短期令牌或操作系统用户边界。
- 日志、支持包和备份默认脱敏。
- 敏感文件创建权限 `0600`，目录 `0700`。
- 不在 URL、命令行参数和环境变量中传递长期密钥。

## 14. 部署形态

### 14.1 嵌入式模式

宿主进程内链接 Kernel Manager、Control Plane 和 sing-box Driver。Wails、MyGo 和桌面 headless 模式优先使用。

优点：启动简单、无 IPC、生命周期一致。缺点：权限、升级和崩溃隔离由宿主承担。

### 14.2 守护进程模式

`boxd` 作为独立进程运行 Kernel Manager 和 Control Plane，通过 HTTP/SSE 暴露。Web UI 和远程客户端连接守护进程。

优点：内核和控制面唯一实例、权限集中、客户端轻量。缺点：需要鉴权、端口管理和进程生命周期管理。

### 14.3 混合模式

同一个 Control Plane 契约支持两种连接方式：

- 进程内直接调用 Application 接口。
- 通过 HTTP/SSE 调用同一组命令、查询和事件。

第一阶段先实现嵌入式模式和 HTTP 入站适配器；守护进程宿主复用同一套 HTTP 适配器。

## 15. 测试策略

### 15.1 内核契约测试

建立 Driver 一致性测试套件，任何 Driver 都必须通过：

- 生命周期幂等性。
- 非法状态转换。
- 配置校验失败。
- 配置应用后重启。
- 应用失败回滚。
- 并发 Start/Stop/Apply。
- 取消和超时。
- 事件顺序和订阅释放。

### 15.2 Fake Engine

提供 `kernel.FakeEngine` 用于控制面测试：

- 可脚本化状态转换。
- 可注入启动失败、重启失败、配置失败。
- 可产生确定性日志、流量、连接事件。
- 可断言调用顺序。

### 15.3 控制面测试

- 每个用例使用 Fake Engine 和内存 Repository。
- 覆盖成功、冲突、回滚、超时、取消和部分失败。
- 后台任务使用 Fake Clock 和 Fake Downloader。

### 15.4 适配器测试

- HTTP：请求/响应、错误码、鉴权、SSE 重连。
- Wails/MyGo：命令映射和事件编码。
- sing-box Driver：真实配置校验、启动、重载和流量观测的集成测试。

### 15.5 质量门槛

- 单元测试覆盖率目标不低于 90%。
- 关键包启用 race 检测。
- 禁止 goroutine 泄漏。
- 对配置生成、状态机和错误映射使用表驱动测试。
- 关键集成测试在 Linux、Windows、macOS 至少各有一条 CI 路径。

## 16. 可观测性

- 使用 `log/slog` 统一结构化日志。
- 关键操作记录：操作名、来源、引擎状态、修订号、耗时、结果。
- 事件流可被 CLI 和诊断包消费。
- 提供 `/healthz` 和 `/readyz` 语义：
  - `healthz` 表示进程和控制面存活。
  - `readyz` 表示 Repository、配置和内核状态满足服务条件。
- 支持导出脱敏诊断包，包含版本、平台、配置摘要、最近日志、规则集状态和最近错误。

## 17. 工程约束

- 函数建议不超过 50 行，文件建议不超过 300 行，嵌套不超过 3 层。
- 所有错误必须被处理或显式忽略。
- 禁止在循环中使用 `defer`。
- 禁止魔法数字，使用具名常量。
- 包的依赖方向由架构测试保护。
- 公开包 `kernel` 和 `control` 禁止依赖 `internal`。
- `internal/driver/singbox` 之外禁止导入 sing-box。
- 所有网络、文件和数据库操作必须接受 `context.Context` 或明确说明不需要。
- 代码格式化和 lint 在提交前执行。

## 18. 实施阶段

### 阶段 0：契约与行为规格

- 完成内核契约。
- 完成状态机规格。
- 完成错误模型和事件模型。
- 完成 Fake Engine。
- 建立契约一致性测试骨架。

### 阶段 1：sing-box Driver

- 实现生命周期。
- 实现配置读取、校验、应用和回滚。
- 实现运行时控制。
- 实现日志、流量和连接观测。
- 实现 sing-box 特性能力。

### 阶段 2：控制面核心

- 实现 Application Facade。
- 实现 Kernel、Config、Settings 用例。
- 实现 Repository 端口和内存实现。
- 用 Fake Engine 完成核心测试。

### 阶段 3：业务模块

- 订阅与节点。
- 测速。
- 策略与规则集。
- 设置、备份、安装流程。

### 阶段 4：入站适配器

- HTTP + SSE。
- CLI。
- Wails。
- MyGo。

### 阶段 5：宿主与加固

- headless/Web 宿主。
- Wails 宿主。
- MyGo 宿主。
- 安全、诊断、性能、跨平台验证。

## 19. 验收标准

架构验收：

- 只有 sing-box Driver 导入 sing-box。
- 控制面可以在没有真实 sing-box 的情况下完成全部业务测试。
- 三个宿主不包含业务逻辑和重复 bootstrap。
- 入站适配器不直接访问存储和配置文件。
- 所有内核变更操作经过单一状态机。

行为验收：

- 运行中应用非法配置不会破坏当前运行实例。
- 配置应用失败可以回滚。
- 并发生命周期操作不会产生中间状态泄漏。
- 事件订阅者断开后无 goroutine 泄漏。
- 停止状态保存配置不会自动启动内核。
- 远程模式与嵌入模式使用同一套行为契约。

工程验收：

- 单元测试覆盖率不低于 90%。
- race 检测通过。
- lint 通过。
- 三平台构建通过。
- 诊断包不包含明文密钥。

## 20. 需要评审确认的决策

以下决策已经给出推荐值，等待逐项指正：

| 决策 | 推荐方案 | 影响 |
| --- | --- | --- |
| 初始部署形态 | 嵌入式 + HTTP 适配器，后续补独立守护进程 | 决定第一阶段范围 |
| 公开包 | `kernel` 和 `control` | 决定客户端依赖边界 |
| 模块拆分 | 先单 module，稳定后再拆 | 降低早期版本管理成本 |
| 配置模式 | 默认 Managed，支持显式 Manual | 决定订阅/节点如何同步配置 |
| 配置重载 | 第一阶段统一重启，保留热重载能力位 | 降低 Driver 复杂度 |
| 事件持久化 | 不持久化内核事件，仅保留有界内存 | 避免存储膨胀 |
| 存储 | SQLite，驱动固定为 `modernc.org/sqlite` | 单文件、事务、无 CGO、跨平台构建简单 |
| 认证归属 | 控制面 Security 模块，HTTP 适配器落地 | 明确 Web 和桌面差异 |
| 引擎抽象程度 | 只抽象生命周期、配置、运行时、观测和必要特性 | 避免复制 sing-box 全量模型 |
| 宿主与内核进程关系 | 嵌入式优先，守护进程模式复用控制面 | 兼顾桌面与 Web |
| 第一版入站范围 | 仅 Mixed 和 TUN | 保持入口模型简单，不混入服务端入站 |
| Mixed 默认监听 | `[::]:1080`，双栈 | 同时接受 IPv4 和 IPv6，需提示局域网暴露风险 |
| 代理编排 | Node、Group、Chain 三层模型，Chain 最多 4 跳 | 支持主流链式代理，同时限制配置复杂度 |
| DNS 入口位置 | 设置页的“网络 / DNS”分组，不进入侧边栏 | 降低导航复杂度，DNS 作为基础设置维护 |
| DNS 配置模型 | 国内、国外、默认三个固定槽位 | 覆盖常见分流且不暴露 DNS 规则编辑器 |

## 21. 不继承旧项目的内容

- 不沿用旧目录划分。
- 不沿用旧包名和类型名。
- 不迁移旧代码。
- 不复用旧 Bootstrap 和巨型依赖集合。
- 不让宿主直接构造 Engine。
- 不让客户端直接依赖引擎实现。
- 不为兼容旧行为而保留不清晰边界。

只借鉴工程经验：配置回滚、错误诊断、事件推送、后台任务管理、安全权限、备份恢复和测试覆盖。
