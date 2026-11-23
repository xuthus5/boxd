# boxd v2 控制面应用层

## 1. 当前边界

控制面通过公开包 `control` 向 HTTP、Wails、MyGo 和 CLI 暴露统一门面。适配器只依赖接口，不访问 SQLite、配置文件或 sing-box API。

```text
入站适配器
  HTTP / Wails / MyGo / CLI
            |
            v
control.Application
  Kernel / Config / Setup / Inbounds / Egress / Subscriptions / Nodes
  Scheduler / Policies / Routes / DNS / RuleSets / Settings / Observability
  Security / Audit
            |
            v
internal/controlplane
  ProfileService + 各模块用例 + 校验 + 期望状态原子更新
            |
            v
control.ConfigCompiler  <----  internal/adapters/outbound/singboxconfig
control.ProfileRepository <-- internal/adapters/outbound/sqlite
kernel.Manager
```

公开包仍然只有 `kernel` 和 `control`。sing-box 依赖被限制在 `internal/driver/singbox`；sing-box JSON 编译器不导入 sing-box 类型。

## 2. Application 门面

| 模块 | 职责 |
| --- | --- |
| `Kernel()` | 启动、停止、重启、状态查询 |
| `Config()` | 原始配置读取、校验、诊断、应用、历史、按期望状态重编译 |
| `Setup()` | 幂等初始化默认状态、读取引擎能力 |
| `Inbounds()` | Mixed/TUN 入口 CRUD |
| `Egress()` | Node、Group、Chain CRUD 和 Selector 运行时切换 |
| `Subscriptions()` | 订阅 CRUD、条件刷新，以及订阅节点的原子替换 |
| `Scheduler()` | 订阅按周期自动刷新、立即执行、运行状态与启停 |
| `Nodes()` | 节点列表、筛选、批量导入、分享链接导入导出、保存、删除、延迟测试和测试历史 |
| `Policies()` | 路由与 DNS 策略的聚合读取，并复用各自用例入口 |
| `Routes()` | 一键模式、默认出口、预设、规则 CRUD 和排序替换 |
| `DNS()` | 国内、国外、默认三槽位 DNS 配置和刷写 |
| `RuleSets()` | 内置/自定义规则集定义、刷新状态和预设依赖 |
| `Settings()` | 通用偏好键值 |
| `Observability()` | 最近日志、流量、连接、关闭连接、事件订阅 |
| `Security()` | 本地管理员口令、PBKDF2 摘要和内存会话 |
| `Audit()` | 自动记录控制面变更，并提供按动作、操作者、来源和时间查询 |

## 3. 期望状态与原子应用

控制面以 `control.DesiredState` 作为 Managed 模式的唯一事实来源。每次业务变更按以下顺序执行：

1. 串行化读取当前快照。
2. 在副本上执行领域变更。
3. 规范化并执行领域校验。
4. 通过 `ConfigCompiler` 编译候选内核配置。
5. 保存新的期望状态快照。
6. 通过 `Kernel Manager` 统一校验、持久化和应用配置。
7. 配置应用失败时恢复上一份期望状态；管理器同时回滚配置数据库和运行实例。

因此 UI 不需要分别调用 “保存路由” 和 “应用配置”。一次领域操作就是一个可回滚的配置修订。

## 4. 领域约束

### 入站

- 第一版仅支持 `mixed` 和 `tun`。
- Mixed 首次初始化使用 `[::]:1080`、UDP 开启、认证关闭。
- Mixed 监听地址必须是合法 IP；默认双栈入口会接受 IPv4 和 IPv6。
- 同一启用地址不能重复绑定。
- 最多启用一个 TUN 入口。

### 出口编排

- 出口分为 Node、Group、Chain 三类。
- Group 成员是 Node ID；第一版不允许 Group 嵌套 Group，避免环路和选择语义混乱。
- Chain 最多 4 跳，方向为 `本机 -> Hop1 -> Hop2 -> ... -> 目标`。
- Chain 跳只能是 Node 或 Group，不能是 `direct`、`reject` 或另一个 Chain。
- 同一 Chain 禁止重复跳。
- Group 不能为空；Selector 的 `selected` 必须是成员。
- Selector 切换会写回期望状态并应用配置，重启后保持上次选择。
- Mixed 开启 UDP 时，所有 Chain 中的 Node 都必须支持 UDP。
- Node 的协议扩展字段放在 `options` 中；`type/tag/server/server_port/detour` 由系统管理，禁止覆盖。

### 路由

- 模式为 `rules`、`global`、`direct`，切换是单次原子操作。
- `global` 模式必须存在实际代理出口。
- 规则按切片顺序执行。
- 规则条件支持 inbound、域名、域名后缀、关键字、IP CIDR、源 IP CIDR、进程、规则集、网络、端口、私有地址。
- 内置预设包括局域网直连、中国大陆直连、广告拦截。
- `direct` 和 `reject` 只能作为规则动作，不能作为 Chain 跳。

### DNS

- 固定为国内、国外、默认三个槽位，不提供任意 DNS 规则编辑器。
- 协议支持 UDP、DoH、DoT。
- 查询出口支持直连、跟随路由默认出口、指定出口。
- 默认 DNS 可以跟随国外槽位。
- 至少一个槽位必须可用；默认槽位跟随国外时要求国外槽位启用。
- 域名型 DNS 服务器和代理节点域名解析使用隐藏的本地 `dns-bootstrap`，避免依赖国外 DNS 或代理链。

## 5. sing-box 配置映射

编译器把中性领域模型映射为 sing-box v1.14.x JSON：

- Mixed 使用 `mixed` inbound，TUN 使用 `tun` inbound。
- 入站 sniff 使用路由 `sniff` action；v1.14 已移除旧入站 sniff 字段。
- Node 使用协议名作为 outbound `type`，协议字段来自 `options`。
- Group 映射为 `selector` 或 `urltest`。
- Chain 每一跳生成独立 outbound；后一跳通过 `detour` 指向前一跳。
- Group 作为 Chain 跳时，会为每个成员生成带 `detour` 的副本，再生成该跳的 selector/urltest。
- DNS 使用 v1.14 的新服务器格式，`dns-bootstrap` 是内部 local 服务器。
- 路由规则产生 `route.rules`，预设规则集按稳定顺序生成远程 `.srs` 定义。
- 配置修订号是生成 JSON 的 SHA-256，相同期望状态得到相同配置修订。

## 6. 存储

- 期望状态整体保存在 `control_profile` 单行快照中，读写保持原子。
- 配置历史和当前配置指针继续使用 `config_history`、`config_metadata`。
- 设置使用 `settings`。
- 时间使用 UTC Unix 毫秒；JSON 只用于聚合快照和非核心查询字段。
- SQLite 驱动仍只允许出现在 `internal/adapters/outbound/sqlite`。

存储分层：

- `control_profile` 保存完整期望状态，包括入口、订阅、节点、分组、链、路由、DNS 和规则集定义。
- `subscriptions`/`nodes` 旧规范化表不再作为业务事实来源；订阅节点以 `control_profile` 为准。
- `rule_set_status` 保存 ETag、Last-Modified、SHA-256、刷新时间和错误，不触发配置重新应用。
- `test_results` 按稳定 node ID 保存测速历史，不依赖旧的规范化 `nodes` 表。
- `security_credential` 只保存盐、PBKDF2 摘要、迭代次数和更新时间，不保存明文口令。
- `audit_events` 保存动作、操作者、来源、JSON 详情和 UTC 时间，支持按动作、操作者、来源和时间范围查询。

## 7. 订阅、节点与规则集

### 订阅刷新

1. 读取订阅当前 ETag 和 Last-Modified。
2. 下载器执行带超时、响应体上限和条件请求的 HTTP GET。
3. 解析器把 sing-box JSON 或 Base64 JSON 转换为引擎无关节点。
4. 以订阅 ID 为范围生成稳定 node ID。
5. 在同一 profile 事务中更新订阅元数据并替换该订阅的全部节点。
6. 候选状态通过完整领域校验后才会编译、落库和应用。

刷新失败不会留下半更新状态。若旧节点仍被分组、链路或路由引用，而新订阅删除了该节点，刷新会因引用校验失败而回滚。

### 节点测速

- 测速调用内核 `OutboundDelay`，使用稳定 outbound tag 而不是显示名称。
- 结果写入 `test_results`，URL、延迟、错误和测试时间均可查询。
- 测速通过 sing-box URLTest 实际发起请求；内核未运行或 outbound 不存在时返回稳定错误，不会伪造成功结果。

### 内核运行时

- `Observability().Connections/CloseConnection/CloseConnections` 直接读取和关闭 sing-box 连接管理器中的连接。
- `DNS().Flush` 同时清理 sing-box 运行态 DNS 缓存和持久化 DNS 缓存。
- `OutboundDelay` 使用 sing-box URLTest 进行真实测速，并写入 urltest 历史。
- Clash 兼容模式通过 sing-box clashmode 管理器读取和切换，不暴露 Clash HTTP API。

### 节点分享链接

- `NodeCodec` 端口把分享链接与引擎无关的 `control.Node` 互转，适配器位于 `internal/adapters/outbound/nodelink`。
- 支持 Shadowsocks、VMess、VLESS、Trojan、Hysteria2、SOCKS5 和 HTTP 链接。
- 导入链接与对象导入复用同一套 ID 稳定化、领域校验和原子落库流程。

### 订阅调度

- `SchedulerService` 按订阅的 `interval_seconds` 扫描到期订阅，并复用 `SubscriptionAPI.Refresh` 的条件请求与原子替换。
- 调度器由组合根随应用启动，在关闭应用时先停止；`RunOnce` 可触发一次手动扫描。
- 扫描间隔、运行状态、最近一次执行时间和错误通过 `SchedulerAPI` 查询。

### 规则集

- `RuleSetDefinition` 属于期望状态，因为路由规则按名称引用它。
- `RuleSetStatus` 属于独立状态数据，刷新不会触发配置重载。
- 内置规则集不可修改或删除；自定义规则集使用同一编译和校验路径。
- 中国直连和广告拦截预设会校验其依赖规则集存在且启用。
- 当前由 sing-box remote rule-set 在引擎内维护下载缓存；控制面保存 ETag、Last-Modified、SHA-256、刷新时间和错误。

## 8. 安全

- 未初始化时模式为 `local`，适用于仅本机可信传输的嵌入式宿主。
- 初始化后模式为 `password`，口令使用随机盐和 PBKDF2-HMAC-SHA256 派生。
- 会话只保存在进程内，默认 24 小时过期；进程重启后需要重新登录。
- 登出立即撤销会话；进程重启会天然清空全部会话令牌。
- 未授权统一返回 `kernel_unauthorized`；错误信息不区分口令长度、摘要或存储细节。
- HTTP/Wails/MyGo 入站适配器必须调用 `Security().ValidateSession`，不得自行读取凭据表。
- 审计事件默认操作者为 `local`、来源为 `control-plane`；入站适配器可在认证后通过 `control.WithAuditIdentity` 注入真实操作者和来源。

## 9. 当前未覆盖

控制面及以下能力已经闭环，以下项目属于明确的范围外或引擎内实现，不应由入站适配器自行补业务：

- 上层真实应用接入：HTTP/SSE、Wails、MyGo、Web UI 和桌面宿主，留待下层契约与模型稳定后实施。
- 多用户、角色权限和多租户；当前设计是单本地管理员，符合第一阶段非目标。
- 规则集本地预取；当前由 sing-box remote rule-set 在引擎内自动下载并缓存，控制面只维护刷新元数据。
- 持久化令牌撤销列表；当前会话为进程内短期令牌，登出和进程重启都会使令牌失效。

新增模块必须沿用同样的边界：公开接口放在 `control`，用例放在 `internal/controlplane`，外部能力通过端口注入，存储和引擎实现放在 outbound adapter。
