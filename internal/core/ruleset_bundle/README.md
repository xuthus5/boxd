# 离线默认规则集

首次启动仅解压这四组完整快照，不访问网络。`.gz` 使用无文件名、零时间戳的 gzip 压缩；`manifest.json` 记录解压数据的 SHA-256 和上游时间。安装后保留上游文件时间，有效的现存缓存不会被快照覆盖。在线更新沿用以下相同来源。

## 域名列表

来源：[Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat/tree/1f22bb4351860cad7195407a330bdb891ba13b3c)，提交 `1f22bb4351860cad7195407a330bdb891ba13b3c`，发布时间 `2026-09-07T23:54:59Z`。原始 URL 为 `https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/<commit>/<file>`。

| 嵌入文件 | 上游文件 | 原始行数 | 原始 SHA-256 |
| --- | --- | ---: | --- |
| `loyalsoldier-direct.gz` | `direct-list.txt` | 111167 | `ae583feaf41b9cbe02576c6a4fb8f533eb9b5291a8120f592bad47be46e73cca` |
| `loyalsoldier-proxy.gz` | `proxy-list.txt` | 27237 | `8a4f40e56e53517a4d2fb92008b347eb8516894078284e11bbcc955c1f821003` |
| `loyalsoldier-reject.gz` | `reject-list.txt` | 186028 | `f3954f28b913f95dfa188c6bac2277e202410ef7e29790b4a7090d3aff52d7b9` |

保持全部 `full:`、`keyword:`、`regexp:` 与域名后缀规则，安装时转换为现有同名 JSON。许可证为 GPL-3.0，完整副本见 [LICENSE-Loyalsoldier](LICENSE-Loyalsoldier)，取自项目提交 [`5c20d2eb5a65b171816949010ede67a27326cbe6`](https://github.com/Loyalsoldier/v2ray-rules-dat/blob/5c20d2eb5a65b171816949010ede67a27326cbe6/LICENSE)。聚合数据的贡献者、生成方式与来源见该提交的 [README](https://github.com/Loyalsoldier/v2ray-rules-dat/blob/5c20d2eb5a65b171816949010ede67a27326cbe6/README.md)，包括 v2fly、dnsmasq-china-list、GFWList、EasyList、AdGuard、Peter Lowe 与 Dan Pollock。

## 中国大陆 IP

来源：[gaoyifan/china-operator-ip](https://github.com/gaoyifan/china-operator-ip/tree/96cdd243a6bd09619ce226727a989afbac838486)，提交 `96cdd243a6bd09619ce226727a989afbac838486`，发布时间 `2026-09-07T07:38:18Z`。原始 URL 为 `https://raw.githubusercontent.com/gaoyifan/china-operator-ip/<commit>/<file>`。

| 上游文件 | 原始行数 | 原始 SHA-256 |
| --- | ---: | --- |
| `china.txt` | 6238 | `ff3c662388ed626b7d14eaa3230afc39b0aafd9421ecc9f6f9e44a4b0d0b5d79` |
| `china6.txt` | 3401 | `eff0d8f9c426b3c04cca66b2658b5895b998598ba3bbb87aa24c624e91d150d9` |

`geoip-cn.gz` 是上述 IPv4、IPv6 原始文件按顺序直接拼接后的 gzip；解压 SHA-256 为 `a25648c221f8b12b41ce168c739ebc3a7b1d95d08f199aa31628a33f2e61822b`。安装和在线更新均校验两种地址族并使用项目内 sing-box 编译为 `geoip-cn.srs`，合并重复和重叠 CIDR。标签、文件名与配置中的 `binary` 格式保持兼容。

许可证为 MIT，Copyright (c) 2017 Yifan Gao；完整副本见 [LICENSE-china-operator-ip](LICENSE-china-operator-ip)，取自项目提交 [`bcdccb71101cefc944dc178b9808f21ecebd5b02`](https://github.com/gaoyifan/china-operator-ip/blob/bcdccb71101cefc944dc178b9808f21ecebd5b02/LICENSE)。

更新快照时固定新提交，下载对应原始文件，核对完整性，使用 `gzip -n -9` 重新压缩，并同步修改 manifest、此来源记录和完整性测试。无需重新安装已有规则集来更新快照版本。
