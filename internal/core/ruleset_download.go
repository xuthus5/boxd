package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// raceDownloadContent 竞速下载：同时向主地址与 jsDelivr 镜像发起请求，取最先成功的内容。
// 串行退避在源全部不通时最长要等待 sum(每源超时)；并行耗时只取决于最快的源。
// 下载完成后立即取消其余源并等待它们退出，避免其写入与调用方并发。
func (i *LoyalsoldierRuleSetInstaller) raceDownloadContent(ctx context.Context, tag, primary string) ([]byte, error) {
	candidates := ruleSetSourceURLs(primary)
	if len(candidates) == 1 {
		return i.downloadContent(ctx, tag, candidates[0])
	}
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan downloadRaceResult, len(candidates))
	var wg sync.WaitGroup
	wg.Add(len(candidates))
	for _, candidate := range candidates {
		go func(candidate string) {
			defer wg.Done()
			content, err := i.downloadContent(raceCtx, tag, candidate)
			results <- downloadRaceResult{content: content, err: err}
		}(candidate)
	}
	var lastErr error
	for range candidates {
		result := <-results
		if result.err == nil {
			cancel()
			wg.Wait()
			return result.content, nil
		}
		if lastErr == nil {
			lastErr = result.err
		}
	}
	wg.Wait()
	return nil, lastErr
}

type downloadRaceResult struct {
	content []byte
	err     error
}

// ruleSetSourceURLs 返回该源的下载地址序列：主地址优先，其次为 jsDelivr
// GitHub CDN 镜像；raw.githubusercontent.com 不可达时自动退避到镜像源。
func ruleSetSourceURLs(primary string) []string {
	mirrors := jsdelivrMirrorURLs(primary)
	urls := make([]string, 0, 1+len(mirrors))
	urls = append(urls, primary)
	urls = append(urls, mirrors...)
	return urls
}

// jsdelivrMirrorURLs 将 raw.githubusercontent.com 地址映射为官方 jsDelivr
// CDN 地址（release 分支文件，见 v2ray-rules-dat README 的推荐下载地址）。
func jsdelivrMirrorURLs(rawURL string) []string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() != "raw.githubusercontent.com" {
		return nil
	}
	parts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 4)
	if len(parts) != 4 {
		return nil
	}
	cdnURL := "https://cdn.jsdelivr.net/gh/" + parts[0] + "/" + parts[1] + "@" + parts[2] + "/" + parts[3]
	return []string{cdnURL, "https://fastly.jsdelivr.net/" + strings.TrimPrefix(cdnURL, "https://cdn.jsdelivr.net/")}
}

func (i *LoyalsoldierRuleSetInstaller) downloadContent(ctx context.Context, tag, sourceURL string) ([]byte, error) {
	if err := ValidatePublicHTTPURL(sourceURL); err != nil {
		return nil, fmt.Errorf("download %s: %w", tag, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", tag, err)
	}
	client := i.client
	if client == nil {
		client = newPublicHTTPClient(ruleSetInstallerHTTPTimeout)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", tag, err)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("download %s: response body is nil", tag)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %d", tag, resp.StatusCode)
	}
	if resp.ContentLength > maxRuleSetBodyBytes {
		return nil, fmt.Errorf("download %s: %w", tag, ErrRuleSetContentTooLarge)
	}
	content, err := readRuleSetBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", tag, err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("download %s: empty rule-set body", tag)
	}
	return content, nil
}
