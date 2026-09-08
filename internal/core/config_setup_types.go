package core

import "errors"

var ErrSetupStale = errors.New("configuration changed since preview; preview the changes again")

// SetupRequest 描述用户明确选择的模块及入站接入方式。
type SetupRequest struct {
	Modules      []string `json:"modules"`
	InboundMode  string   `json:"inbound_mode"`
	IPv6         string   `json:"ipv6"`
	ResetInvalid bool     `json:"reset_invalid"`
	SourceHash   string   `json:"source_hash,omitempty"`
}

// SetupEnvironment 为预览提供运行环境，不启动内核或修改系统网络。
type SetupEnvironment struct {
	DataDir       string
	Capabilities  NetworkCapabilities
	KernelRunning bool
}

// SetupPlan 是基于特定源配置生成的完整、可校验预览。
type SetupPlan struct {
	SourceHash    string         `json:"source_hash"`
	CurrentConfig map[string]any `json:"current_config"`
	Config        map[string]any `json:"config"`
	Modules       []string       `json:"modules"`
	Warnings      []string       `json:"warnings"`
	WillRestart   bool           `json:"will_restart"`
}

// ConfigModuleStatus 区分模块的存在、缺失和语义错误。
type ConfigModuleStatus struct {
	ID           string   `json:"id"`
	State        string   `json:"state"`
	Count        int      `json:"count"`
	Dependencies []string `json:"dependencies"`
	Message      string   `json:"message,omitempty"`
}

// SetupListener 仅返回客户端接入所需信息，不包含入站凭证。
type SetupListener struct {
	Tag        string   `json:"tag"`
	Type       string   `json:"type"`
	Listen     string   `json:"listen"`
	ListenPort int      `json:"listen_port,omitempty"`
	Address    []string `json:"address,omitempty"`
}

// SetupStatus 汇总配置模块和运行状态，不能代替客户端可达测试。
type SetupStatus struct {
	SourceHash    string               `json:"source_hash"`
	Capabilities  NetworkCapabilities  `json:"capabilities"`
	Modules       []ConfigModuleStatus `json:"modules"`
	Listeners     []SetupListener      `json:"listeners"`
	KernelRunning bool                 `json:"kernel_running"`
	ProxyReady    bool                 `json:"proxy_ready"`
	ConfigError   string               `json:"config_error,omitempty"`
}

type SetupError struct {
	Code    string
	Message string
}

func (e *SetupError) Error() string { return e.Message }

var setupModuleOrder = []string{"outbounds", "rule_sets", "inbounds", "dns", "route", "experimental"}

var setupModuleDependencies = map[string][]string{
	"inbounds":     {"outbounds", "dns", "route"},
	"outbounds":    {},
	"rule_sets":    {},
	"dns":          {"outbounds", "rule_sets"},
	"route":        {"outbounds", "rule_sets", "dns"},
	"experimental": {},
}
