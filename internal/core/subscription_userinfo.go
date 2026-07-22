package core

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

// parseSubscriptionUserinfo 解析常见订阅响应头 subscription-userinfo。
// 格式示例：upload=123; download=456; total=1073741824; expire=1719859200
func parseSubscriptionUserinfo(header http.Header) *model.SubscriptionTraffic {
	if header == nil {
		return nil
	}
	raw := header.Get("Subscription-Userinfo")
	if raw == "" {
		raw = header.Get("subscription-userinfo")
	}
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	values := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	if len(values) == 0 {
		return nil
	}

	traffic := &model.SubscriptionTraffic{}
	hasAny := false
	if upload, ok := parseInt64Field(values["upload"]); ok {
		traffic.Upload = upload
		hasAny = true
	}
	if download, ok := parseInt64Field(values["download"]); ok {
		traffic.Download = download
		hasAny = true
	}
	if total, ok := parseInt64Field(values["total"]); ok {
		traffic.Total = total
		hasAny = true
	}
	if expireRaw, ok := values["expire"]; ok && expireRaw != "" {
		if expireUnix, err := strconv.ParseInt(expireRaw, 10, 64); err == nil && expireUnix > 0 {
			expire := time.Unix(expireUnix, 0).UTC()
			traffic.Expire = &expire
			hasAny = true
		}
	}
	if !hasAny {
		return nil
	}
	return traffic
}

func parseInt64Field(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}
