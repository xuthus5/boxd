package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

// RuntimeHandler 暴露运行时交互能力：selector 切换、分组查询。
type RuntimeHandler struct {
	instance runtimeInstance
}

// runtimeInstance 抽取 SBInstance 中运行时控制所需子集，便于测试注入。
type runtimeInstance interface {
	OutboundGroups() []core.OutboundGroupInfo
	SelectOutbound(groupTag, outTag string) error
	URLTestDelays(ctx context.Context, groupTag string) (map[string]uint16, error)
	FlushDNS() error
	FlushFakeIP() error
	OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error)
}

func NewRuntimeHandler(instance runtimeInstance) *RuntimeHandler {
	return &RuntimeHandler{instance: instance}
}

// OutboundGroups GET /api/nodes/groups
func (h *RuntimeHandler) OutboundGroups(w http.ResponseWriter, r *http.Request) {
	groups := h.instance.OutboundGroups()
	if groups == nil {
		groups = []core.OutboundGroupInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}

// SelectOutbound PUT /api/nodes/selectors/{group}/select
func (h *RuntimeHandler) SelectOutbound(w http.ResponseWriter, r *http.Request) {
	group := chi.URLParam(r, "group")
	if group == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "group is required")
		return
	}

	var req struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}
	if req.Tag == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag is required")
		return
	}

	if err := h.instance.SelectOutbound(group, req.Tag); err != nil {
		switch {
		case errors.Is(err, core.ErrNotRunning):
			writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, err.Error())
		case errors.Is(err, core.ErrGroupNotFound):
			writeJSONErrorCode(w, http.StatusNotFound, model.ErrorRuntimeGroupNotFound, err.Error())
		case errors.Is(err, core.ErrNotSelectable):
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorRuntimeNotSelectable, err.Error())
		case errors.Is(err, core.ErrTagNotInGroup):
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorRuntimeNotSelectable, err.Error())
		default:
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		}
		return
	}

	writeJSONWithMeta(w, http.StatusOK, map[string]string{"selected": req.Tag}, map[string]string{"group": group})
}

// URLTestDelays POST /api/nodes/groups/{group}/urltest
func (h *RuntimeHandler) URLTestDelays(w http.ResponseWriter, r *http.Request) {
	group := chi.URLParam(r, "group")
	if group == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "group is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	delays, err := h.instance.URLTestDelays(ctx, group)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotRunning):
			writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, err.Error())
		case errors.Is(err, core.ErrGroupNotFound):
			writeJSONErrorCode(w, http.StatusNotFound, model.ErrorRuntimeGroupNotFound, err.Error())
		case errors.Is(err, core.ErrNotSelectable):
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorRuntimeNotSelectable, err.Error())
		default:
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		}
		return
	}
	writeJSONWithMeta(w, http.StatusOK, delays, map[string]string{"group": group})
}

// FlushDNS POST /api/runtime/dns/flush —— 清空内核 DNS 缓存。
func (h *RuntimeHandler) FlushDNS(w http.ResponseWriter, r *http.Request) {
	if err := h.instance.FlushDNS(); err != nil {
		if errors.Is(err, core.ErrNotRunning) {
			writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, err.Error())
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// FlushFakeIP POST /api/runtime/fakeip/flush —— 清空 FakeIP 存储。
func (h *RuntimeHandler) FlushFakeIP(w http.ResponseWriter, r *http.Request) {
	if err := h.instance.FlushFakeIP(); err != nil {
		switch {
		case errors.Is(err, core.ErrNotRunning):
			writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, err.Error())
		case errors.Is(err, core.ErrFeatureNotEnabled):
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		default:
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// OutboundDelay GET /api/nodes/{tag}/delay —— 单出站延迟测试。
// 查询参数：url（可选，空时用 urltest 默认）、timeout（毫秒，可选，默认 10000）。
func (h *RuntimeHandler) OutboundDelay(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag is required")
		return
	}

	link := r.URL.Query().Get("url")
	timeoutMs := int64(10000)
	if raw := r.URL.Query().Get("timeout"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid timeout")
			return
		}
		timeoutMs = parsed
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	delay, err := h.instance.OutboundDelay(ctx, tag, link, time.Duration(timeoutMs)*time.Millisecond)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotRunning):
			writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, err.Error())
		case errors.Is(err, core.ErrOutboundNotFound):
			writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, err.Error())
		default:
			writeJSONErrorCode(w, http.StatusBadGateway, model.ErrorRuntimeDelayFailed, fmt.Sprintf("delay test failed: %v", err))
		}
		return
	}
	if delay == 0 {
		writeJSONErrorCode(w, http.StatusBadGateway, model.ErrorRuntimeDelayFailed, "delay test failed: no response")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tag": tag, "delay": delay})
}
