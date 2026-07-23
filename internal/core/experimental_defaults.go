package core

// ExperimentalDefaultsInstaller 安装常用 experimental 段默认值。
type ExperimentalDefaultsInstaller interface {
	Install(cfg map[string]any) (*ExperimentalDefaultsResult, error)
}

// ExperimentalDefaultsResult 返回合并后的 experimental 与本次写入的子集。
type ExperimentalDefaultsResult struct {
	Experimental map[string]any
	Installed    map[string]any
}

// DefaultExperimentalInstaller 确保 clash_api 可用的最小配置。
type DefaultExperimentalInstaller struct{}

// NewDefaultExperimentalInstaller 创建默认 experimental 安装器。
func NewDefaultExperimentalInstaller() *DefaultExperimentalInstaller {
	return &DefaultExperimentalInstaller{}
}

// Install 补齐 experimental.clash_api（本机控制器 + Rule 默认模式），不覆盖已有字段。
func (i *DefaultExperimentalInstaller) Install(cfg map[string]any) (*ExperimentalDefaultsResult, error) {
	experimental := copyMap(asMap(cfg["experimental"]))
	clashAPI := copyMap(asMap(experimental["clash_api"]))
	installed := map[string]any{}
	ensureString(clashAPI, installed, "external_controller", "127.0.0.1:9090")
	ensureString(clashAPI, installed, "default_mode", "rule")
	experimental["clash_api"] = clashAPI
	if len(installed) > 0 {
		installed = map[string]any{"clash_api": installed}
	}
	return &ExperimentalDefaultsResult{
		Experimental: experimental,
		Installed:    installed,
	}, nil
}

func asMap(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func ensureString(target, installed map[string]any, key, value string) {
	if existing, ok := target[key].(string); ok && existing != "" {
		return
	}
	target[key] = value
	installed[key] = value
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return cloneAnyMap(in)
}
