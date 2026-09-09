package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

type remoteRuleSetDownload struct {
	response   *http.Response
	url        string
	cachedETag string
}

func (u *RuleSetUpdater) updateRemote(ctx context.Context, entry map[string]any, result model.RuleSetUpdateResult) model.RuleSetUpdateResult {
	download, err := u.fetchRemoteRuleSet(ctx, entry)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if download.response == nil || download.response.Body == nil {
		return failRuleSetResult(result, "rule-set response body is nil", nil)
	}
	defer func() { _ = download.response.Body.Close() }()
	return u.finishRemoteRuleSet(ctx, download, result)
}

func (u *RuleSetUpdater) fetchRemoteRuleSet(ctx context.Context, entry map[string]any) (remoteRuleSetDownload, error) {
	url := stringValue(entry["url"])
	if strings.TrimSpace(url) == "" {
		return remoteRuleSetDownload{}, errors.New("remote rule-set url is empty")
	}
	if err := ValidatePublicHTTPURL(url); err != nil {
		return remoteRuleSetDownload{}, err
	}
	etag := u.remoteRuleSetETag(stringValue(entry["tag"]), url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return remoteRuleSetDownload{}, err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	client := u.client
	if client == nil {
		client = newPublicHTTPClient(ruleSetHTTPTimeout)
	}
	resp, err := client.Do(req)
	return remoteRuleSetDownload{response: resp, url: url, cachedETag: etag}, err
}

func (u *RuleSetUpdater) remoteRuleSetETag(tag, url string) string {
	cache, err := u.openCacheReadOnly()
	if err != nil || cache == nil {
		return ""
	}
	defer func() { _ = cache.Close() }()
	if saved := loadRuleSetCacheForURL(cache, tag, url); saved != nil {
		return saved.LastEtag
	}
	return ""
}

func (u *RuleSetUpdater) finishRemoteRuleSet(ctx context.Context, download remoteRuleSetDownload, result model.RuleSetUpdateResult) model.RuleSetUpdateResult {
	resp := download.response
	now := time.Now()
	if err := ctx.Err(); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	switch resp.StatusCode {
	case http.StatusNotModified:
		if download.cachedETag == "" {
			return failRuleSetResult(result, "unexpected status 304 without a matching cached rule-set", nil)
		}
		if err := u.touchRemoteCache(result.Tag, now); err != nil && !errors.Is(err, ErrRuleSetCacheDisabled) {
			return failRuleSetResult(result, err.Error(), err)
		}
		result.OK, result.NotModified, result.UpdatedAt = true, true, &now
		return result
	case http.StatusOK:
	default:
		return failRuleSetResult(result, fmt.Sprintf("unexpected status %d", resp.StatusCode), nil)
	}
	content, err := readRemoteRuleSetResponse(resp)
	if err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	if err := ctx.Err(); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	saved := &savedRuleSetBinary{Content: content, LastEtag: resp.Header.Get("Etag"), LastUpdated: now, URLHash: ruleSetURLHash(download.url)}
	if err := u.saveRemoteCache(result.Tag, saved); err != nil {
		return failRuleSetResult(result, err.Error(), err)
	}
	result.OK, result.UpdatedAt = true, &now
	return result
}

func readRemoteRuleSetResponse(resp *http.Response) ([]byte, error) {
	if resp.ContentLength > maxRuleSetBodyBytes {
		return nil, ErrRuleSetContentTooLarge
	}
	content, err := readRuleSetBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, errors.New("empty rule-set body")
	}
	return content, nil
}
