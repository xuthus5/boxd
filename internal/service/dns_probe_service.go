package service

import (
	"context"
	"sync"
)

// DNSProbe 提供 DNS 服务器可达性与解析延迟探测用例逻辑。
type DNSProbe struct{}

// defaultBatchConcurrency 默认批量探测并发数。
const defaultBatchConcurrency = 8

// Probe 探测单个 DNS 服务器。
func (s *DNSProbe) Probe(ctx context.Context, req DNSProbeRequest) (DNSProbeResult, error) {
	result := probeDNSServer(ctx, req)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

// ProbeBatch 并发探测多个 DNS 服务器。
func (s *DNSProbe) ProbeBatch(ctx context.Context, items []DNSProbeRequest, concurrency int) ([]DNSProbeResult, error) {
	if len(items) == 0 {
		return nil, Errorf(400, "invalid_request", "items is empty")
	}
	if len(items) > maxDNSProbeItems {
		return nil, Errorf(400, "invalid_request", "too many dns probe items")
	}
	if concurrency <= 0 {
		concurrency = defaultBatchConcurrency
	}
	if concurrency > maxDNSProbeConcurrency {
		concurrency = maxDNSProbeConcurrency
	}

	results, completed := runDNSProbeBatch(ctx, items, concurrency)
	if !completed {
		return nil, ctx.Err()
	}
	return results, nil
}

// runDNSProbeBatch 固定数量 worker 并发探测。
func runDNSProbeBatch(ctx context.Context, items []DNSProbeRequest, concurrency int) ([]DNSProbeResult, bool) {
	results := make([]DNSProbeResult, len(items))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workerCount := min(max(concurrency, 1), len(items))
	for range workerCount {
		wg.Go(func() {
			for {
				if ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					results[index] = probeDNSServer(ctx, items[index])
				}
			}
		})
	}
	completed := true
sendJobs:
	for index := range items {
		select {
		case <-ctx.Done():
			completed = false
			break sendJobs
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	return results, completed && ctx.Err() == nil
}
