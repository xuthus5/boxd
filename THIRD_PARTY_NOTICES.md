# Third-Party Notices

boxd 自有代码按仓库根目录的 Apache License 2.0 授权。正式发布的 boxd 二进制静态链接了 GPL-3.0 组件，因此二进制分发必须同时满足 GPL-3.0 的适用义务。本文件不是法律意见；商业分发方应结合实际修改和交付方式进行合规审查。

## GPL-3.0 组件

- `github.com/sagernet/sing-box` — GPL-3.0
- `github.com/sagernet/sing` — GPL-3.0

发布归档必须包含来自 sing-box 模块的完整 GPL-3.0 文本 `LICENSE-GPL-3.0`。发布对应源码可从本 boxd Git tag、`go.mod`、`go.sum` 以及其中固定的上游版本重建。分发经过修改的二进制时，分发方还需提供相应修改后的完整源码和构建信息。

## 主要宽松许可证依赖

- `github.com/go-chi/chi/v5` — MIT
- `github.com/golang-jwt/jwt/v5` — MIT
- `go.etcd.io/bbolt` — MIT
- React、Vite、TanStack Query、Tailwind CSS、CodeMirror、Lucide 与 shadcn/ui 相关依赖 — MIT 或其他兼容的宽松许可证，具体版本见 `ui/package-lock.json`
- Go 扩展模块与其他传递依赖 — 具体版本见 `go.sum`

每次发布生成的 SPDX SBOM 是该制品依赖版本的权威机器可读清单。许可证文本及声明仍以各上游项目随附内容为准。

## 外部规则与数据

boxd 可以下载或引用 Loyalsoldier、SagerNet 等第三方规则集。规则数据不属于 boxd 自有代码，授权条款由对应数据源决定。商业交付方应在预装、镜像缓存或再分发这些数据前单独确认其许可。
