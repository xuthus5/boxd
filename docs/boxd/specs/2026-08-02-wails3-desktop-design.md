# boxd Wails3 桌面端设计

日期：2026-08-02

## 目标

为 boxd 增加 Wails3 桌面应用形态，采用**原生绑定模式**（Wails Services + Events），同时**完整保留现有 headless（Web 服务）构建与功能**。桌面端与 Web 端共享同一套业务核心与前端界面，能力 1:1 对齐。

## 范围与约束

- 目标平台：Linux 优先（GTK4 + WebKitGTK 6.0，本机已具备 4.22.4 / 2.52.5）。Wails3 官方支持 `-tags gtk3` 遗留路径，后续如需扩 Windows/macOS 再议。
- 桌面运行形态：**双模式**——默认内嵌（进程内自含 boxd 核心，数据目录默认用户目录），同时支持配置连接已有 boxd 服务（复用现有 JWT 登录）。
- 原生能力：系统托盘、开机自启、单实例守护、原生文件对话框、系统通知、URL Scheme 深链、全局快捷键、隐私/极简模式。
- 认证：内嵌模式启动时自动注入一次性本地凭据免登录；远程模式保留现有登录页与 JWT。
- 打包：产出二进制 + `.desktop` 入口，以及 deb/rpm/AppImage。
- 代码质量门禁沿用项目既有要求（Go 覆盖率 ≥90%、函数 ≤50 行、文件 ≤300 行、golangci-lint、goimports-reviser；前端 vitest 覆盖率 ≥90%）。

## 现状分析（headless 基线）

| 层 | 现状 |
| --- | --- |
| 入口 | `cmd/boxd/main.go`：`config.Parse` → DB → settings → `newHandler` → chi router → `serveUntilSignal` |
| 组装 | `cmd/boxd/bootstrap.go`：`newHandlerBuildState` 组装 core 依赖，`newAPIHandlerSet` 组装 15 个 HTTP handler，`newHandlerRouter` 组装 chi router |
| 业务 | `internal/core`：SBInstance、SubscriptionManager、NodeManager、SettingsManager、RuleSetUpdater 等，全部传输无关 |
| 接入 | `internal/api`：15 个 handler，全部耦合 `http.Request` / `http.ResponseWriter`、JWT 中间件、`APIResponse` 信封、SSE |
| 前端 | `ui/`：React 19 + Vite + shadcn，`src/lib/api`（client/endpoints/sse），`src/lib/session`（JWT sessionStorage） |
| 静态资源 | `cmd/boxd/static_embed.go`（`embed_ui` build tag 嵌入 `ui/dist`）/ `static_dev.go`（非 embed_ui 空 FS） |

关键约束：现有 15 个 handler 的全部业务逻辑耦合在 `internal/api`，且已有高覆盖测试。直接为 Wails 重写会导致两套业务逻辑漂移。因此核心策略是**抽取传输无关的共享 service 层**，HTTP handler 与 Wails service 复用同一层。

## 技术架构

### 双构建形态

```
headless（保留）:  go build -tags "embed_ui ..." ./cmd/boxd        → bin/boxd（现状不变）
desktop（新增）:   wails3 build / go build -tags desktop ./cmd/boxd-desktop → boxd-desktop 桌面应用
```

- `cmd/boxd/` 完全保留，产出与服务部署流程（systemd/Docker/二进制）不受影响。
- 新增 `cmd/boxd-desktop/`：Wails3 入口，`//go:build desktop` 约束，避免影响 headless 的 `go build ./...` / `go test ./...`。
- 前端 `ui/dist` 产物同时供 headless 嵌入（现状）与桌面端 Wails 资源服务器使用。

### 依赖方向（保持单向）

```
页面组件 → 业务 Query/Mutation Hooks → 类型化 API Client（transport 抽象）
         → internal/service（传输无关用例层）
              ↕ 共享
         internal/core（领域逻辑，现状不变）
```

- `internal/service`：新增包，抽取 handler 中的用例逻辑，签名 `func(ctx, req) (resp, error)`，不接触 HTTP。
- `internal/api`：handler 变薄，仅做 HTTP 解析/信封/中间件，转发 service 层。
- `internal/desktop`：新增包，Wails Services 封装同一 service 层，Events 转发实时流。

### Go 侧模块划分

| 路径 | 职责 |
| --- | --- |
| `internal/service/` | 传输无关用例层（15 个 handler 的业务逻辑迁入） |
| `internal/api/` | 薄 HTTP handler + 中间件，调用 service |
| `internal/desktop/` | Wails Services + Events + 原生能力封装 |
| `cmd/boxd/` | headless 入口（现状） |
| `cmd/boxd-desktop/` | 桌面入口（Wails `app.Run`） |

## 详细设计

### 1. 共享 service 层（最大工作量）

对现有 15 个 handler（auth、config、service、stats、import、subscription、nodes、test、settings、backup、network、kernel、runtime、ruleSet + health）逐一抽取：

```go
// internal/service/config.go 示例
type ConfigService interface {
    Get(ctx context.Context) (*model.SingBoxConfig, error)
    Update(ctx context.Context, body []byte, source string) (Status, *model.APIError, error)
    Validate(ctx context.Context, body []byte) (*ValidateResult, error)
    ApplyHistory(ctx context.Context) (*ApplyHistory, error)
    // ... InstallDefaultDNS / RuleSets / Outbounds / Inbounds / Route / Experimental ...
}
```

要点：
- 复用 `internal/core` 不变（SBInstance、SubscriptionManager 等已传输无关）。
- 迁移 `config_apply.go` 的 `applyConfigBytesWithSource`（dry-run 校验、apply、回滚）、`nodes_sync.go`、`subscription_refresh.go` 等复杂逻辑为 service 方法。
- `APIError` 信封与 HTTP 状态码映射保留在 api 层，service 返回领域错误。
- 现有 `internal/api/*_test.go` 覆盖拆分为 service 层测试 + 薄 handler 测试，总覆盖率仍 ≥90%。

### 2. Wails Services 绑定

- 每个 handler 域对应一个 Wails Service 方法，如 `DesktopService.ConfigGet()` / `DesktopService.ConfigUpdate(payload)` / `DesktopService.ServiceStart()`。
- Wails 绑定通过 `wails3 generate bindings` 生成 TS 类型，前端 `src/lib/api/` 的 transport 抽象在桌面模式下调用生成的 `Bindings` 而不再走 fetch。
- 远程模式仍走现有 HTTP transport（fetch + JWT），因此前端 API Client 增加 transport 选择。

### 3. 实时流（SSE → Wails Events）

现状 SSE 通道：`traffic`、`logs`、`app-logs`、`connections`（`stats_handler.go`，轮询采样 + `LogWriter.SnapshotAndSubscribe`）。

桌面模式迁移：
- 保留 `internal/core` 的 `LogWriter` / `TrafficTracker` 作为数据源（现状不变）。
- 新增 `internal/desktop/streamer`：订阅 `LogWriter`，将 `LogEntry` 转发为 Wails 事件 `boxd:log` / `boxd:app-log`。
- 流量/连接改为 ticker 采样后 `app.Event.Emit("boxd:traffic", ...)` / `boxd:connections`。
- 前端 transport 在桌面模式将 `openSSE` 替换为 `Events.On(...)`，其余页面逻辑不变。

事件命名采用 Wails 约定（`boxd:traffic`、`boxd:connections`、`boxd:kernel-log`、`boxd:app-log`）。

### 4. 双模式与认证

- **内嵌模式**（默认）：
  - `cmd/boxd-desktop` 启动时以用户目录（`~/.local/share/boxd`、`~/.config/boxd/config.json`）初始化 DB 与 core，进程内自含。
  - 启动时 `SettingsManager` 生成一次性本地凭据（短期 JWT），在 `WindowRuntimeReady` 后通过事件或 URL 注入 WebView，前端自动携带，免手动登录。
  - 内嵌模式下 HTTP 服务**不对外监听**（仅 core 进程内运行），不暴露端口。
- **远程模式**：前端 transport 指向配置的 boxd 地址（含默认 `http://127.0.0.1:9091`），保留现有登录页与 JWT 流程。

### 5. 原生能力

| 能力 | 实现 |
| --- | --- |
| 系统托盘 | `app.SystemTray`：显示内核状态，快捷启动/停止/重启，打开窗口，退出 |
| 开机自启 | Wails `app.Preferences` / Linux XDG autostart `.desktop` 到 `~/.config/autostart` |
| 单实例守护 | 应用启动互斥（锁文件或 Wails 单实例选项），二次启动聚焦已有窗口 |
| 原生文件对话框 | 导入/导出本地 JSON、备份导出用 `app.Dialog`（GTK4 走 `xdg-desktop-portal`） |
| 系统通知 | Wails notifications 服务（freedesktop 通知规范，Linux 直连 DBus，不依赖 notify-send） |
| URL Scheme 深链 | 注册 `boxd://` scheme，`ApplicationLaunchedWithUrl` 处理 |
| 系统代理切换 | ~~Linux `gsettings` 设置 org.gnome.system.proxy~~ 已移除（不直接修改系统代理配置） |
| 全局快捷键 | Wails KeyBinding（显示/隐藏窗口、启停内核） |
| 隐私/极简模式 | 关闭窗口最小化到托盘，设置内可切换 |

### 6. 前端改造

- `src/lib/api/` 新增 transport 抽象：
  - `webTransport`：现状 fetch + SSE + JWT。
  - `desktopTransport`：调用 Wails 生成 Bindings + `Events.On`。
  - 运行时探测（`window.go` / 生成绑定存在性）选择 transport，页面代码不感知。
- `session` 层：内嵌模式使用注入凭据，远程模式沿用 sessionStorage。
- 配置页新增"运行形态"设置（内嵌 / 远程地址），与自启等原生设置并入 Settings。

### 7. 打包与交付

- `cmd/boxd-desktop/build/linux/` 按 Wails3 模板组织：`Taskfile.yml`（APP_NAME=boxd-desktop）、`nfpm.yaml`（deb/rpm 元数据）、`appicon.png`。
- `wails3 package GOOS=linux` 产出 AppImage + DEB + RPM；`wails3 task linux:create:appimage` 等可单独产出。
- 二进制备选：`wails3 build` 产出 `boxd-desktop` 二进制 + 随附 `.desktop` 文件。
- headless 的现有 `./scripts/package-release.sh` 与 systemd/Docker 流程保持不动。

### 8. 配置文件

| Env | Flag | 默认 | 说明 |
| --- | --- | --- | --- |
| `BOXD_DESKTOP_MODE` | `--desktop-mode` | `embedded` | `embedded` / `remote` |
| `BOXD_REMOTE_URL` | `--remote-url` | `http://127.0.0.1:9091` | 远程模式目标地址 |
| `BOXD_AUTOSTART` | `--autostart` | `false` | 开机自启 |
| `BOXD_SINGLE_INSTANCE` | `--single-instance` | `true` | 单实例守护 |

内嵌模式数据目录默认 `~/.local/share/boxd`，配置文件 `~/.config/boxd/config.json`，目录权限 0700、文件 0600（沿用项目安全规则）。

## 里程碑

1. **M1 服务层抽取**（✅ 已完成）：`internal/service` 建包，15 个 handler 域业务逻辑迁入（config、dns_probe、nodes_sync、network、kernel、service_control、test、settings、subscription、auth、runtime、import、health、stats、backup），api 包保持公开签名不变，headless 行为零变化。service 覆盖率 87.8%，整体覆盖率 91.13%（≥90% 通过），`make check-go` 中 test/race/lint 全绿。
2. **M2 桌面壳**（✅ 完成）：`desktop/` 独立 Go module（独立 go.mod + replace 引用主 module，headless 零污染），`main.go` 窗口 + 系统托盘 + 服务注册，`runtime.go` 内嵌/远程双模式初始化，`bindings.go` Wails 服务绑定（config/service/settings/auth/stats），`bridge.go` REST 路径到用例层的通用分发，`BoxdAuthService.AutoLogin` 内嵌自动登录，`scripts/build-desktop.sh` 构建桌面二进制（含全部 sing-box build tags + 前端 embed）。前端 transport：`ui/src/lib/api/desktop.ts` 运行环境探测 + bridge 调用 + autoLogin，`client.ts` GET 请求桌面模式走 bridge，Wails bindings 生成到 `ui/src/lib/api/bindings`，`auth-context.tsx` 内嵌自动登录注入会话。前端覆盖率门禁通过（整体 >90%，desktop.ts 100%）。
3. **M3 实时流与全页面**（✅ 完成）：`desktop/streamer.go` EventStreamer 订阅 `core.LogWriter` 与 `core.TrafficTracker`，通过 Wails Events（`boxd:traffic`/`boxd:connections`/`boxd:kernel-log`/`boxd:app-log`）推送给前端，替代桌面模式 SSE。前端 `useStreamBuffer` 桌面模式自动切换为事件订阅（`subscribeDesktopStream`），`client.ts` 桌面模式所有请求（含 POST/PUT/DELETE 写操作）走 bridge，未知路径回退 HTTP。desktop.ts 覆盖率 100%，前端整体 >90%。
4. **M4 原生能力**（✅ 完成）：`desktop/native.go` NativeCapabilities（开机自启 XDG autostart、系统通知 Wails notifications 服务（freedesktop 规范，不依赖 notify-send；系统代理切换 gsettings 已移除，不直接修改系统代理配置）、数据目录/配置路径查询），`desktop/dialog.go` DialogService（原生打开/保存文件对话框，GTK4 xdg-desktop-portal），`desktop/url_scheme.go` URLHandler（boxd:// 深链：import 导入节点链接、show 聚焦窗口），单实例守护（SingleInstanceOptions 二次启动聚焦窗口）、隐私/极简模式（窗口关闭隐藏到托盘）、全局快捷键（Ctrl+Shift+B 显示/隐藏窗口）。bridge 新增 `/api/desktop/*` 原生能力路径。
5. **M5 打包**（✅ 完成）：`scripts/package-desktop.sh` 完整打包流程（前端构建 → bindings 生成 → 桌面二进制 → .desktop 入口 → deb/rpm/AppImage），`desktop/build/linux/nfpm/nfpm.yaml` 包元数据与 GTK 依赖。产出：裸二进制（55M）、deb（26M）、rpm（27M）、AppImage（128M）、.desktop。AppImage 需将图标复制为与 .desktop Icon 名一致（规避 wails3 appimage 插件 bug）。CI 集成：`ci.yml` 新增 desktop job（GTK4/WebKitGTK 6.0 安装、desktop 测试、桌面二进制构建门禁），`release.yml` 新增 desktop-package job（tag 触发时上传桌面包到 GitHub Release）。
6. **M6 收尾**（✅ 完成）：全量验证通过（主 module check-go 覆盖率 91.02%、lint 0 issues、race 通过；desktop 测试/race/lint 通过、完整构建成功；前端 typecheck/lint/build 通过、246 文件 1296 测试通过、覆盖率全维度 >90%）。README 补充桌面构建/打包说明。构建产物已清理，工作区干净。

## 验证方式

- 每阶段：`make check-go`（test/race/coverage/lint/govulncheck）、`make check-ui`（typecheck/lint/coverage/build）、`make check-embedded-ui`。
- 桌面：`wails3 doctor` 依赖检查，`wails3 build` 构建冒烟，内嵌/远程双模式手工验收，托盘与自启冒烟。
- 打包：`wails3 package GOOS=linux` 产物可安装、可启动。

## 风险与权衡

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| Wails3 为 ALPHA（v3.0.0-alpha） | API 可能变动 | 锁定版本，最小化 Wails 依赖面，核心逻辑留在 service/core |
| 服务层抽取工作量大 | M1 周期长 | 按 handler 域分批迁移，每批跑全量回归 |
| GTK4/WebKitGTK 6.0 系统依赖 | 老旧发行版不可用 | 提供 `-tags gtk3` 遗留路径说明；打包产物做依赖声明 |
| 内嵌模式无端口 vs 远程模式差异 | 前端双 transport 复杂度 | transport 抽象收敛差异，页面层不感知 |
| sing-box TUN 需 root/NET_ADMIN | 内嵌桌面默认无 TUN 权限 | 内嵌模式提供 TUN 依赖提示；完整 TUN 能力建议连接 systemd 服务（远程模式） |

## 非目标

- 不修改现有 headless 构建与部署流程。
- 不在本期扩展 Windows/macOS 打包。
- 不重写 `internal/core` 领域逻辑。
- 内嵌模式不引入 root 提权能力（TUN 保持远程模式承载）。
