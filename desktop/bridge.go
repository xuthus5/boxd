package main

import (
	"context"
	"encoding/json"
	"strings"
)

// BridgeRequest 描述一次传输无关的用例调用。
type BridgeRequest struct {
	Path string          `json:"path"`
	Body json.RawMessage `json:"body,omitempty"`
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
	switch path {
	case "/api/service/status":
		return okResult(s.rt.svc.Service().ServiceStatus(ctx))
	case "/api/service/start":
		return errResult(s.rt.svc.Service().ServiceStart(ctx))
	case "/api/service/stop":
		return errResult(s.rt.svc.Service().ServiceStop(ctx))
	case "/api/service/restart":
		return errResult(s.rt.svc.Service().ServiceRestart(ctx))
	case "/api/config/", "/api/config/raw":
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
	case "/api/runtime/version":
		return BridgeResponse{Data: s.rt.svc.Kernel().KernelVersion(ctx), Status: "ok"}, nil
	case "/api/runtime/memory":
		return BridgeResponse{Data: s.rt.svc.Kernel().KernelMemory(ctx), Status: "ok"}, nil
	case "/api/stats/traffic/history":
		return okResult(s.rt.svc.Stats().Traffic(ctx))
	case "/api/stats/connections":
		return okResult(s.rt.svc.Stats().Connections(ctx))
	case "/readyz":
		return okResult(s.rt.svc.Health().Readiness(ctx))
	case "/healthz", "/health":
		return okResult(s.rt.svc.Health().Liveness(ctx))
	default:
		return BridgeResponse{}, ErrorfBridge(404, "not_found", "unknown path %q", path)
	}
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
