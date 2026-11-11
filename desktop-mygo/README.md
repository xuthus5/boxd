# boxd MyGo 原生桌面原型

本目录是使用 [MyGo](https://github.com/egoist/mygo) 原生 UI 重写 boxd 桌面端的独立原型模块。它不使用 WebView、HTML、CSS 或 JavaScript，运行时默认复用现有 `~/.boxd/` 数据库和配置格式。

> 当前 Wails 应用仍是默认桌面客户端。本原型不会覆盖 `/usr/local/bin/boxd`、现有 Wails 构建产物或发布流程。

## 当前能力

- Dashboard：内核状态、流量图表、开始/停止/重启。
- Nodes：导入代理链接、编辑/删除手动机节点、查看来源、单节点与批量测速。
- Subscriptions：新增、编辑、删除、单个/全部刷新、URL Test 覆盖和 selector 出口选择。
- Connections：按目标/出口/规则筛选、关闭单个连接、关闭筛选结果和关闭全部。
- Logs：应用与内核日志、级别过滤、固定容量缓冲和末尾跟随。
- Config：JSON 编辑、格式化、校验、应用，以及原生文件导入导出。
- Settings：主题、语言、内核自启、桌面登录自启、测试 URL 和运行时路径。
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

## 本地运行

```bash
cd desktop-mygo
BOXD_DATA_DIR="$PWD/.local/data" \
BOXD_CONFIG="$PWD/.local/data/config.json" \
GOTOOLCHAIN=auto go run .
```

默认数据目录为 `~/.boxd/`。可用 `BOXD_DATA_DIR`、`BOXD_CONFIG`、`BOXD_USERNAME`、`BOXD_PASSWORD` 覆盖；`BOXD_DESKTOP_MODE=remote` 仅返回未实现提示。

## 无显示服务器验收

安装与当前 MyGo 版本匹配的 automation 客户端：

```bash
GOTOOLCHAIN=auto go install github.com/egoist/mygo/plugins/automation/cmd/mygo-automation@v0.9.0
```

执行 Xvfb 真实窗口验收：

```bash
cd desktop-mygo
./scripts/mygo-headless-e2e.sh
```

脚本使用临时数据目录，构建带 `e2e` 标签的应用，通过 automation socket 验证 `content` 窗口、七个页面导航，以及节点导入/删除、订阅创建/删除和连接管理入口。日志、环境信息和真实窗口截图写入 `artifacts/acceptance/<run-id>/`。脚本只清理本轮自己启动的应用和 Xvfb。

E2E 与 soak 脚本使用各自独立的 `mygo-automation --session`，可并行运行而不会覆盖彼此的默认会话。

首次启动的内核自启可能涉及真实网络和配置；无显示验收默认设置 `BOXD_SKIP_KERNEL_AUTOSTART=1` 隔离桌面壳验证。

如果需要在服务器上人工查看，只能临时使用绑定 `127.0.0.1` 的 x11vnc/noVNC，并通过 SSH 隧道访问；不要将端口暴露到公网。

## 已知限制

- 当前是快速原型，不包含完整高级配置页、Diff、终端、远程模式、自动更新和正式发布签名。
- Xvfb 使用软件渲染，只验证 GTK3 窗口、布局、导航、截图和基础稳定性；真实 GPU、Wayland、输入法、HiDPI、多显示器、托盘体验和长时间主观流畅度仍需真实桌面确认。
- 无托盘环境（例如当前服务器）会记录降级日志；关闭窗口时应用退出而不是隐藏。
- 本轮不要求 90% 行覆盖率。
