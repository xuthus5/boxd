package model

import "time"

// RuleSetAutoUpdate 控制本地内置规则集的定时更新。
// 默认关闭；开启后仅更新 BoxUI 管理的 local 内置源。
type RuleSetAutoUpdate struct {
	Enabled  bool   `json:"enabled"`
	Interval string `json:"interval"`
}

// RuleSetStatusItem 描述配置中单个规则集的可更新状态。
type RuleSetStatusItem struct {
	Tag            string     `json:"tag"`
	Type           string     `json:"type"`
	Format         string     `json:"format,omitempty"`
	Path           string     `json:"path,omitempty"`
	URL            string     `json:"url,omitempty"`
	UpdateInterval string     `json:"update_interval,omitempty"`
	DownloadDetour string     `json:"download_detour,omitempty"`
	Builtin        bool       `json:"builtin"`
	Updatable      bool       `json:"updatable"`
	LastUpdated    *time.Time `json:"last_updated,omitempty"`
	LastEtag       string     `json:"last_etag,omitempty"`
	FileSize       int64      `json:"file_size,omitempty"`
	Note           string     `json:"note,omitempty"`
}

// RuleSetUpdateResult 单条规则集更新结果。
type RuleSetUpdateResult struct {
	Tag         string     `json:"tag"`
	Type        string     `json:"type"`
	OK          bool       `json:"ok"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	NotModified bool       `json:"not_modified,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// RuleSetUpdateResponse 批量更新汇总。
type RuleSetUpdateResponse struct {
	Results      []RuleSetUpdateResult `json:"results"`
	UpdatedCount int                   `json:"updated_count"`
	FailedCount  int                   `json:"failed_count"`
	SkippedCount int                   `json:"skipped_count"`
	Restarted    bool                  `json:"restarted"`
}
