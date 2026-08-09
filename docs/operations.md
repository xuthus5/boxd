# boxd 运维手册

适用对象：单节点自托管部署（个人 / 小团队）。不覆盖多租户 SaaS。

## 1. 质量门禁

- [ ] `./scripts/check-go.sh` 通过（测试、race、覆盖率 ≥90%、golangci-lint、govulncheck）
- [ ] `cd ui && npm run check` 通过（typecheck、lint、coverage ≥90%、build）
- [ ] `./scripts/check-embedded-ui.sh` 通过
- [ ] Docker 镜像构建通过（可选但推荐）
- [ ] 关键 Git tag 与 `git describe` 版本可追溯

## 2. 安全发布前检查

- [ ] **禁止**使用默认密码上线；首次登录后必须轮换
- [ ] 面板在默认密码状态下会强制跳转到「应用设置」完成轮换
- [ ] 轮换 JWT 签名密钥后，确认旧会话失效
- [ ] 生产环境启用 HTTPS（见下方 TLS 部署）
- [ ] `BOXD_CORS_ALLOWED_ORIGINS` 仅配置可信来源
- [ ] systemd 单元以非 root 服务用户运行，二进制权限 `root:boxd 0750`
- [ ] 数据目录 `/var/lib/boxd` 权限收敛（目录 0700，关键文件 0600）

## 3. ICMP 测速网络权限

ICMP 测速使用原始套接字，需要 `CAP_NET_RAW`。缺失时测速报
`icmp raw socket requires CAP_NET_RAW`（TCP/HTTP 测速不受影响）。

| 部署形态 | 处理方式 |
| --- | --- |
| systemd（deb/rpm） | `boxd.service` 已内置 `AmbientCapabilities=CAP_NET_RAW`，无需额外操作 |
| 桌面应用（rpm/deb，非 root 运行） | 安装包 `%post` 已自动 `setcap cap_net_raw+ep` 授予二进制能力，无需额外操作；升级包时同样自动生效 |
| 桌面应用（手动安装 / AppImage） | 运行 `sudo ./scripts/grant-desktop-icmp.sh`（自动解析桌面用户 GID，写入 `ping_group_range`），或 `sudo ./scripts/grant-desktop-icmp.sh $USER setcap` |
| 直接运行二进制 | 以 root 运行，或为服务用户授予 `CAP_NET_RAW`（如 `AmbientCapabilities`） |
| 容器 | `docker run --cap-add NET_RAW ...`（容器内进程以 root 运行，能力生效） |

自建 systemd unit 的最小配置：

```ini
[Service]
User=boxd
CapabilityBoundingSet=CAP_NET_RAW
AmbientCapabilities=CAP_NET_RAW
```

验证 ICMP 测速可用：

```bash
curl -s 'http://127.0.0.1:9091/api/v1/test/probe' \
  -H 'Authorization: Bearer <token>' \
  -d '{"server":"1.1.1.1","test_type":"icmp"}'
```

## 4. TLS 推荐部署

boxd 支持内置 TLS：

```bash
BOXD_TLS_CERT=/etc/boxd/tls/fullchain.pem \
BOXD_TLS_KEY=/etc/boxd/tls/privkey.pem \
BOXD_LISTEN=[::]:9091 \
/usr/local/bin/boxd
```

或使用反向代理（推荐在已有证书体系时）：

1. 反代终止 TLS（Caddy / Nginx / Traefik）
2. 上游仅监听本机 `127.0.0.1:9091`
3. 反代开启 WebSocket/SSE 透传（日志与流量流式接口需要）

最低要求：

- 证书与私钥文件权限 0600，属主 root 或服务可读用户
- 同时配置 cert/key，缺一不可
- 对外只暴露 HTTPS，不暴露明文管理口

## 5. 默认密码与账号

密码优先级：数据库哈希 → 首次 `BOXD_PASSWORD` → `admin123`。

发布动作：

1. 安装时通过 `BOXD_PASSWORD` 注入强初始密码，或
2. 首次登录后立即在「应用设置」轮换
3. 确认设置页不再显示默认密码告警
4. 轮换后使用新密码重新登录

## 6. 备份与恢复演练

面板「设置 → 数据备份」可导出同格式归档（需登录）。恢复仍仅支持 CLI。

创建备份：

```bash
/usr/local/bin/boxd --backup /var/backups/boxd/boxd-$(date +%F).tar.gz
```

恢复：

```bash
systemctl stop boxd.service
/usr/local/bin/boxd --restore /var/backups/boxd/boxd-YYYY-MM-DD.tar.gz --data-dir /var/lib/boxd --config /etc/sing-box/config.json
systemctl start boxd.service
```

验收：

- [ ] 备份产物非空且可解压
- [ ] 恢复后可登录
- [ ] 订阅 / 设置 / 内核配置关键数据在
- [ ] 服务 `active (running)`，`/health` 返回正常

## 7. 升级回滚

升级：

1. 备份：`boxd --backup ...`
2. 构建或获取新版本二进制
3. `install -o root -g boxd -m 0750 bin/boxd /usr/local/bin/boxd`
4. `systemctl restart boxd.service`
5. 检查：`systemctl is-active boxd.service`、`journalctl -u boxd.service -n 50`、登录面板

回滚：

1. 停服务
2. 还原上一版二进制
3. 如配置不兼容，使用升级前备份 `--restore`
4. 启动并验证

## 8. 长稳与真链路验收

短时 smoke（每次发布）：

```bash
export BOXD_PASSWORD='***'
./scripts/e2e-live.sh
./scripts/soak-runtime.sh --duration 300 --interval 10
```

GA 前长稳（推荐）：

```bash
./scripts/soak-runtime.sh \
  --duration 86400 \
  --interval 60 \
  --max-alloc-growth-mb 128 \
  --max-sys-growth-mb 192 \
  --max-goroutine-growth 80 \
  --output /var/log/boxd/soak-24h.csv
```

通过标准：

- `E2E_LIVE_PASS`
- `SOAK_PASS`
- 发布窗口内无启动失败、无配置回滚风暴

## 9. 发布说明模板

- 版本号：
- 对应 commit：
- 兼容内核版本：
- 变更摘要：
- 已知限制：
- 备份文件路径：
- 长稳报告路径：
