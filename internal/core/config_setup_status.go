package core

import (
	"errors"
	"os"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

// ReadSetupSource 将不存在的配置视为待初始化；其余读取错误必须报告。
func ReadSetupSource(path string) ([]byte, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return body, err
}

// InspectSetup 保留错误状态，使损坏配置也能进入显式恢复流程。
func InspectSetup(body []byte, env SetupEnvironment) SetupStatus {
	status := SetupStatus{SourceHash: ConfigContentHash(body), Capabilities: env.Capabilities,
		Modules: []ConfigModuleStatus{}, Listeners: []SetupListener{}, KernelRunning: env.KernelRunning}
	cfg, err := setupConfigObject(body)
	if err != nil {
		status.ConfigError = err.Error()
		cfg = map[string]any{}
	}
	for _, id := range setupModuleOrder {
		count := setupModuleCount(cfg, id)
		state := "missing"
		if count > 0 {
			state = "ready"
		}
		status.Modules = append(status.Modules, ConfigModuleStatus{ID: id, State: state, Count: count,
			Dependencies: append([]string{}, setupModuleDependencies[id]...)})
	}
	if status.ConfigError == "" && len(cfg) > 0 {
		applySetupDiagnostics(&status, AnalyzeConfig(body))
	}
	status.Listeners = setupListeners(cfg)
	checkSetupInboundCapabilities(&status)
	status.ProxyReady = setupProxyReady(cfg)
	for _, module := range status.Modules {
		if module.ID == "outbounds" && module.State == "invalid" {
			status.ProxyReady = false
		}
	}
	return status
}

func checkSetupInboundCapabilities(status *SetupStatus) {
	message := ""
	for _, listener := range status.Listeners {
		if listener.Type != "tun" {
			continue
		}
		if !status.Capabilities.TUNAvailable {
			message = status.Capabilities.TUNReason
			break
		}
		for _, address := range listener.Address {
			unavailable := status.Capabilities.IPv6Reason == NetworkIPv6Disabled ||
				status.Capabilities.IPv6Reason == NetworkIPv6Unsupported
			if strings.Contains(address, ":") && unavailable {
				message = status.Capabilities.IPv6Reason
			}
		}
	}
	if message == "" {
		return
	}
	for index := range status.Modules {
		if status.Modules[index].ID == "inbounds" {
			status.Modules[index].State = "invalid"
			status.Modules[index].Message = message
		}
	}
}

func setupModuleCount(cfg map[string]any, id string) int {
	var value any
	switch id {
	case "inbounds", "outbounds":
		value = cfg[id]
	case "dns":
		value = objectValue(cfg["dns"])["servers"]
	case "route":
		value = objectValue(cfg["route"])["rules"]
	case "rule_sets":
		value = objectValue(cfg["route"])["rule_set"]
	case "experimental":
		options := objectValue(cfg["experimental"])
		count := 0
		if objectValue(options["clash_api"]) != nil {
			count++
		}
		if objectValue(options["cache_file"])["enabled"] == true {
			count++
		}
		return count
	}
	items, _ := value.([]any)
	return len(items)
}

func applySetupDiagnostics(status *SetupStatus, diagnostics model.ConfigDiagnostics) {
	for _, issue := range diagnostics.Issues {
		if issue.Severity != model.ConfigDiagnosticSeverityError {
			continue
		}
		module := setupDiagnosticModule(issue.Path)
		if status.ConfigError == "" {
			status.ConfigError = issue.Path + ": " + issue.Code
		}
		for index := range status.Modules {
			if module == "" || status.Modules[index].ID == module {
				status.Modules[index].State = "invalid"
				status.Modules[index].Message = issue.Path + ": " + issue.Code
			}
		}
	}
}

func setupDiagnosticModule(path string) string {
	if strings.HasPrefix(path, "route.rule_set") {
		return "rule_sets"
	}
	for _, id := range setupModuleOrder {
		if strings.HasPrefix(path, id+".") || strings.HasPrefix(path, id+"[") || path == id {
			return id
		}
	}
	return ""
}

func setupProxyReady(cfg map[string]any) bool {
	for _, item := range configuredEgresses(cfg) {
		outbound, _ := item.(map[string]any)
		if stringValue(outbound["tag"]) == "" {
			continue
		}
		if isProxyCandidate(outbound) {
			return true
		}
	}
	return false
}

func setupListeners(cfg map[string]any) []SetupListener {
	const maxPort = 65535
	listeners := []SetupListener{}
	inbounds, _ := cfg["inbounds"].([]any)
	for _, value := range inbounds {
		inbound, ok := value.(map[string]any)
		if !ok {
			continue
		}
		listener := SetupListener{Tag: stringValue(inbound["tag"]), Type: stringValue(inbound["type"]),
			Listen: stringValue(inbound["listen"]), Address: asStringSlice(inbound["address"])}
		if port, ok := inbound["listen_port"].(float64); ok && port > 0 && port <= maxPort {
			listener.ListenPort = int(port)
		}
		listeners = append(listeners, listener)
	}
	return listeners
}
