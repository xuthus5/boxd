# sing-box 版本策略

## 当前基线

- 模块：`github.com/sagernet/sing-box`
- 锁定版本：`v1.14.4`
- 兼容线：`v1.14.x`
- 配套模块：以 `go.mod` 中 Go Modules 解析结果为准，例如 `github.com/sagernet/sing`。

## 设计原则

1. `go.mod` 是精确版本的事实来源。生产构建不使用 `@latest`，也不依赖浮动分支。
2. Go Modules 本身按精确版本构建。即使同一条 minor 线出现新补丁，只有显式执行 `go get` 才会升级。
3. Driver 在 `Factory.Open` 前执行兼容性检查，从二进制 build info 读取实际 sing-box 模块版本。
4. 只接受 `v1.14.x`。`v1.14.5`、`v1.14.10` 这类补丁版本自动通过；`v1.15.x`、`v2.x` 默认拒绝。
5. 如果 sing-box 通过 `-X github.com/sagernet/sing-box/constant.Version=...` 注入版本，Driver 会优先使用注入值并执行同样校验。
6. 无法从 build info 确定版本时失败关闭，返回 `kernel_engine_incompatible`，避免静默运行未知内核。

实现位置：`internal/driver/singbox/version.go`。

## 补丁升级：v1.14.x

补丁升级不要求修改 Go 代码，按以下顺序执行：

```powershell
go get github.com/sagernet/sing-box@v1.14.5
go mod tidy
go list -m -f '{{.Path}} {{.Version}}' github.com/sagernet/sing-box
make engine-version
go test ./...
go test -race ./...
go vet ./...
go build -tags with_utls,with_quic -o bin/boxd.exe ./cmd/boxd
```

提交 `go.mod`、`go.sum` 和必要的适配代码。不要单独修改 `go.sum`，也不要手工编辑间接依赖版本。

建议同时检查：

```powershell
go list -m all | Select-String -Pattern '^github.com/sagernet/'
```

如果补丁升级同时拉高了 `sing-tun`、`sing`、`quic-go` 等核心依赖，需要额外验证 TUN、Mixed 入站、DNS、路由和代理编排。

## 次版本升级：v1.15.x

次版本升级视为 Driver 兼容性变更，不允许依赖机器人自动合并。流程至少包括：

1. 新建独立升级分支，并记录升级原因、上游变更和已知行为差异。
2. 修改 `internal/driver/singbox/version.go` 中的 `SupportedMinor` 和 `SupportedSeries`。
3. 审查 sing-box `option`、`include`、`adapter`、日志和运行控制 API 的破坏性变化。
4. 补齐配置生成、诊断、入站、出站、路由、DNS、连接观测和生命周期回归测试。
5. 在三个宿主形态上完成构建与冒烟验证后再合并。

如果新旧版本需要并行过渡，可以把兼容线升级为显式版本区间，但必须保持公开 `kernel` 契约不变。

## 回退

回退时同时回退 `go.mod`、`go.sum` 和 Driver 适配代码，然后重新执行 `go mod tidy` 与全量测试。不要只回退 `go.mod` 中的单个版本，否则可能留下不匹配的传递依赖。

## 自动化边界

- 允许自动创建补丁版本升级 PR，但必须运行 `make engine-version` 和全量测试。
- 不允许自动合并跨 minor 或 major 的升级。
- 不允许修改 `SupportedSeries` 来绕过兼容性测试。
