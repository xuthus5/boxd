package service

import (
	"context"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// ImportService 提供节点导入用例逻辑。
type ImportService struct {
	nodeManager *core.NodeManager
	subManager  *core.SubscriptionManager
	configPath  string
	instance    restartable
}

// NewImportService 构造导入用例服务。
func NewImportService(nodeManager *core.NodeManager, subManager *core.SubscriptionManager, configPath string, instance restartable) *ImportService {
	return &ImportService{nodeManager: nodeManager, subManager: subManager, configPath: configPath, instance: instance}
}

// ParseLink 解析代理链接但不保存。
func (s *ImportService) ParseLink(_ context.Context, link string) (*model.ImportResult, error) {
	if link == "" {
		return nil, Errorf(400, model.ErrorInvalidRequest, "link is required")
	}
	result, err := core.ParseProxyLink(link)
	if err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	return result, nil
}

// SaveNode 保存导入节点并同步配置。
func (s *ImportService) SaveNode(_ context.Context, input NodeInput) error {
	if s.nodeManager == nil {
		return Errorf(500, model.ErrorInternal, "node manager not available")
	}
	outbound := model.Outbound{
		Tag:    input.Tag,
		Type:   input.Type,
		Server: input.Server,
		Port:   input.Port,
		Raw:    input.Config,
	}
	if err := s.nodeManager.Add(outbound); err != nil {
		return Errorf(500, model.ErrorInternal, "failed to save node")
	}
	if err := s.syncConfig(); err != nil {
		return Errorf(500, model.ErrorNodeUpdateFailed, "failed to synchronize node configuration: %v", err)
	}
	return nil
}

// NodeInput 描述节点保存请求。
type NodeInput struct {
	Tag    string `json:"tag"`
	Type   string `json:"type"`
	Server string `json:"server"`
	Port   int    `json:"port"`
	Config any    `json:"config"`
}

func (s *ImportService) syncConfig() error {
	if s.nodeManager == nil || s.subManager == nil {
		return nil
	}
	return SyncOutboundsAndRestart(s.nodeManager, s.subManager, s.configPath, s.instance)
}
