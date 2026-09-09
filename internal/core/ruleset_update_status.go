package core

import (
	"os"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func (u *RuleSetUpdater) statusOf(entry map[string]any, cache *bbolt.DB) model.RuleSetStatusItem {
	tag, _ := entry["tag"].(string)
	typ, _ := entry["type"].(string)
	if typ == "" {
		typ = "inline"
	}
	item := model.RuleSetStatusItem{
		Tag:            tag,
		Type:           typ,
		Format:         stringValue(entry["format"]),
		Path:           stringValue(entry["path"]),
		URL:            stringValue(entry["url"]),
		UpdateInterval: stringValue(entry["update_interval"]),
		DownloadDetour: stringValue(entry["download_detour"]),
	}
	policy := ruleSetEntryHTTPPolicy(entry)
	item.HTTPClient = policy.tag
	item.HTTPClientSource = policy.source
	item.KernelManaged = policy.managed
	switch typ {
	case "local":
		u.localRuleSetStatus(&item)
	case "remote":
		u.remoteRuleSetStatus(&item, cache)
	default:
		item.Updatable = false
		item.Note = "inline rule-set has no remote update"
	}
	return item
}

func (u *RuleSetUpdater) localRuleSetStatus(item *model.RuleSetStatusItem) {
	item.Builtin = u.installer.IsBuiltinLocal(item.Tag)
	item.Updatable = item.Builtin
	if !item.Updatable {
		item.Note = "custom local rule-set is managed by file path"
	}
	if item.Path != "" {
		if info, err := os.Stat(item.Path); err == nil {
			updated := info.ModTime()
			item.LastUpdated = &updated
			item.FileSize = info.Size()
		}
	}
}

func (u *RuleSetUpdater) remoteRuleSetStatus(item *model.RuleSetStatusItem, cache *bbolt.DB) {
	item.Updatable = item.URL != ""
	if item.URL != "" {
		if err := ValidatePublicHTTPURL(item.URL); err != nil {
			item.Updatable = false
			item.Note = "remote rule-set URL is invalid or blocked"
		}
	}
	if item.UpdateInterval == "" {
		item.UpdateInterval = DefaultRemoteRuleSetInterval()
	}
	if cache != nil {
		if saved := loadRuleSetCacheForURL(cache, item.Tag, item.URL); saved != nil {
			updated := saved.LastUpdated
			item.LastUpdated = &updated
			item.LastEtag = saved.LastEtag
			item.FileSize = int64(len(saved.Content))
		}
	}
	if item.KernelManaged {
		item.Updatable = false
		item.NoteCode = RuleSetErrorKernelManaged
		item.Note = "remote rule-set updates are managed by the kernel HTTP client"
	}
}
