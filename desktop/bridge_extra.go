package main

import (
	"context"
	"encoding/json"

	"github.com/xuthus5/boxd/internal/service"
)

// loginBridge 解析登录请求并调用认证服务。
func loginBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return rt.svc.Auth().Login(context.Background(), "", service.AuthCredentials{
		Username: req.Username,
		Password: req.Password,
	})
}

// importLinkBridge 解析并导入代理链接。
func importLinkBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	var req struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return rt.svc.Import().ParseLink(context.Background(), req.Link)
}

// importSaveBridge 保存导入的节点。
func importSaveBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	input, err := bridgeBody[service.NodeInput](body)
	if err != nil {
		return nil, err
	}
	return nil, rt.svc.Import().SaveNode(context.Background(), input)
}

// testRunBridge 执行单点测速。
func testRunBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[service.TestRequest](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Test().Run(context.Background(), req)
}

// testBatchBridge 执行批量测速。
func testBatchBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[service.TestBatchRequest](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.Test().RunBatch(context.Background(), req)
}

// probeDNSBridge 执行单点 DNS 探测。
func probeDNSBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	req, err := bridgeBody[service.DNSProbeRequest](body)
	if err != nil {
		return nil, err
	}
	return rt.svc.DNSProbe().Probe(context.Background(), req)
}

// probeDNSBatchBridge 执行批量 DNS 探测。
func probeDNSBatchBridge(rt *desktopRuntime, body json.RawMessage) (any, error) {
	var req struct {
		Items       []service.DNSProbeRequest `json:"items"`
		Concurrency int                       `json:"concurrency"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return rt.svc.DNSProbe().ProbeBatch(context.Background(), req.Items, req.Concurrency)
}

// listNodesBridge 返回节点列表。
func listNodesBridge(rt *desktopRuntime) (any, error) {
	if rt == nil || rt.svc == nil || rt.svc.Deps.NodeManager == nil {
		return nil, errNotReady()
	}
	return rt.svc.Deps.NodeManager.List(), nil
}
