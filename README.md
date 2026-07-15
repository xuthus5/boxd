# boxui

sing-box Web 管理面板，提供节点管理、订阅、配置编辑、流量监控等功能。

## 技术栈

- 后端：Go + chi + bbolt + sing-box
- 前端：React 19 + Vite + shadcn/ui + Tailwind CSS
- 认证：JWT（HS256）

## 快速开始

```bash
# 构建前端并嵌入二进制
make build

# 运行（JWT 密钥首次启动自动生成并持久化到数据库）
BOXUI_PASSWORD=your-initial-password ./bin/boxui

# 开发模式（前后端分离）
make dev
```

## 配置

| 环境变量                     | Flag         | 默认值                      | 说明                                     |
| ---------------------------- | ------------ | --------------------------- | ---------------------------------------- |
| `BOXUI_LISTEN`               | `--listen`   | `[::]:9091`                 | 监听地址                                 |
| `BOXUI_PORT`                 | -            | `9091`                      | 监听端口号                               |
| `BOXUI_CONFIG`               | `--config`   | `/etc/sing-box/config.json` | sing-box 配置路径                        |
| `BOXUI_DATA_DIR`             | `--data-dir` | `/var/lib/boxui`            | 数据目录                                 |
| `BOXUI_USERNAME`             | `--username` | `admin`                     | 登录用户名                               |
| `BOXUI_PASSWORD`             | `--password` | `admin123`                  | 首次初始化密码；数据库已有密码后不再覆盖 |
| `BOXUI_CORS_ALLOWED_ORIGINS` | -            | -                           | CORS 允许的源（逗号分隔）                |


## 面板能力

- **仪表盘**：服务启停/重启、实时流量、内存 GC、DNS/FakeIP 维护、最近日志
- **代理配置**：入站、出站与默认出站
- **流量策略**：路由规则、规则集默认安装、DNS 服务器与 DNS 规则
- **节点与订阅**：响应式节点卡片、按来源分组、节点级与分组批量 TCP/HTTP/ICMP 测速，订阅与手动节点管理自动同步配置
- **运行观测**：内核日志、应用日志、活跃连接管理
- **sing-box 高级配置**：Endpoints、Experimental、完整 JSON 配置编辑
- **应用设置**：浅色/深色主题、中英文切换、密码轮换、JWT 密钥轮换、测速地址、内核自启

## 预置规则

路由配置页提供一键安装常用规则，覆盖常见客户端场景：

- **嗅探**：协议探测后再按规则分流
- **劫持 DNS**：把 DNS 请求交给内核 DNS 模块
- **绕过局域网**：私有 IP 走直连
- **绕过 ICMP**：ping 走直连，避免被拦截
- **屏蔽 QUIC**：拒绝 UDP 443，回落到 TCP
- **屏蔽广告**：Loyalsoldier reject / geosite-category-ads-all → block
- **中国域名**：Loyalsoldier direct / geosite-cn → direct
- **中国 IP**：geoip-cn → direct
- **中国 Google Play**：geosite-google-play → proxy
- **代理**：Loyalsoldier proxy → proxy

规则集默认安装两类：Loyalsoldier 文本规则集（本地转换）与 SagerNet 二进制规则集（远程缓存）。

## JWT 密钥管理

- 首次启动时若数据库无密钥，则随机生成并存入 bbolt（`settings` bucket 的 `jwt_secret` 键），不再读取任何环境变量。
- 密钥持久化在本地数据库，无需环境变量。
- 可在「设置 → JWT 签名密钥」页面查看脱敏状态并轮换；轮换后所有已登录会话立即失效。

## 管理员密码

- 管理员密码使用 Argon2id 哈希存入 bbolt，不保存明文。
- 密码优先级为：数据库中的密码哈希 → 首次启动时的 `BOXUI_PASSWORD` → `admin123`。
- 后台修改密码后，环境变量不会在重启时覆盖新密码。
- 使用默认密码时设置页面会持续显示警告；修改密码会使所有现有登录失效。

## Docker

```bash
docker build -t boxui .
docker run -p 9091:9091 -e BOXUI_PASSWORD=your-password boxui
```

## 开发

```bash
# 前端开发
cd ui && npm ci && npm run dev

# 后端
go run ./cmd/boxui/

# 测试与生产构建
go test ./...
cd ui && npm run check

# 浏览器端到端测试（Vite 预览服务 + 浏览器 API Mock）
npm run e2e:install
npm run e2e

# Lint
golangci-lint run ./...
goimports-reviser -rm-unused -set-alias -local github.com/xuthus5/boxui -project-path ./internal ./cmd
```

完整质量门禁：

```bash
make check-go          # test、race、>=90% 覆盖率、lint、govulncheck
make check-ui          # >=90% 业务覆盖率、lint、生产构建
make check-embedded-ui # 验证最终嵌入资源及动态依赖完整性
```

E2E 不连接本机 9091、systemd 或生产数据；Playwright 关闭时会停止 Vite 预览服务。

创建 Linux amd64 发布包：

```bash
./scripts/package-release.sh v1.0.0 release
```

推送 `v*` Git tag 后，Release workflow 会在全部质量门禁通过后发布归档、SHA-256 和 SPDX SBOM。

## 许可证

BoxUI 自有代码采用 [MIT License](LICENSE)。正式二进制静态链接 GPL-3.0 的 sing-box，分发时还需遵守 [第三方声明](THIRD_PARTY_NOTICES.md) 中说明的 GPL-3.0 义务。
