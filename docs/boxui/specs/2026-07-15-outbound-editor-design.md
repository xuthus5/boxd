# 出站配置编辑器设计

## 目标

将当前仅支持 Tag、类型、服务器和端口的出站编辑器，升级为与入站编辑器一致的模块化表单。表单对照项目使用的 sing-box 1.13.14 配置类型，结构化覆盖常用字段，并通过高级 JSON 保留复杂对象和未知扩展字段。

本次仅修改 React + TypeScript 前端，不修改后端 API、配置保存逻辑或 sing-box 启动逻辑。

## 支持范围

结构化支持当前构建注册的常规出站类型：

- 基础：`direct`、`block`
- 分组：`selector`、`urltest`
- 代理：`socks`、`http`、`shadowsocks`、`vmess`、`vless`、`trojan`、`naive`、`shadowtls`、`anytls`
- QUIC：`hysteria`、`hysteria2`、`tuic`
- 其他：`ssh`、`tor`

不提供以下已移除类型：

- `dns`：sing-box 1.13 已移除，改用路由规则动作。
- `wireguard`：sing-box 1.13 已改为 endpoint。
- `shadowsocksr`：已移除。

编辑现有未知类型时，类型选择器仍保留该值，避免破坏历史配置；未知字段始终可在高级 JSON 中编辑。

## 页面结构

出站 Dialog 使用与入站一致的响应式页签结构：

1. 基础
   - Tag
   - 出站类型 Select
   - 服务器和端口；仅服务器型协议显示
2. 拨号与网络
   - 公共 `DialerOptions`
   - detour、绑定接口、绑定地址、路由标记、连接超时、TCP/UDP 行为、域名解析器与网络策略
3. 协议或分组
   - 根据出站类型展示协议专属字段
   - `selector` / `urltest` 展示成员、默认节点、测试地址、间隔、容差和连接中断策略
4. TLS / uTLS / Reality
   - 仅支持 TLS 的协议显示
   - 覆盖服务器名称、证书校验、ALPN、TLS 版本、证书、uTLS、ECH、Reality 和客户端证书
5. 传输与复用
   - VMess、VLESS、Trojan 根据 `transport.type` 动态展示 HTTP、WebSocket、QUIC、gRPC 或 HTTPUpgrade 字段
   - 仅支持 multiplex 的协议展示复用参数
6. 高级 JSON
   - 编辑完整出站对象
   - 保留结构化表单未覆盖的复杂嵌套对象和未来字段

Dialog 使用 shadcn 的 `Dialog`、`Tabs`、`FieldGroup`、`Field`、`Select`、`Switch`、`Input` 和 `Textarea`，不增加全局样式。

## 组件与代码边界

采用“共享字段渲染器 + 独立出站字段模型”：

- 将入站现有的字段渲染、嵌套路径更新、JSON 对象校验和 Select 稳定化能力提取为代理配置共享模块。
- 入站和出站分别维护类型列表、字段元数据、类型切换清理规则及页签组合。
- `ProxyEditorDialog` 只负责按 `inbounds` / `outbounds` 委托给对应编辑器。
- 单个文件不超过 300 行，函数不超过 50 行。

共享模块只处理通用表单行为，不包含任何具体协议规则，避免入站与出站逻辑耦合。

## 数据同步与清理

结构化字段和高级 JSON 操作同一个完整 JSON 对象：

- 表单更新通过嵌套路径修改对象，再序列化到高级 JSON。
- 高级 JSON 修改成功解析后，切回结构化页签立即反映新值。
- JSON 根节点不是对象、缺少类型或结构化 JSON 字段无效时禁用保存。
- 数字及数字列表不允许产生 `NaN` 或 `null`。

切换出站类型时：

- 保留 Tag。
- 服务器型协议之间保留 `server` / `server_port`。
- 支持 `DialerOptions` 的类型之间保留公共拨号字段。
- 同时支持 TLS、transport 或 multiplex 的类型之间保留对应共享对象及其中未知字段。
- 仅删除旧类型不再支持的已知字段路径。
- 用户凭证等协议语义不同的字段不跨协议继承。

切换 transport 类型时只清理旧 transport 子类型独有字段，保留共同字段和未知字段。

## Selector 与 URLTest

`selector` 与 `urltest` 作为普通 sing-box 出站类型提供结构化编辑。

页面同时显示提示：订阅功能自动生成或维护的分组应在订阅页面配置；直接编辑这些分组可能在下次订阅同步时被覆盖。前端不尝试推断分组来源，也不改变后端自动同步逻辑。

## 错误处理

- 无效 JSON 在字段下方显示明确错误，并禁用保存。
- 高级 JSON 语法错误继续使用现有 `JsonEditor` 错误提示。
- 复杂对象采用 JSON 对象或数组输入，避免用 CSV 造成未知字段或包含逗号的值丢失。
- 后端保存失败、回滚及 toast 行为继续复用现有配置保存流程。

## 测试与验收

测试至少覆盖：

- 类型选择器包含全部有效出站类型，并排除已移除类型。
- VLESS、VMess、Trojan 的认证、TLS、uTLS、Reality、transport 和 multiplex。
- Shadowsocks、Hysteria2、TUIC、SSH 的主要字段和 JSON 类型。
- Selector 和 URLTest 的分组字段及订阅管理提示。
- HTTP 与 HTTPUpgrade 的 host 类型差异，以及各 transport 只显示其支持字段。
- 类型切换保留公共拨号、TLS、transport、multiplex 和未知嵌套字段。
- 无效结构化 JSON 在切换页签后仍阻止保存。
- 未知类型和未知字段可无损保存。
- Dialog 在移动端可滚动，桌面端使用正确最大宽度。
- 现有入站编辑器测试全部通过，确保共享模块提取没有回归。

最终执行前端完整覆盖率门禁、Go 门禁、Playwright、生产构建及嵌入资源验证；成功后提交并部署到本机 `boxui.service`。
