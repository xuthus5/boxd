package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

// BridgeRequest 描述一次传输无关的用例调用。
type BridgeRequest struct {
	Path   string          `json:"path"`
	Method string          `json:"method,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// BridgeResponse 描述用例调用结果。
type BridgeResponse struct {
	Data   any    `json:"data"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

// BoxdBridgeService 提供 REST 风格路径到用例层的通用分发，供前端 transport 复用。
type BoxdBridgeService struct {
	rt *desktopRuntime
}

func newBoxdBridgeService(rt *desktopRuntime) *BoxdBridgeService {
	return &BoxdBridgeService{rt: rt}
}

// Call 按路径分派到对应用例方法。返回统一信封，前端无需感知 bindings 细节。
func (s *BoxdBridgeService) Call(req BridgeRequest) (BridgeResponse, error) {
	if s.rt == nil || s.rt.svc == nil {
		return BridgeResponse{}, errNotReady()
	}
	return s.dispatch(context.Background(), req)
}

func (s *BoxdBridgeService) dispatch(ctx context.Context, req BridgeRequest) (BridgeResponse, error) {
	path := strings.TrimSpace(req.Path)
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}

	// 写操作：service 控制
	if method == "POST" {
		switch path {
		case "/api/service/start":
			return errResult(s.rt.svc.Service().ServiceStart(ctx))
		case "/api/service/stop":
			return errResult(s.rt.svc.Service().ServiceStop(ctx))
		case "/api/service/restart":
			return errResult(s.rt.svc.Service().ServiceRestart(ctx))
		case "/api/runtime/gc":
			return errResult(s.rt.svc.Kernel().KernelGC(ctx))
		case "/api/runtime/dns/flush":
			return errResult(s.rt.svc.Runtime().FlushDNS(ctx))
		case "/api/runtime/fakeip/flush":
			return errResult(s.rt.svc.Runtime().FlushFakeIP(ctx))
		case "/api/config/dns/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultDNS(ctx))
		case "/api/config/rule-sets/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultRuleSets(ctx))
		case "/api/config/outbounds/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultOutbounds(ctx))
		case "/api/config/inbounds/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultInbounds(ctx))
		case "/api/config/experimental/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultExperimental(ctx))
		case "/api/config/route/defaults":
			return okResult(s.rt.svc.Config().InstallDefaultRouteRules(ctx))
		case "/api/nodes/sync-config":
			return errResult(syncNodesConfig(s.rt))
		case "/api/config/rule-sets/update":
			req, err := bridgeBody[core.RuleSetUpdateRequest](req.Body)
			if err != nil {
				return errResult(err)
			}
			return okResult(s.rt.svc.RuleSets().Update(ctx, req))
		case "/api/auth/login":
			return okResult(loginBridge(s.rt, req.Body))
		case "/api/auth/logout":
			return errResult(s.rt.svc.Auth().Logout(ctx))
		case "/api/import/link":
			return okResult(importLinkBridge(s.rt, req.Body))
		case "/api/import/save":
			return okResult(importSaveBridge(s.rt, req.Body))
		case "/api/nodes/test":
			return okResult(testRunBridge(s.rt, req.Body))
		case "/api/nodes/test-batch":
			return okResult(testBatchBridge(s.rt, req.Body))
		case "/api/runtime/dns/probe":
			return okResult(probeDNSBridge(s.rt, req.Body))
		case "/api/runtime/dns/probe-batch":
			return okResult(probeDNSBatchBridge(s.rt, req.Body))
		case "/api/subscriptions/refresh-all":
			return okResult(s.rt.svc.Subscriptions().RefreshAll(ctx))
		}
	}

	// 写操作：带请求体
	if method == "PUT" || method == "POST" {
		switch path {
		case "/api/config/", "/api/config/raw":
			return okResult(s.rt.svc.Config().ApplyConfig(ctx, req.Body, "desktop"))
		case "/api/config/validate":
			return errResult(s.rt.svc.Config().ValidateConfig(ctx, req.Body, "validate"))
		case "/api/settings/password":
			return okResult(changePasswordBridge(s.rt, req.Body))
		case "/api/settings/url-test":
			return okResult(setTestURLBridge(s.rt, req.Body))
		case "/api/settings/kernel-autostart":
			return okResult(setKernelAutostartBridge(s.rt, req.Body))
		case "/api/settings/preferences":
			return okResult(setUIPreferencesBridge(s.rt, req.Body))
		case "/api/settings/jwt-secret":
			return okResult(setJWTSecretBridge(s.rt, req.Body))
		case "/api/runtime/clash-mode":
			return okResult(setClashModeBridge(s.rt, req.Body))
		case "/api/desktop/autostart":
			return errResult(setAutostartBridge(s.rt, req.Body))
		case "/api/config/rule-sets/auto-update":
			cfg, err := bridgeBody[model.RuleSetAutoUpdate](req.Body)
			if err != nil {
				return errResult(err)
			}
			if err := s.rt.svc.RuleSets().SetAutoUpdate(cfg); err != nil {
				return errResult(err)
			}
			saved, err := s.rt.svc.RuleSets().AutoUpdate()
			return okResult(saved, err)
		case "/api/config/route/rule-metadata":
			metadata, err := bridgeBody[[]model.RouteRuleMetadata](req.Body)
			if err != nil {
				return errResult(err)
			}
			return okResult(s.rt.svc.Config().UpdateRouteRuleMetadata(metadata))
		}
	}

	// 读操作
	switch path {
	case "/api/service/status":
		return okResult(s.rt.svc.Service().ServiceStatus(ctx))
	case "/api/config/":
		return okResult(s.rt.svc.Config().GetConfig(ctx))
	case "/api/config/raw":
		return okResult(s.rt.svc.Config().GetConfig(ctx))
	case "/api/settings/preferences":
		return okResult(s.rt.svc.Settings().GetUIPreferences(ctx))
	case "/api/settings/password":
		return okResult(s.rt.svc.Settings().GetPasswordStatus(ctx))
	case "/api/settings/kernel-autostart":
		return okResult(s.rt.svc.Settings().GetKernelAutostart(ctx))
	case "/api/settings/url-test":
		return okResult(s.rt.svc.Settings().GetTestURL(ctx))
	case "/api/settings/urltest-defaults":
		return okResult(s.rt.svc.Settings().GetURLTestDefaults(ctx))
	case "/api/settings/jwt-secret":
		return okResult(s.rt.svc.Settings().GetJWTSecret(ctx))
	case "/api/runtime/version":
		return BridgeResponse{Data: s.rt.svc.Kernel().KernelVersion(ctx), Status: "ok"}, nil
	case "/api/runtime/memory":
		return BridgeResponse{Data: s.rt.svc.Kernel().KernelMemory(ctx), Status: "ok"}, nil
	case "/api/runtime/clash-mode":
		return okResult(s.rt.svc.Runtime().GetClashMode(ctx))
	case "/api/runtime/groups":
		return okResult(s.rt.svc.Runtime().OutboundGroups(ctx))
	case "/api/stats/traffic/history":
		return okResult(s.rt.svc.Stats().Traffic(ctx))
	case "/api/stats/connections":
		return okResult(s.rt.svc.Stats().Connections(ctx))
	case "/api/nodes/groups":
		return okResult(s.rt.svc.Runtime().OutboundGroups(ctx))
	case "/api/network/interfaces":
		return okResult(s.rt.svc.Network().ListInterfaces(ctx))
	case "/api/config/rule-sets/status":
		return okResult(s.rt.svc.RuleSets().Status(ctx))
	case "/api/config/rule-sets/auto-update":
		return okResult(s.rt.svc.RuleSets().AutoUpdate())
	case "/api/config/apply-history":
		return okResult(s.rt.svc.Config().ApplyHistory())
	case "/api/config/diagnostics":
		return BridgeResponse{Data: s.rt.svc.Config().Diagnostics(), Status: "ok"}, nil
	case "/api/config/route/rule-metadata":
		return okResult(s.rt.svc.Config().GetRouteRuleMetadata())
	case "/api/nodes/":
		return okResult(listNodesBridge(s.rt))
	case "/api/nodes/test-results":
		return okResult(s.rt.svc.Test().ListResults(ctx))
	case "/api/settings/backup":
		return okResult(s.rt.svc.Backup().CreateBackupArchive(ctx, ""))
	case "/api/subscriptions/":
		return okResult(s.rt.svc.Subscriptions().List(ctx))
	case "/api/desktop/runtime":
		return BridgeResponse{Data: NewNativeCapabilities(s.rt).Runtime(ctx), Status: "ok"}, nil
	case "/api/desktop/autostart":
		return okResult(NewNativeCapabilities(s.rt).IsAutostartEnabled(ctx))
	case "/api/desktop/data-dir":
		return okResult(NewNativeCapabilities(s.rt).DataDir(ctx))
	case "/api/desktop/config-path":
		return okResult(NewNativeCapabilities(s.rt).ConfigPath(ctx))
	case "/readyz":
		return okResult(s.rt.svc.Health().Readiness(ctx))
	case "/healthz", "/health":
		return okResult(s.rt.svc.Health().Liveness(ctx))
	default:
		return s.dispatchPathParam(ctx, path, method, req.Body)
	}
}

// dispatchPathParam 处理带路径参数的路由（apply-history 快照/恢复）。
func (s *BoxdBridgeService) dispatchPathParam(ctx context.Context, path, method string, body json.RawMessage) (BridgeResponse, error) {
	if method == "POST" {
		const prefix = "/api/config/apply-history/"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, "/restore") {
			id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/restore")
			return okResult(s.rt.svc.Config().RestoreConfigSnapshot(ctx, id))
		}
	}
	if method == "GET" {
		const prefix = "/api/config/apply-history/"
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, "/snapshot") {
			id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/snapshot")
			body, err := s.rt.svc.Config().ConfigSnapshot(id)
			if err != nil {
				return errResult(err)
			}
			return BridgeResponse{Data: string(body), Status: "ok"}, nil
		}
	}
	return BridgeResponse{}, ErrorfBridge(404, "not_found", "unknown path %q", path)
}

// syncNodesConfig 同步托管出站配置并重启内核。
func syncNodesConfig(rt *desktopRuntime) error {
	return service.SyncOutboundsAndRestart(
		rt.svc.Deps.NodeManager,
		rt.svc.Deps.SubManager,
		rt.svc.Deps.ConfigPath,
		coreRestarter{rt.svc.Deps.Instance},
	)
}

// coreRestarter 适配 *core.SBInstance 满足 service 的 Restart 能力。
type coreRestarter struct {
	instance *core.SBInstance
}

func (r coreRestarter) Restart() error {
	if r.instance == nil {
		return nil
	}
	return r.instance.Restart()
}

// bridgeBody 解析请求体为指定结构。
func bridgeBody[T any](body json.RawMessage) (T, error) {
	var out T
	if len(body) == 0 {
		return out, ErrorfBridge(400, "invalid_request", "empty request body")
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, ErrorfBridge(400, "invalid_request", "invalid request body: %v", err)
	}
	return out, nil
}

func changePasswordBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	parsed, err := bridgeBody[struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}](body)
	if err != nil {
		return nil, err
	}
	req = parsed
	return rt.svc.Settings().ChangePassword(ctx(), req.CurrentPassword, req.NewPassword)
}

func setTestURLBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[struct {
		URL string `json:"url"`
	}](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Settings().SetTestURL(ctx(), req.URL)
}

func setKernelAutostartBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[struct {
		Enabled bool `json:"enabled"`
	}](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Settings().SetKernelAutostart(ctx(), req.Enabled)
}

func setUIPreferencesBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[model.UIPreferences](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Settings().SetUIPreferences(ctx(), req)
}

func setJWTSecretBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[struct {
		Secret string `json:"secret"`
	}](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Settings().SetJWTSecret(ctx(), req.Secret)
}

func setClashModeBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[struct {
		Mode string `json:"mode"`
	}](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Runtime().SetClashMode(ctx(), req.Mode)
}

func setAutostartBridge(rt *desktopRuntime, body json.RawMessage) error {
	req, err := bridgeBody[struct {
		Enabled bool `json:"enabled"`
	}](body)
	if err != nil {
		return err
	}
	return NewNativeCapabilities(rt).SetAutostart(ctx(), req.Enabled)
}

func okResult(data any, err error) (BridgeResponse, error) {
	if err != nil {
		return errResult(err)
	}
	return BridgeResponse{Data: data, Status: "ok"}, nil
}

func errResult(err error) (BridgeResponse, error) {
	if err == nil {
		return BridgeResponse{Status: "ok"}, nil
	}
	return BridgeResponse{Error: err.Error(), Status: "error"}, err
}
