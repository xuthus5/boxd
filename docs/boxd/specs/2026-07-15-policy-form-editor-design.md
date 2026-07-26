# 路由与 DNS 表单编辑器设计

## 目标

将当前路由与 DNS 页面共用的纯 JSON 编辑器升级为 shadcn 卡片与表单界面，同时保留完整高级 JSON 编辑能力。

本次仅修改 React + TypeScript 前端，不修改后端 API、配置结构、默认配置安装逻辑或 sing-box 启动逻辑。保存时继续提交完整配置，由后端负责校验、回滚和重载内核。

## 设计原则

- 结构化覆盖 sing-box 1.13.14 中日常使用频率较高的路由、规则集、DNS 服务器和 DNS 规则字段。
- 未结构化覆盖的复杂字段、逻辑规则和未来扩展通过高级 JSON 管理。
- 表单与高级 JSON 操作同一个完整对象，未知字段必须保留。
- 路由规则和 DNS 规则按数组顺序匹配，卡片提供上移和下移，不引入拖拽依赖。
- 严格使用现有 shadcn/Base UI 组件、语义颜色和布局工具，不新增全局自定义样式。
- 页面、组件和字段模型保持模块化，单个文件不超过 300 行，函数不超过 50 行。

## 页面总体结构

路由页和 DNS 页使用一致的页面骨架：

```text
页面标题与说明
├── 保存配置
├── 安装默认配置
└── Tabs
    ├── 可视化配置
    │   ├── 全局设置卡片
    │   ├── 资源卡片列表
    │   └── 规则卡片列表
    └── 高级 JSON
```

页面使用 `Tabs`、`Card`、`FieldGroup`、`Field`、`Input`、`Select`、`Switch`、`Textarea`、`Dialog`、`AlertDialog`、`DropdownMenu`、`Badge`、`Empty` 和现有 `JsonEditor`。

桌面端将主要操作放在卡片右上角；移动端保留首要编辑操作，将复制、排序和删除收进 `DropdownMenu`。Dialog 限制最大高度并在内容区滚动。

## 路由页面

### 全局设置卡片

结构化覆盖：

- `final`
- `find_process`
- `auto_detect_interface`
- `override_android_vpn`
- `default_interface`
- `default_mark`
- `default_domain_resolver`
- `default_network_strategy`
- `default_network_type`
- `default_fallback_network_type`
- `default_fallback_delay`

GeoIP、Geosite 兼容字段继续保留在高级 JSON 中，不作为主要表单能力推广。

### 路由规则卡片

每张卡片展示：

- 数组序号和规则类型
- 域名、IP、端口、协议、入站或规则集等匹配摘要
- 执行动作和目标出站
- 编辑、复制、上移、下移和删除操作

新增和编辑使用 Dialog，分为以下页签：

1. 基础与网络
   - `type`
   - `inbound`
   - `ip_version`
   - `network`
   - `auth_user`
   - `protocol`
   - `client`
   - `invert`
2. 域名与地址
   - `domain`
   - `domain_suffix`
   - `domain_keyword`
   - `domain_regex`
   - `source_ip_cidr`
   - `source_ip_is_private`
   - `ip_cidr`
   - `ip_is_private`
3. 端口与进程
   - `source_port`
   - `source_port_range`
   - `port`
   - `port_range`
   - `process_name`
   - `process_path`
   - `process_path_regex`
   - `package_name`
   - `user`
   - `user_id`
4. 规则集与网络环境
   - `rule_set`
   - `rule_set_ip_cidr_match_source`
   - `clash_mode`
   - `network_type`
   - `network_is_expensive`
   - `network_is_constrained`
   - `wifi_ssid`
   - `wifi_bssid`
5. 执行动作
   - 常用 `route` / outbound
   - `direct`
   - `bypass`
   - `reject`
   - `sniff`
   - `resolve`
   - `hijack-dns`
   - route-options 中常用网络策略与解析选项
6. 高级 JSON
   - 编辑单条完整规则

默认规则提供完整常用表单。逻辑规则结构化编辑 `mode` 和 `invert`，嵌套子规则通过单条规则高级 JSON 管理，避免无限嵌套表单。

### 规则集卡片

支持 `inline`、`local` 和 `remote`：

- 公共字段：`tag`、`type`、`format`
- Local：`path`
- Remote：`url`、`download_detour`、`update_interval`
- Inline：规则列表结构化管理 sing-box headless rule；Route/DNS 逻辑规则的直接子规则使用卡片与对话框，更深层逻辑嵌套和未知字段保留 JSON 兜底

卡片展示 Tag、类型、格式和路径或 URL，并提供编辑、复制和删除操作。规则集不参与顺序匹配，因此不提供排序按钮。

## DNS 页面

### 全局设置卡片

结构化覆盖：

- `final`
- `strategy`
- `disable_cache`
- `disable_expire`
- `independent_cache`
- `cache_capacity`
- `client_subnet`
- `reverse_mapping`

### FakeIP 卡片

为兼容 boxd 当前默认 DNS 配置，继续结构化支持旧式顶层 `fakeip`：

- `enabled`
- `inet4_range`
- `inet6_range`

同时支持现代 `fakeip` DNS 服务器类型。两者均按原始 JSON 形态保存，前端不擅自迁移配置。

### DNS 服务器卡片

每张卡片展示 Tag、类型、地址、端口、出站和策略摘要，提供编辑、复制和删除操作。

结构化支持以下常见类型：

- Legacy address
- `local`
- `hosts`
- `udp`
- `tcp`
- `tls`
- `quic`
- `https`
- `h3`
- `dhcp`
- `fakeip`

服务器 Dialog 分为：

1. 基础：Tag、类型、服务器、端口、路径和接口。
2. 拨号与解析：detour、domain resolver、strategy、client subnet 和常用 DialerOptions。
3. TLS 与 HTTP：TLS、服务器名称、证书校验、ALPN、HTTP method 和 headers。
4. 类型专属：Hosts、DHCP、FakeIP 等字段。
5. 高级 JSON：完整服务器对象。

旧式 `address` 服务器保留原字段，不自动转换为现代 `type` 结构。

### DNS 规则卡片

每张卡片展示数组序号、主要匹配条件、动作和目标 DNS 服务器，提供编辑、复制、上移、下移和删除操作。

结构化覆盖：

- `inbound`
- `ip_version`
- `query_type`
- `network`
- `auth_user`
- `protocol`
- `domain`
- `domain_suffix`
- `domain_keyword`
- `domain_regex`
- `source_ip_cidr`
- `source_ip_is_private`
- `ip_cidr`
- `ip_is_private`
- `source_port` / `source_port_range`
- `port` / `port_range`
- `process_name` / `process_path` / `package_name`
- `user` / `user_id`
- `outbound`
- `clash_mode`
- `rule_set`
- `network_type`
- `wifi_ssid` / `wifi_bssid`
- `invert`

常用动作覆盖选择 DNS 服务器、拒绝、预定义响应、缓存控制、TTL 重写和客户端子网。逻辑规则的嵌套子规则继续通过单条规则高级 JSON 管理。

## 数据模型与同步

页面加载完整配置后，仅取出 `route` 或 `dns` 对象作为本地编辑状态。所有结构化更新使用不可变嵌套路径操作并重新序列化到高级 JSON。

卡片 Dialog 接收单个对象副本：

- 保存时替换对应数组元素或追加新元素。
- 复制时深拷贝并插入原元素之后。
- 删除时通过 `AlertDialog` 确认。
- 上移和下移只交换相邻数组元素。

高级 JSON 成功解析后重新生成全部卡片。JSON 根节点不是对象、规则或服务器不是对象、结构化 JSON 字段无效时禁止保存。

保存仍调用现有配置 PUT API：

```text
当前完整配置
  └── 替换 route 或 dns
       └── PUT /api/config/
            ├── 后端校验
            ├── 重载内核
            └── 失败时回滚
```

安装默认路由继续先安装规则集再安装路由；安装默认 DNS 沿用现有接口。安装完成后重新拉取配置并重置本地编辑状态。

## 未知字段与类型切换

- 表单更新只修改已知字段路径，不重建整个对象。
- 未知顶层和嵌套字段始终保留。
- 类型切换只清理旧类型已知且与目标类型不兼容的字段。
- 未知规则动作、DNS server 类型或复杂对象仍可在高级 JSON 中编辑和保存。
- 前端不进行配置格式迁移，避免改变 sing-box 实际语义。

## 错误处理

- JSON 语法错误使用现有 `JsonEditor` Alert，并禁止保存。
- 结构化 JSON 字段使用 `data-invalid` 和 `aria-invalid`。
- 必填 Tag、URL、路径、服务器或动作缺失时在 Dialog 内提示并禁止确认。
- 数字字段拒绝 `NaN`，端口和缓存容量不允许负数。
- API 错误、内核校验失败和回滚继续使用现有 toast 行为。
- 默认配置安装失败时保留当前本地内容，不显示成功状态。

## 测试与验收

测试至少覆盖：

- 路由和 DNS 页面不再默认显示纯 JSON 编辑器。
- 全局字段表单修改后保存完整配置且保留未知字段。
- 路由规则、规则集、DNS server 和 DNS rule 的新增、编辑、复制与删除。
- 路由和 DNS 规则上移、下移及边界按钮禁用。
- 类型切换只清理不兼容已知字段。
- 卡片摘要展示匹配条件、动作和目标。
- 无效结构化 JSON 跨页签持续阻止保存。
- 高级 JSON 修改后卡片立即同步。
- 安装默认配置成功、失败和回滚行为保持不变。
- 320px 移动端无横向溢出，Dialog 内容可滚动。
- shadcn Card、Dialog、Field、Select、Tabs、AlertDialog、DropdownMenu 和 Empty 组合符合 Base UI 规范。
- 前端 statements、branches、functions 和 lines 覆盖率均不低于 90%。
- Go 门禁、Playwright、生产构建和嵌入资源验证通过。

完成后按项目约定提交 Git，重新构建二进制，替换 `/usr/local/bin/boxd` 并重启验证 `boxd.service`。
