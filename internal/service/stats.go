package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// statsInstance 抽象内核流量/连接能力。
type statsInstance interface {
	TrafficTracker() *core.TrafficTracker
	CloseConnection(id int64) bool
	CloseAllConnections() int
	CloseConnectionsByOutbound(outbound string) int
	CloseConnectionsByRule(rule string) int
	CloseConnectionsByProcess(process string) int
	CloseConnectionsByIDs(ids []int64) int
}

// TrafficHistoryPoint 流量历史采样点。
type TrafficHistoryPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	UploadBytes   int64     `json:"upload_bytes"`
	DownloadBytes int64     `json:"download_bytes"`
}

// StatsService 提供流量与连接用例逻辑。
type StatsService struct {
	instance statsInstance
}

// NewStatsService 构造流量/连接用例服务。
func NewStatsService(instance statsInstance) *StatsService {
	return &StatsService{instance: instance}
}

// Traffic 返回当前累计上下行流量。
func (s *StatsService) Traffic(_ context.Context) (map[string]any, error) {
	if s.instance == nil || s.instance.TrafficTracker() == nil {
		return map[string]any{"upload_bytes": int64(0), "download_bytes": int64(0)}, nil
	}
	up, down := s.instance.TrafficTracker().Total()
	return map[string]any{
		"upload_bytes":   up,
		"download_bytes": down,
		"timestamp":      time.Now().Format(time.RFC3339),
	}, nil
}

// Connections 返回当前连接快照。
func (s *StatsService) Connections(_ context.Context) (map[string]any, error) {
	if s.instance == nil || s.instance.TrafficTracker() == nil {
		return map[string]any{"active_connections": 0, "list": []core.TrafficConn{}}, nil
	}
	list := s.instance.TrafficTracker().Connections()
	return map[string]any{
		"active_connections": len(list),
		"list":               list,
	}, nil
}

// CloseConnections 关闭连接。filters 中 outbound/rule/process/ids 互斥。
func (s *StatsService) CloseConnections(_ context.Context, filters ConnectionCloseFilters) (map[string]any, error) {
	if s.instance == nil {
		return nil, Errorf(503, model.ErrorUnavailable, "service not available")
	}
	count := 0
	filterCount := 0
	if filters.Outbound != "" {
		filterCount++
	}
	if filters.Rule != "" {
		filterCount++
	}
	if filters.Process != "" {
		filterCount++
	}
	if len(filters.IDs) > 0 {
		filterCount++
	}
	if filterCount > 1 {
		return nil, Errorf(400, model.ErrorInvalidRequest, "specify only one of outbound, rule, process, or ids")
	}
	switch {
	case filters.Outbound != "":
		count = s.instance.CloseConnectionsByOutbound(filters.Outbound)
	case filters.Rule != "":
		count = s.instance.CloseConnectionsByRule(filters.Rule)
	case filters.Process != "":
		count = s.instance.CloseConnectionsByProcess(filters.Process)
	case len(filters.IDs) > 0:
		ids := make([]int64, 0, len(filters.IDs))
		for _, raw := range filters.IDs {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		count = s.instance.CloseConnectionsByIDs(ids)
	default:
		count = s.instance.CloseAllConnections()
	}
	return map[string]any{
		"closed":   count,
		"outbound": filters.Outbound,
		"rule":     filters.Rule,
		"process":  filters.Process,
		"ids":      strings.Join(filters.IDs, ","),
	}, nil
}

// CloseConnection 关闭单个连接。
func (s *StatsService) CloseConnection(_ context.Context, id int64) (map[string]int64, error) {
	if s.instance == nil {
		return nil, Errorf(503, model.ErrorUnavailable, "service not available")
	}
	if !s.instance.CloseConnection(id) {
		return nil, Errorf(404, model.ErrorNotFound, "connection not found")
	}
	return map[string]int64{"closed_id": id}, nil
}

// ConnectionCloseFilters 描述批量关闭连接过滤条件。
type ConnectionCloseFilters struct {
	Outbound string
	Rule     string
	Process  string
	IDs      []string
}

// ParseConnectionIDs 解析逗号分隔的连接 ID 列表。
func ParseConnectionIDs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, err := strconv.ParseInt(part, 10, 64); err != nil || part[0] == '-' {
			return nil, Errorf(400, model.ErrorInvalidRequest, "invalid connection id %q", part)
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		ids = append(ids, part)
		if len(ids) > maxCloseConnectionIDs {
			return nil, Errorf(400, model.ErrorInvalidRequest, "too many connection ids (max %d)", maxCloseConnectionIDs)
		}
	}
	return ids, nil
}

// maxCloseConnectionIDs 限制批量关闭连接数。
const maxCloseConnectionIDs = 500
