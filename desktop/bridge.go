package main

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

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

	// 剥离查询串：带参数路由（test-history、delay、stats 过滤）从 query 取值。
	path, rawQuery, _ := strings.Cut(path, "?")
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return errResult(ErrorfBridge(400, "invalid_request", "invalid query string"))
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
		case "/api/subscriptions/":
			input, err := bridgeBody[service.SubscriptionInput](req.Body)
			if err != nil {
				return errResult(err)
			}
			return okResult(s.rt.svc.Subscriptions().Create(ctx, input))
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
	case "/api/stats/traffic":
		up, down := int64(0), int64(0)
		if s.rt.instance != nil && s.rt.instance.TrafficTracker() != nil {
			up, down = s.rt.instance.TrafficTracker().Total()
		}
		return okResult(map[string]any{
			"upload_bytes":   up,
			"download_bytes": down,
			"timestamp":      time.Now().Format(time.RFC3339),
		}, nil)
	case "/api/stats/logs":
		return okResult(map[string]any{"entries": s.rt.svc.Deps.KernelLogWriter.Recent()}, nil)
	case "/api/stats/app-logs":
		return okResult(map[string]any{"entries": s.rt.svc.Deps.AppLogWriter.Recent()}, nil)
	case "/api/stats/connections":
		if method == "DELETE" {
			resp, err, handled := s.dispatchStatsClosePath(ctx, path, params)
			if handled {
				return resp, err
			}
		}
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
	case "/api/nodes/test-history":
		return okResult(s.rt.svc.Test().ListHistory(ctx, params.Get("tag")))
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
		return s.dispatchPathParam(ctx, path, params, method, req.Body)
	}
}

// dispatchPathParam 处理带路径参数或查询参数的路由。
func (s *BoxdBridgeService) dispatchPathParam(ctx context.Context, path string, params url.Values, method string, body json.RawMessage) (BridgeResponse, error) {
	if resp, err, handled := s.dispatchApplyHistoryPath(ctx, path, method); handled {
		return resp, err
	}
	if resp, err, handled := s.dispatchSubscriptionPath(ctx, path, method, body); handled {
		return resp, err
	}
	if resp, err, handled := s.dispatchNodePath(ctx, path, params, method, body); handled {
		return resp, err
	}
	if resp, err, handled := s.dispatchStatsClosePath(ctx, path, params); handled {
		return resp, err
	}
	return BridgeResponse{}, ErrorfBridge(404, "not_found", "unknown path %q", path)
}

// dispatchApplyHistoryPath 处理 apply-history 快照/恢复路由。
func (s *BoxdBridgeService) dispatchApplyHistoryPath(ctx context.Context, path, method string) (BridgeResponse, error, bool) {
	const prefix = "/api/config/apply-history/"
	if !strings.HasPrefix(path, prefix) {
		return BridgeResponse{}, nil, false
	}
	id := strings.TrimPrefix(path, prefix)
	switch {
	case method == "POST" && strings.HasSuffix(id, "/restore"):
		resp, err := okResult(s.rt.svc.Config().RestoreConfigSnapshot(ctx, strings.TrimSuffix(id, "/restore")))
		return resp, err, true
	case method == "GET" && strings.HasSuffix(id, "/snapshot"):
		body, err := s.rt.svc.Config().ConfigSnapshot(strings.TrimSuffix(id, "/snapshot"))
		if err != nil {
			resp, rerr := errResult(err)
			return resp, rerr, true
		}
		return BridgeResponse{Data: string(body), Status: "ok"}, nil, true
	}
	return BridgeResponse{}, nil, false
}

// dispatchSubscriptionPath 处理订阅参数路由：GET/PUT/DELETE /api/subscriptions/{id}、POST /{id}/refresh。
func (s *BoxdBridgeService) dispatchSubscriptionPath(ctx context.Context, path, method string, body json.RawMessage) (BridgeResponse, error, bool) {
	const prefix = "/api/subscriptions/"
	if !strings.HasPrefix(path, prefix) {
		return BridgeResponse{}, nil, false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/")
	switch method {
	case "GET":
		resp, err := okResult(s.rt.svc.Subscriptions().Get(ctx, rest))
		return resp, err, true
	case "PUT":
		input, err := bridgeBody[service.SubscriptionInput](body)
		if err != nil {
			resp, rerr := errResult(err)
			return resp, rerr, true
		}
		resp, rerr := errResult(s.rt.svc.Subscriptions().Update(ctx, rest, input))
		return resp, rerr, true
	case "DELETE":
		resp, err := errResult(s.rt.svc.Subscriptions().Delete(ctx, rest))
		return resp, err, true
	case "POST":
		if strings.HasSuffix(rest, "/refresh") {
			resp, err := errResult(s.rt.svc.Subscriptions().Refresh(ctx, strings.TrimSuffix(rest, "/refresh")))
			return resp, err, true
		}
	}
	return BridgeResponse{}, nil, false
}

// dispatchNodePath 处理节点参数路由：GET/PUT/DELETE /api/nodes/{tag}、GET /{tag}/delay。
func (s *BoxdBridgeService) dispatchNodePath(ctx context.Context, path string, params url.Values, method string, body json.RawMessage) (BridgeResponse, error, bool) {
	const prefix = "/api/nodes/"
	if !strings.HasPrefix(path, prefix) {
		return BridgeResponse{}, nil, false
	}
	rest := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/")
	if method == "POST" {
		return s.dispatchNodeActionPath(ctx, rest, body)
	}
	if tag, ok := strings.CutSuffix(rest, "/delay"); ok && method == "GET" {
		resp, err := okResult(s.rt.svc.Runtime().OutboundDelay(ctx, unescapePathSegment(tag), params.Get("url"), delayTimeout(params)))
		return resp, err, true
	}
	if strings.Contains(rest, "/") {
		return BridgeResponse{}, nil, false
	}
	tag := unescapePathSegment(rest)
	switch method {
	case "GET":
		node := s.rt.svc.Deps.NodeManager.Get(tag)
		if node == nil {
			resp, err := errResult(ErrorfBridge(404, "not_found", "node %q not found", tag))
			return resp, err, true
		}
		resp, err := okResult(node, nil)
		return resp, err, true
	case "PUT":
		input, err := bridgeBody[service.NodeInput](body)
		if err != nil {
			resp, rerr := errResult(err)
			return resp, rerr, true
		}
		updated := model.Outbound{
			Tag:    input.Tag,
			Type:   input.Type,
			Server: input.Server,
			Port:   input.Port,
			Raw:    input.Config,
		}
		resp, rerr := errResult(s.rt.svc.Deps.NodeManager.Update(tag, updated))
		return resp, rerr, true
	case "DELETE":
		resp, err := errResult(s.rt.svc.Deps.NodeManager.Delete(tag))
		return resp, err, true
	}
	return BridgeResponse{}, nil, false
}

// dispatchNodeActionPath 处理节点动作路径（POST）：/selectors/{group}/select、/groups/{group}/urltest。
func (s *BoxdBridgeService) dispatchNodeActionPath(ctx context.Context, rest string, body json.RawMessage) (BridgeResponse, error, bool) {
	if group, ok := strings.CutSuffix(rest, "/select"); ok && strings.HasPrefix(group, "selectors/") {
		group = unescapePathSegment(strings.TrimPrefix(group, "selectors/"))
		input, err := bridgeBody[struct {
			Tag string `json:"tag"`
		}](body)
		if err != nil {
			resp, rerr := errResult(err)
			return resp, rerr, true
		}
		selected, err := s.rt.svc.Runtime().SelectOutbound(ctx, group, input.Tag)
		if err != nil {
			resp, rerr := errResult(err)
			return resp, rerr, true
		}
		resp, rerr := okResult(map[string]string{"selected": selected}, nil)
		return resp, rerr, true
	}
	if group, ok := strings.CutSuffix(rest, "/urltest"); ok && strings.HasPrefix(group, "groups/") {
		group = unescapePathSegment(strings.TrimPrefix(group, "groups/"))
		resp, err := okResult(s.rt.svc.Runtime().URLTestDelays(ctx, group))
		return resp, err, true
	}
	return BridgeResponse{}, nil, false
}

// dispatchStatsClosePath 处理统计的关闭连接路由：DELETE /api/stats/connections、DELETE /{id}。
func (s *BoxdBridgeService) dispatchStatsClosePath(ctx context.Context, path string, params url.Values) (BridgeResponse, error, bool) {
	const prefix = "/api/stats/connections"
	if !strings.HasPrefix(path, prefix) {
		return BridgeResponse{}, nil, false
	}
	if strings.TrimPrefix(path, prefix) == "" {
		ids, err := service.ParseConnectionIDs(params.Get("ids"))
		if err != nil {
			resp, rerr := errResult(ErrorfBridge(400, "invalid_request", "invalid ids"))
			return resp, rerr, true
		}
		filters := service.ConnectionCloseFilters{
			Outbound: params.Get("outbound"),
			Rule:     params.Get("rule"),
			Process:  params.Get("process"),
			IDs:      ids,
		}
		resp, rerr := okResult(s.rt.svc.Stats().CloseConnections(ctx, filters))
		return resp, rerr, true
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(path, prefix+"/"), 10, 64)
	if err != nil || id <= 0 {
		return BridgeResponse{}, nil, false
	}
	resp, rerr := okResult(s.rt.svc.Stats().CloseConnection(ctx, id))
	return resp, rerr, true
}

// delayTimeout 解析 /api/nodes/{tag}/delay 的 timeout 参数，缺失或非法时使用默认值。
func delayTimeout(params url.Values) int64 {
	raw := params.Get("timeout")
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 || parsed > 60000 {
		return 0
	}
	return parsed
}

// unescapePathSegment 解码 URL 路径段，失败时原样返回。
func unescapePathSegment(value string) string {
	if decoded, err := url.PathUnescape(value); err == nil {
		return decoded
	}
	return value
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
