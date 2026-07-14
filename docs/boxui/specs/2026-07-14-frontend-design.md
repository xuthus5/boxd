# BoxUI 前端设计

日期：2026-07-14

## 目标

为现有 BoxUI Go 后端新增 React 19、TypeScript 与 Vite 前端，完整对接现有 API，不修改后端代码。界面严格遵循 shadcn/ui 的组件组合、语义色和布局规范，不增加额外的全局视觉样式。

## 范围与约束

- 提供仪表盘、代理配置、流量策略、节点与订阅、运行观测、高级配置和应用设置。
- 登录用户名由后端启动配置决定。由于现有 API 不支持查询或修改用户名，前端仅说明该限制，不提供用户名轮换。
- 支持管理员密码和 JWT 密钥轮换，成功后清除当前会话并要求重新登录。
- 常用 sing-box 字段采用结构化表单，复杂字段提供单项 JSON 编辑，完整配置提供独立 JSON 编辑器。
- 前端业务代码的 lines、functions、statements、branches 覆盖率均不低于 90%。
- 函数不超过 50 行，文件不超过 300 行，嵌套不超过 3 层。

## 技术架构

- React 19 + TypeScript + Vite：应用基础。
- React Router：受保护路由和页面级懒加载。
- TanStack Query：HTTP API 缓存、刷新、失效与错误状态。
- React Hook Form + Zod：表单状态和前端校验。
- shadcn/ui：Sidebar、Card、Tabs、Table、Dialog、Sheet、Field、Alert、Empty、Skeleton、Sonner 等界面组件。
- Recharts：通过 shadcn Chart 展示实时和历史流量。
- CodeMirror：单项高级 JSON 与完整配置编辑。
- i18next：中英文切换。
- Vitest、Testing Library、MSW：单元、组件和 API 集成测试。
- Playwright：关键流程浏览器端冒烟测试。

依赖方向保持单向：

```text
页面组件
  -> 业务 Query / Mutation Hooks
  -> 类型化 API Client
  -> 现有 Go API
```

页面组件不直接拼接 URL 或处理响应格式。API Client 同时兼容直接 JSON 响应和 `APIResponse` 包装响应，并将失败统一归一化为类型化 `ApiError`。

## 导航与页面

侧边栏结构：

```text
仪表盘
代理配置
├── 入站配置
└── 出站配置
流量策略
├── 路由
└── DNS
节点与订阅
├── 节点
└── 订阅
运行观测
├── 活跃连接
└── 日志
高级配置
├── Endpoints
├── Experimental
└── 完整配置
应用设置
```

“内核日志”和“应用日志”不是侧边栏导航项，只在日志页面内部通过 shadcn `Tabs` 切换。

### 登录

提交用户名与密码到 `/api/auth/login`。JWT 与过期时间保存在 `sessionStorage`，刷新页面时恢复有效会话。登出调用 `/api/auth/logout` 并清除本地会话。

### 仪表盘

- 展示服务运行状态、版本、运行时间和内存。
- 提供启动、停止和重启操作。
- 使用历史接口初始化流量图，再通过 SSE 追加实时上下行流量。
- 提供 GC、DNS 缓存清理和 FakeIP 清理。
- 展示最近内核日志摘要。

### 代理配置

入站和出站分别读取完整配置中的 `inbounds` 与 `outbounds`。支持列表、新增、编辑、删除和排序。常用字段使用表单，未知或协议特有字段通过高级 JSON 保留，避免结构化编辑丢失数据。出站页面提供默认出站安装操作。

### 流量策略

- 路由页面管理 `route.rules`、`route.rule_set` 和最终出站，支持顺序调整以及默认路由和规则集安装。
- DNS 页面管理服务器、规则和基础 DNS 选项，支持安装默认 DNS。
- 保存时更新完整配置，并依赖后端校验、自动重启和失败回滚。

### 节点与订阅

- 节点页面支持链接导入、保存、编辑、删除、单项与批量测速、结果展示、同步配置、运行时选择器切换和 URLTest 延迟查询。
- 订阅页面支持新增、编辑、删除、单项刷新和全部刷新，展示更新时间、节点数量及错误状态。

### 运行观测

- 活跃连接页面通过 SSE 展示连接列表、流量和持续时间，支持关闭单项或全部连接。
- 日志页面使用 Tabs 切换内核日志和应用日志。两类日志拥有独立缓冲、过滤条件和滚动位置，支持等级过滤、关键词搜索、暂停、自动跟随和清空当前显示。

### 高级配置

- Endpoints 和 Experimental 页面提供结构化概览与 JSON 编辑。
- 完整配置页面提供 JSON 编辑、格式化、撤销未保存修改和保存确认。
- 本地 JSON 语法校验通过前禁止提交。

### 应用设置

- dark、light 和跟随系统主题。
- 中文与英文界面切换。
- 登录用户名配置方式说明。
- 管理员密码轮换。
- JWT 密钥状态展示和轮换。
- 系统测速地址配置。
- sing-box 内核自启配置。

## 视觉与组件规则

- 只使用 shadcn/ui 已有组件、内置 variants 和语义化颜色。
- `className` 仅用于布局，不覆盖组件颜色和字体系统。
- 桌面端使用 Sidebar，窄屏使用其移动端折叠能力。
- 表单使用 `FieldGroup`、`Field`、`FieldSet` 等组件组合。
- 数据密集页面使用 Table、Tabs、Sheet 和 Dialog。
- 危险操作使用确认 Dialog；操作结果使用 Sonner；页面错误使用 Alert。
- 加载状态使用 Skeleton，空状态使用 Empty，不自行创建替代组件。
- 不添加装饰性渐变、自定义全局颜色或额外全局组件样式。

## 数据与会话处理

- 普通 API 请求统一附加 Bearer Token。
- `401` 清除会话并跳转登录页；`429` 展示限流提示。
- 主题和语言以带版本号的数据结构保存到 `localStorage`。
- 日志、流量和连接使用 Fetch Stream 解析 SSE，因为原生 `EventSource` 无法携带 Bearer Header。
- SSE 在页面卸载时主动取消，断开后使用有限指数退避重连。
- 配置保存成功后使相关 Query 失效并重新获取。
- 后端返回 `rolled_back` 时明确提示保存未生效且配置已回滚。
- 编辑配置时保留未知字段，避免前端覆盖后丢失后端支持的配置能力。

## 错误处理

- 所有 Promise、Stream 和存储操作均显式处理失败。
- 配置校验失败时保留编辑内容，并在对应表单或编辑器展示错误。
- 删除、关闭全部连接、轮换 JWT 和覆盖完整配置必须二次确认。
- 密码或 JWT 轮换成功后立即清除当前会话。
- 页面级失败不以空数据伪装，必须展示可重试状态。

## 测试与验收

采用 TDD 实现高风险和共享逻辑：

- API Client：直接响应、包装响应、错误响应、认证失效和回滚响应。
- SSE：事件分片、跨分片数据、错误事件、取消和重连。
- 认证：登录、登出、过期会话和受保护路由。
- 配置：字段保留、数组操作、JSON 校验和保存结果处理。
- 页面：加载、空数据、成功、失败和危险操作确认。
- 流程：服务控制、配置编辑、节点测速、订阅刷新、日志 Tab、连接关闭、密码及 JWT 轮换。
- 浏览器冒烟：登录、仪表盘、配置保存和日志切换。

完成前运行：

```text
npm run check
npm run build
go test ./...
golangci-lint run ./...
goimports-reviser -rm-unused -set-alias -local github.com/xuthus5/boxui -project-path ./internal ./cmd
make check-embedded-ui
```

Vitest 覆盖率门禁对 lines、functions、statements、branches 均设置为 90%。测试产生的临时目录、构建产物和进程必须清理。

## 非目标

- 不修改或新增后端 API。
- 不实现前端用户名轮换。
- 不实现后端未提供的数据持久化能力。
- 不覆盖全部 sing-box 字段为结构化表单；完整能力由 JSON 编辑器承接。
