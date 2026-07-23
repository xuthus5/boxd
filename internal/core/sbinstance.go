package core

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	singboxconstant "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

var Version = "dev"

// 运行时控制相关哨兵错误，便于 API 层用 errors.Is 精确映射 HTTP 状态。
var (
	ErrNotRunning    = errors.New("sing-box not running")
	ErrGroupNotFound = errors.New("outbound group not found")
	ErrNotSelectable = errors.New("outbound is not a selectable group")
	ErrTagNotInGroup = errors.New("outbound not in group")
)

// SBInstance 封装 sing-box 内核实例，提供运行控制与运行时能力访问。
type SBInstance struct {
	mu          sync.Mutex
	running     bool
	configPath  string
	box         boxInstance
	boxCtx      context.Context // box 启动使用的 context，含注册的 CacheFile 等服务
	cancel      context.CancelFunc
	startTime   time.Time
	lastError   string
	lastErrorAt time.Time
	LogWriter   *LogWriter
	Traffic     *TrafficTracker
}

// OutboundGroupInfo 描述一个出站分组（selector/urltest）及其当前选中节点。
type OutboundGroupInfo struct {
	Type string   `json:"type"`
	Tag  string   `json:"tag"`
	Now  string   `json:"now"`
	All  []string `json:"all"`
}

type boxInstance interface {
	Start() error
	Close() error
	Router() boxRouter
	Outbound() boxOutboundManager
}

type boxRouter interface {
	AppendTracker(tracker adapter.ConnectionTracker)
}

// boxOutboundManager 暴露 box 的出站管理器接口，boxd 通过它查询分组与切换。
type boxOutboundManager interface {
	Outbound(tag string) (adapter.Outbound, bool)
	Outbounds() []adapter.Outbound
}

// groupOutbound 覆盖 selector/urltest 等出站分组（adapter.OutboundGroup）。
// 避免直接依赖 protocol/group 包。
type groupOutbound interface {
	adapter.Outbound
	Now() string
	All() []string
}

// selectableOutbound 额外支持手动切换成员（selector）。
type selectableOutbound interface {
	groupOutbound
	SelectOutbound(string) bool
}

var newBox = func(options box.Options) (boxInstance, error) {
	instance, err := box.New(options)
	if err != nil {
		return nil, err
	}
	return realBox{box: instance}, nil
}

type realBox struct {
	box *box.Box
}

func (b realBox) Start() error {
	return b.box.Start()
}

func (b realBox) Close() error {
	return b.box.Close()
}

func (b realBox) Router() boxRouter {
	return b.box.Router()
}

func (b realBox) Outbound() boxOutboundManager {
	return b.box.Outbound()
}

func NewSBInstance(configPath string, logWriter *LogWriter) *SBInstance {
	return &SBInstance{
		configPath: configPath,
		LogWriter:  logWriter,
	}
}

func (s *SBInstance) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	configData, err := os.ReadFile(s.configPath)
	if err != nil {
		s.recordStartErrorLocked(err)
		return err
	}

	ctx, cancel := context.WithCancel(include.Context(context.Background()))

	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, configData); err != nil {
		cancel()
		s.recordStartErrorLocked(err)
		return err
	}

	instance, err := newBox(box.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: s.LogWriter,
	})
	if err != nil {
		cancel()
		s.recordStartErrorLocked(err)
		return err
	}

	// 在启动前注册流量追踪器
	s.Traffic = NewTrafficTracker()
	instance.Router().AppendTracker(s.Traffic)

	if err := instance.Start(); err != nil {
		_ = instance.Close()
		cancel()
		s.recordStartErrorLocked(err)
		return err
	}

	s.box = instance
	s.boxCtx = ctx
	s.cancel = cancel
	s.running = true
	s.startTime = time.Now()
	s.lastError = ""
	s.lastErrorAt = time.Time{}
	return nil
}

func (s *SBInstance) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.box == nil {
		return nil
	}

	closeErr := s.box.Close()
	s.box = nil
	s.boxCtx = nil
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
	return closeErr
}

func (s *SBInstance) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// TrafficTracker 返回当前流量追踪器，便于 API 层在不持有具体类型时访问。
func (s *SBInstance) TrafficTracker() *TrafficTracker {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Traffic
}

func (s *SBInstance) Status() model.ServiceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := model.ServiceStatus{
		Running:    s.running,
		Version:    singboxconstant.Version,
		ConfigPath: s.configPath,
	}

	if s.running && !s.startTime.IsZero() {
		st.Uptime = time.Since(s.startTime).Round(time.Second).String()
		startedAt := s.startTime.UTC()
		st.StartedAt = &startedAt
	}
	if s.lastError != "" {
		st.LastError = s.lastError
		if !s.lastErrorAt.IsZero() {
			at := s.lastErrorAt.UTC()
			st.LastErrorAt = &at
		}
	}

	return st
}

func (s *SBInstance) recordStartErrorLocked(err error) {
	if err == nil {
		return
	}
	s.lastError = err.Error()
	s.lastErrorAt = time.Now().UTC()
}
