# sing-box 1.14 配置能力升级

目标版本固定为 **1.14.0**，以该标签的 `option/`、注册表、运行代码和迁移说明为准。编辑器与默认配置面向新格式；旧配置中的错误保留在诊断中说明，不通过兼容转换掩盖。

## 验收范围

| 范围 | 实现要求 | 验证 |
| --- | --- | --- |
| 依赖与构建 | 主模块、桌面模块、容器和打包版本一致；启用新协议所需 build tags | 正式构建及跨平台打包测试通过 |
| 内核集成 | 保留原生网络监视器，处理 1.14 启动竞态；支持命名空间辅助进程及安全 pid 文件 | 回归测试通过 |
| 默认策略 | 去掉已无意义的 independent_cache；VPN 客户端可进入代理组，bridge 不作为代理 | 回归测试通过 |
| DNS/路由 | 新 DNS 响应评估、缓存、超时、匹配字段与规则集多标签可编辑、校验和保存 | 回归测试通过 |
| 协议与高级配置 | Snell、cloudflared、bridge、VPN 端点及新增服务有配置入口 | 回归测试通过 |
| 共享资源 | network_namespaces、certificate_providers、http_clients 可编辑与引用 | 回归测试通过 |
| 配置诊断 | 已移除字段报错；仍支持的弃用字段告警；新引用、依赖环和 DNS 模式冲突可定位 | 回归测试通过 |
| 部署 | 构建已提交版本，并验证 Podman 与 systemd 实际进程 | 交付时记录版本、健康状态与二进制摘要 |

复杂协议的嵌套对象继续使用各编辑器的 Advanced JSON；本轮范围是配置能力，不另建一套远端控制台或交互式 VPN 登录界面。

## 已确认的约束

- 1.14 的旧 DNS server / `dns.fakeip` 格式已移除。`independent_cache`、`store_rdrc`、内联 `tls.acme` 等仍被内核接受但已弃用，默认值与编辑器不再生成它们。
- 新 DNS `query_type` / `ip_version` 与旧地址过滤、旧 DNS action `strategy` 混用会导致启动错误，不能仅做普通弃用提醒。
- `block` 文档的移除说明与 1.14.0 实际注册表不一致；原生阻止出站仍被注册且可运行。默认无节点占位保留原生阻止能力，路由阻止动作使用 `reject`。
- 既有 `proxy`、自定义 DNS、节点、监听地址和内核自启选择继续保留；有效的用户配置不因升级被整份覆盖。
- 网络命名空间 helper 只服务内核生命周期；不会把宿主路由或其他服务纳入测试。
- 正式单文件静态构建启用 OpenVPN、OpenConnect、cloudflared、USB/IP、Tailscale 等功能；Naive 出站另需平台原生库，不包含在该构建中。配置诊断按实际编译能力报告缺失功能，避免将可解析误称为可运行。

## 验证依据

- [1.14.0 发布说明](https://github.com/SagerNet/sing-box/releases/tag/v1.14.0)
- [固定版本配置类型](https://github.com/SagerNet/sing-box/tree/v1.14.0/option)
- [固定版本迁移说明](https://github.com/SagerNet/sing-box/blob/v1.14.0/docs/migration.md)
- Go 全仓测试、race、lint 和安全扫描通过，整体覆盖率 92.48%；桌面模块覆盖率 68.59%，满足其 60% 门禁。
- 前端 1,448 项测试通过，行覆盖率 96.27%、函数 92.78%、分支 90.77%；TypeScript、ESLint 与生产构建通过。
- 真实浏览器的配置保存/重载与诊断深链共 12 项通过，375px 视口无横向溢出。
- 原生 Snell 4→5、6→6 均完成带证书校验的 HTTPS；隔离容器的双栈 TUN、TCP/UDP DNS 与 Rule/Global 模式通过，捕获到的非预期出口 TCP/UDP 包为 0。
- gRPC 升至 1.82.1，修复升级后可达的 GO-2026-6061、GO-2026-4762；扫描未发现代码可达漏洞。
- Windows amd64 与 Darwin arm64 完成交叉编译；没有宣称在对应系统实测驱动。
