package core

type RuleSetSource struct {
	Tag      string
	FileName string
	URL      string
	// Format 源码格式："" 或 "source" 表示文本列表（下载后转换为 JSON），
	// "binary" 原样保存 .srs，"cidr" 合并 IPv4/IPv6 列表后编译为 .srs。
	Format string
}

// builtinRuleSetSources 保留全部可更新来源，包括旧配置使用的 SagerNet 规则。
// 默认使用 Loyalsoldier 域名列表与 china-operator-ip 的完整 IPv4/IPv6 CIDR。
func builtinRuleSetSources() []RuleSetSource {
	return []RuleSetSource{
		{
			Tag:      "loyalsoldier-direct",
			FileName: "loyalsoldier-direct.json",
			URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt",
		},
		{
			Tag:      "loyalsoldier-proxy",
			FileName: "loyalsoldier-proxy.json",
			URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt",
		},
		{
			Tag:      "loyalsoldier-reject",
			FileName: "loyalsoldier-reject.json",
			URL:      "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/reject-list.txt",
		},
		{
			Tag:      "geosite-cn",
			FileName: "geosite-cn.srs",
			URL:      "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
			Format:   "binary",
		},
		{
			Tag:      "geoip-cn",
			FileName: "geoip-cn.srs",
			URL:      "https://raw.githubusercontent.com/gaoyifan/china-operator-ip/ip-lists/china.txt",
			Format:   "cidr",
		},
		{
			Tag:      "geosite-google-play",
			FileName: "geosite-google-play.srs",
			URL:      "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-google-play.srs",
			Format:   "binary",
		},
		{
			Tag:      "geosite-category-ads-all",
			FileName: "geosite-category-ads-all.srs",
			URL:      "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
			Format:   "binary",
		},
	}
}

func defaultRuleSetSources() []RuleSetSource {
	sources := make([]RuleSetSource, 0)
	for _, src := range builtinRuleSetSources() {
		switch src.Tag {
		case "loyalsoldier-direct", "loyalsoldier-proxy", "loyalsoldier-reject", "geoip-cn":
			sources = append(sources, src)
		}
	}
	return sources
}

// BuiltinRuleSetTags 返回全部可更新的内置标签，包含旧版默认规则。
func BuiltinRuleSetTags() []string {
	sources := builtinRuleSetSources()
	tags := make([]string, 0, len(sources))
	for _, src := range sources {
		tags = append(tags, src.Tag)
	}
	return tags
}

func DefaultRemoteRuleSetInterval() string { return "1d" }

func BuiltinLocalRuleSetTags() []string {
	return BuiltinRuleSetTags()
}

func (i *LoyalsoldierRuleSetInstaller) RuleSetDir() string { return i.ruleSetDir }

func (i *LoyalsoldierRuleSetInstaller) SourceByTag(tag string) (RuleSetSource, bool) {
	for _, src := range i.sources {
		if src.Tag == tag {
			return src, true
		}
	}
	// 默认仅安装四组规则，既有配置中的其他内置规则仍可更新。
	for _, src := range builtinRuleSetSources() {
		if src.Tag == tag {
			return src, true
		}
	}
	return RuleSetSource{}, false
}

func (i *LoyalsoldierRuleSetInstaller) IsBuiltinLocal(tag string) bool {
	_, ok := i.SourceByTag(tag)
	return ok
}
