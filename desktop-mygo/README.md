# boxd MyGo 原生桌面原型

本目录是使用 [MyGo](https://github.com/egoist/mygo) 原生 UI 重写 boxd 桌面端的独立原型模块。它不使用 WebView、HTML、CSS 或 JavaScript，运行时默认复用现有 `~/.boxd/` 数据库和配置格式。

> 当前 Wails 应用仍是默认桌面客户端。本原型不会覆盖 `/usr/local/bin/boxd`、现有 Wails 构建产物或发布流程。

## 当前能力

- Dashboard：内核状态、流量图表、开始/停止/重启，以及模块 Setup 状态、预览和应用闭环。
- Nodes：导入代理链接、编辑/删除手动机节点、查看来源、单节点与批量测速。
- Subscriptions：新增、编辑、删除、单个/全部刷新、URL Test 覆盖和 selector 出口选择。
- Connections：按目标/出口/规则筛选、关闭单个连接、关闭筛选结果和关闭全部。
- Logs：应用与内核日志、级别过滤、固定容量缓冲和末尾跟随。
- Config：JSON 编辑、语义 Diff、恢复到已加载版本、格式化、校验、应用，以及原生文件导入导出。
- Inbounds、Outbounds、Route、DNS 与高级配置段：原生分段 JSON 编辑、校验、应用、默认安装和配置诊断。
- Log、NTP、Experimental、DNS、Route 和 Certificate 常用字段提供原生可视化表单，复杂字段仍可在原始 JSON 中编辑。
- Route 规则、Route rule set、DNS 服务器和 DNS 规则的条目弹窗提供常用字段编辑，高级字段仍通过原始 JSON 编辑。
- Config history：应用历史快照查看、校验后恢复；Rule sets：状态查看、更新和本地自动更新设置。
- Settings：主题、语言、内核自启、桌面登录自启、测试 URL、运行时路径、管理员密码、脱敏支持包和备份导出。
- Terminal：根据真实监听地址生成 Bash、Fish、PowerShell 和 CMD 代理环境变量命令，不执行 shell。
- Logs 与 Config history 支持原生复制/导出；Raw configuration 路由兼容 Wails 的 `/advanced/raw`。
- 后台刷新、错误/过期提示、固定流量与日志缓冲区。
- 单实例、深链 `boxd://import?link=...`、关闭到托盘（托盘不可用时允许关闭退出）。

## 构建与测试

```bash
cd desktop-mygo
GOTOOLCHAIN=auto go build ./...
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./...
golangci-lint run ./...
goimports-reviser -project-name github.com/xuthus5/boxd/desktop-mygo \
  -company-prefixes github.com/xuthus5/boxd \
  -imports-order std,general,company,project ./...
```

正式原型构建：

```bash
cd desktop-mygo
GOTOOLCHAIN=auto go tool mygo build
```

构建产物位于 `desktop-mygo/build/`。`ui.Tester` 截图验收位于 `desktop-mygo/artifacts/acceptance/<run-id>/screenshots/`。

## GitHub 实验构建

`.github/workflows/desktop-mygo-experimental.yml` 在手动触发、`desktop-mygo/` 相关 push 或 PR 时运行。它执行测试、`go vet`、`go tool mygo vet` 和脚本语法检查，并构建可下载的产物：

- Linux amd64/arm64：可执行文件、`.deb`、`.rpm`、`.tar.gz`
- Windows amd64：`.exe`
- macOS arm64：`.app`

该工作流仅用于实验验证，权限为 `contents: read`，不上传 GitHub Release，不接入现有 Wails 发布流程；Windows 安装器和 macOS 签名/公证不在本轮范围内。

## 本地运行

```bash
cd desktop-mygo
BOXD_DATA_DIR="$PWD/.local/data" \
BOXD_CONFIG="$PWD/.local/data/config.json" \
GOTOOLCHAIN=auto go run .
```

默认数据目录为 `~/.boxd/`。可用 `BOXD_DATA_DIR`、`BOXD_CONFIG`、`BOXD_USERNAME`、`BOXD_PASSWORD` 覆盖；`BOXD_DESKTOP_MODE=remote` 仅返回未实现提示。

## 无显示服务器验收

完整验收门禁、证据格式和任务级要求见 `docs/boxd/plans/2026-11-11-mygo-desktop-prototype-ears.md` 的 6.1 至 6.5、7 和 12 节。

安装与当前 MyGo 版本匹配的 automation 客户端：

```bash
GOTOOLCHAIN=auto go install github.com/egoist/mygo/plugins/automation/cmd/mygo-automation@v0.9.0
```

执行 Xvfb 真实窗口验收（覆盖基础页面、Terminal、全部高级配置路由、节点/订阅/连接管理）：

```bash
cd desktop-mygo
./scripts/mygo-headless-e2e.sh
```

脚本使用临时数据目录，构建带 `e2e` 标签的应用，通过 automation socket 验证 `content` 窗口、基础与高级页面导航，以及节点导入/删除、订阅创建/删除和连接管理入口。日志、环境信息和真实窗口截图写入 `artifacts/acceptance/<run-id>/`。脚本只清理本轮自己启动的应用和 Xvfb。

E2E 与 soak 脚本使用各自独立的 `mygo-automation --session`，可并行运行而不会覆盖彼此的默认会话。

首次启动的内核自启可能涉及真实网络和配置；无显示验收默认设置 `BOXD_SKIP_KERNEL_AUTOSTART=1` 隔离桌面壳验证。

在真实桌面执行 TS-026 自动采集与 30 分钟 soak：

```bash
cd desktop-mygo
./scripts/mygo-desktop-acceptance.sh
```

脚本使用临时数据目录，记录显示会话、GPU、缩放和输入法信息，执行七页路由巡检并生成 `manual-checklist.md`。默认在自动采样后保留 5 分钟供人工检查托盘、HiDPI、多显示器、输入法和主观流畅度；可用 `MYGO_DESKTOP_HOLD_SECONDS` 调整。Xvfb 运行只能用于脚本自测，不能作为真实桌面验收结论。

如果需要在服务器上人工查看，可先安装 `x11vnc`，再执行：

```bash
cd desktop-mygo
./scripts/mygo-remote-view.sh --check   # 检查 Xvfb/x11vnc/go
./scripts/mygo-remote-view.sh           # 启动临时会话
```

脚本只监听 `127.0.0.1`，运行时生成临时密码并打印 SSH 隧道命令；`Ctrl-C` 会停止应用、Xvfb 和 x11vnc。不要将端口暴露到公网。

若服务器不可安装 `x11vnc`，且本地工作站已有 X Server，可通过 `ssh -Y` 做临时可视检查；该方式仍受网络和远程 X11 渲染影响，不能替代本机 GPU/Wayland 验收。

## 已知限制

- 当前是快速原型，不包含全部高级字段的完整字段级编辑、完整终端仿真、远程模式、备份导入/恢复、自动更新和正式发布签名。
- Xvfb 使用软件渲染，只验证 GTK3 窗口、布局、导航、截图和基础稳定性；真实 GPU、Wayland、输入法、HiDPI、多显示器、托盘体验和长时间主观流畅度仍需真实桌面确认。
- 无托盘环境（例如当前服务器）会记录降级日志；关闭窗口时应用退出而不是隐藏。
- 本轮不要求 90% 行覆盖率。
