package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/xuthus5/boxd/internal/core"
)

func (s *BoxdBridgeService) dispatchSetup(ctx context.Context, request BridgeRequest) (BridgeResponse, error, bool) {
	if request.Path == "/api/config/setup" && request.Method == "GET" {
		response, err := okResult(s.rt.svc.Config().GetSetup(ctx))
		return response, err, true
	}
	if request.Path != "/api/config/setup/preview" && request.Path != "/api/config/setup/apply" {
		return BridgeResponse{}, nil, false
	}
	if request.Method != "POST" {
		response, err := errResult(ErrorfBridge(405, "invalid_request", "module setup requires POST"))
		return response, err, true
	}
	input, err := decodeSetupBridgeRequest(request.Body)
	if err != nil {
		response, err := errResult(err)
		return response, err, true
	}
	if request.Path == "/api/config/setup/preview" {
		response, err := okResult(s.rt.svc.Config().PreviewSetup(ctx, input))
		return response, err, true
	}
	response, err := okResult(s.rt.svc.Config().ApplySetup(ctx, input))
	return response, err, true
}

func decodeSetupBridgeRequest(body json.RawMessage) (core.SetupRequest, error) {
	var request core.SetupRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, ErrorfBridge(400, "invalid_request", "invalid module setup request")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return request, ErrorfBridge(400, "invalid_request", "invalid module setup request")
	}
	return request, nil
}
