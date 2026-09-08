# 容器首启与模块化配置验收

本轮目标是可重复验证的单节点部署闭环：新卷启动后具备可接入的基础配置，导入节点或订阅后能够使用代理；高级 TUN 模式按实际平台能力启用，失败可恢复且不误报成功。

## 已复现的问题

使用 Podman 5.8.4、公开 `nightly` 镜像（源码提交 `5c0724c`）实际复现：

| 场景 | 原有行为（已复现） | 验收要求 |
| --- | --- | --- |
| 全新卷 | 有 mixed，但监听容器内回环；宿主映射端口拒绝连接 | 默认入站可通过明确公布的端口连接 |
| 卷中已有 `{}` | 跳过初始化，入站为空 | 空文件/空对象可恢复完整默认配置 |
| 无 TUN 设备时安装默认入站 | HTTP 200、`rolled_back`，实际仍无入站 | 默认操作不安装不可用 TUN，界面正确解释失败 |
| 禁用 IPv6 的容器 | 固定双栈 TUN 导致 `add address ... permission denied` | 地址族按能力配置，显式选择与错误原因可见 |
| 保存/导入到停止的内核 | `Restart` 意外启动内核 | 保存保持停止；显式启动与运行中重载分开 |

用户报告的 `set ipv6 address: element not found` 文本位于当前依赖的 Windows 路径；Linux 容器的地址配置失败不是该 Windows 错误的直接复现。不能用 Linux 测试声称验证了 Windows 驱动行为。

## 实施范围

- 统一首启和手动默认入站逻辑；默认显式代理，TUN 需明确选择。
- 检测平台、容器、TUN 权限/设备及 IPv6 支持；未知状态明确说明。
- 认证后的模块状态、依赖补齐、差异预览和一次应用；源配置变化时拒绝旧预览。
- 自定义配置保持；损坏 JSON 或语义无效配置需明确选择重建，预览展示原有可解析内容的删除，并保存原文件。
- 新实例配置完成后启用基础代理；已有用户的内核自启设置保持。
- 默认安装、配置回滚和重载失败在 Web/桌面端具有一致语义。
- 上手流程允许订阅或手动节点，区分面板就绪、内核运行与实际代理可达。
- 容器运行文档、端口、权限、健康检查和版本信息与实际构建一致。

## 社区参考

- [v2rayA 快速上手](https://v2raya.org/docs/prologue/quick-start/)及[容器运行方式](https://v2raya.org/docs/prologue/installation/docker/)：导入、选择、启动与客户端接入形成完整流程，透明代理与桥接端口映射分别配置。
- [GUI.for.SingBox QuickStart](https://github.com/GUI-for-Cores/GUI.for.SingBox/blob/5cb8fbb534e64b5a45038752dac87bb315deecc4/frontend/src/views/HomeView/components/QuickStart.vue)：组合订阅与必要配置，提供生成前预览；不照搬旧内核字段。
- [Clash Verge Rev 入门](https://www.clashverge.dev/guide/quickstart.html)及[配置应用恢复](https://github.com/clash-verge-rev/clash-verge-rev/blob/f32fd0eaa9115b8546ba26995fbfce0b79fa5bde/src-tauri/src/cmd/save_profile.rs)：显式模式选择、验证后应用、失败恢复和对应的日志。

## 交付门禁

1. Podman 新卷、空配置卷、重复启动均通过；入站实际存在且能握手。
2. 无 TUN 权限/设备与 IPv6 禁用的路径有真实容器验证。
3. 导入受控节点后，从容器外完成真实 HTTPS 与 DNS 请求；重启后仍有效。
4. 模块预览不写文件；依赖一次应用；旧预览冲突、失败回滚与停止态保存有测试。
5. 浏览器完成登录、模块管理、导入/接入指引及失败反馈验证。
6. Go 测试/race/覆盖率/lint、前端检查、正式镜像构建和部署验证通过。

可重复执行核心容器闭环（需要 Podman、Python 3、curl 和 `/dev/net/tun`）：

```bash
podman build --format docker --build-arg VERSION=local -t localhost/boxd:local .
python3 scripts/check-podman.py localhost/boxd:local
```

脚本只创建带随机前缀的测试容器、临时数据目录和受控 VLESS 节点，验证后清理；不使用宿主网络或修改宿主路由。除接入验证外，还检查健康状态、真实端口冲突回滚和损坏配置的显式恢复。

实验出口可完成代理 HTTPS 与默认国内 DNS 分流查询；直连 `8.8.8.8:443` 和 `1.1.1.1:443` 超时，受控直连出口节点也有相同限制。因此 DNS smoke 使用默认国内 DoH 分流；远端 Google DoH 的公网可达性仍取决于实际代理节点，不能据此宣称已验证该上游。

2026-09-08 的额外依赖版本检查返回失败：既有 Wails `v3.0.0-beta.16` 落后于上游 `beta.17`。本轮主模块与桌面测试、race、lint、安全扫描通过，该独立 CI 版本门禁仍需后续依赖升级处理。

这份验收针对单节点配置和运行可靠性；不以“商用”一词代替实际测试，也不宣称已提供多租户、HA 或合规认证。
