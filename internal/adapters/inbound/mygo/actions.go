package mygo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func (c *Controller) startKernel() {
	c.run("启动内核", func(ctx context.Context, app control.Application) error { return app.Kernel().Start(ctx) })
}

func (c *Controller) stopKernel() {
	c.run("停止内核", func(ctx context.Context, app control.Application) error { return app.Kernel().Stop(ctx) })
}

func (c *Controller) restartKernel() {
	c.run("重启内核", func(ctx context.Context, app control.Application) error { return app.Kernel().Restart(ctx) })
}

func (c *Controller) refreshSubscription(id string) {
	c.run("刷新订阅", func(ctx context.Context, app control.Application) error {
		_, err := app.Subscriptions().Refresh(ctx, id)
		return err
	})
}

func (c *Controller) refreshAllSubscriptions() {
	c.run("刷新全部订阅", func(ctx context.Context, app control.Application) error {
		_, err := app.Subscriptions().RefreshAll(ctx)
		return err
	})
}

func (c *Controller) runSchedulerOnce() {
	c.run("同步订阅", func(ctx context.Context, app control.Application) error {
		_, err := app.Scheduler().RunOnce(ctx)
		return err
	})
}

func (c *Controller) refreshRuleSet(name string) {
	c.run("刷新规则集", func(ctx context.Context, app control.Application) error {
		_, err := app.RuleSets().Refresh(ctx, name)
		return err
	})
}

func (c *Controller) refreshAllRuleSets() {
	c.run("刷新全部规则集", func(ctx context.Context, app control.Application) error {
		_, err := app.RuleSets().RefreshAll(ctx)
		return err
	})
}

func (c *Controller) flushDNS() {
	c.run("刷新 DNS 缓存", func(ctx context.Context, app control.Application) error { return app.DNS().Flush(ctx) })
}

func (c *Controller) testDNS() {
	if c.runtime == nil || c.busy {
		return
	}
	c.busy = true
	c.dnsResults = nil
	c.notice = "DNS 探测中…"
	c.noticeErr = false
	c.window.Invalidate()
	app := c.runtime.Control
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 20*time.Second)
		defer cancel()
		results, err := app.DNS().Test(ctx)
		c.window.Update(func() {
			c.busy = false
			if err != nil {
				c.notice = "DNS 探测失败：" + err.Error()
				c.noticeErr = true
			} else {
				c.dnsResults = results
				c.notice = "DNS 探测完成"
				c.noticeErr = false
			}
			c.refresh(false)
		})
	}()
}

func (c *Controller) testNodes(ids []string, kind control.NodeTestKind, target string) {
	if c.runtime == nil || c.busy || len(ids) == 0 {
		return
	}
	c.busy = true
	c.notice = "节点测速中…"
	c.noticeErr = false
	for _, id := range ids {
		c.nodeTests[id] = "测速中…"
	}
	c.window.Invalidate()
	app := c.runtime.Control
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 90*time.Second)
		defer cancel()
		const batchSize = 200
		outcomes := make([]control.NodeTestOutcome, 0, len(ids))
		var firstErr error
		for start := 0; start < len(ids); start += batchSize {
			end := min(start+batchSize, len(ids))
			batch, err := app.Nodes().Test(ctx, control.NodeTestRequest{NodeIDs: ids[start:end], Kind: kind, URL: target})
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			outcomes = append(outcomes, batch...)
		}
		c.window.Update(func() {
			c.busy = false
			for _, outcome := range outcomes {
				if outcome.Error != "" {
					c.nodeTests[outcome.NodeID] = "失败 · " + oneLine(outcome.Error)
				} else {
					c.nodeTests[outcome.NodeID] = duration(outcome.Latency)
				}
			}
			if firstErr != nil {
				c.notice = "部分节点测速失败：" + firstErr.Error()
				c.noticeErr = true
			} else {
				c.notice = fmt.Sprintf("已完成 %d 个节点测速", len(outcomes))
				c.noticeErr = false
			}
			c.refresh(false)
		})
	}()
}

func (c *Controller) testAllNodes(kind control.NodeTestKind, target string) {
	ids := make([]string, 0, len(c.snapshot.Catalog.Nodes))
	for _, node := range c.snapshot.Catalog.Nodes {
		ids = append(ids, node.ID)
	}
	c.testNodes(ids, kind, target)
}

func (c *Controller) selectGroupMember(groupID, nodeID string) {
	if nodeID == "" {
		return
	}
	c.run("切换策略组节点", func(ctx context.Context, app control.Application) error {
		return app.Egress().SelectGroupMember(ctx, groupID, nodeID)
	})
}

func (c *Controller) setRouteMode(mode control.RouteMode) {
	if mode == c.snapshot.Routes.Mode {
		return
	}
	c.run("切换路由模式", func(ctx context.Context, app control.Application) error {
		_, err := app.Routes().SetMode(ctx, mode)
		return err
	})
}

func (c *Controller) setDefaultEgress(ref control.EgressRef) {
	c.run("设置默认出口", func(ctx context.Context, app control.Application) error {
		_, err := app.Routes().SetDefaultEgress(ctx, ref)
		return err
	})
}

func (c *Controller) setPresets(presets control.RoutePresets) {
	c.run("更新预设策略", func(ctx context.Context, app control.Application) error {
		_, err := app.Routes().SetPresets(ctx, presets)
		return err
	})
}

func (c *Controller) moveRule(id string, delta int) {
	if c.runtime == nil || c.busy {
		return
	}
	rules := append([]control.RouteRule(nil), c.snapshot.Routes.Rules...)
	index := -1
	for i := range rules {
		if rules[i].ID == id {
			index = i
			break
		}
	}
	to := index + delta
	if index < 0 || to < 0 || to >= len(rules) {
		return
	}
	rules[index], rules[to] = rules[to], rules[index]
	c.run("调整规则顺序", func(ctx context.Context, app control.Application) error {
		_, err := app.Routes().ReplaceRules(ctx, rules)
		return err
	})
}

func (c *Controller) saveSettings() {
	maxAttempts, err := strconv.Atoi(strings.TrimSpace(c.settingsDraft[control.SettingSubscriptionMaxAttempts]))
	if err != nil || maxAttempts < 1 || maxAttempts > 10 {
		c.notice = "同步重试次数必须是 1 到 10 之间的整数"
		c.noticeErr = true
		c.window.Invalidate()
		return
	}
	values := map[string]string{
		control.SettingAutoRefresh:                strconv.FormatBool(settingBool(c.settingsDraft, control.SettingAutoRefresh, true)),
		control.SettingSubscriptionMaxAttempts:    strconv.Itoa(maxAttempts),
		control.SettingSubscriptionConflictPolicy: strings.TrimSpace(c.settingsDraft[control.SettingSubscriptionConflictPolicy]),
	}
	if values[control.SettingSubscriptionConflictPolicy] == "" {
		values[control.SettingSubscriptionConflictPolicy] = control.SubscriptionConflictKeepLocal
	}
	c.run("保存设置", func(ctx context.Context, app control.Application) error {
		for key, value := range values {
			if _, err := app.Settings().Set(ctx, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Controller) setDNSStrategy(strategy string) {
	if strategy == "" || strategy == c.snapshot.DNS.Strategy {
		return
	}
	c.run("更新 DNS 策略", func(ctx context.Context, app control.Application) error {
		_, err := app.DNS().Patch(ctx, control.DNSProfilePatch{Strategy: &strategy})
		return err
	})
}

func (c *Controller) setDNSCache(enabled bool) {
	if enabled == c.snapshot.DNS.Cache {
		return
	}
	c.run("更新 DNS 缓存", func(ctx context.Context, app control.Application) error {
		_, err := app.DNS().Patch(ctx, control.DNSProfilePatch{Cache: &enabled})
		return err
	})
}

func (c *Controller) closeConnection(id int64) {
	c.run("关闭连接", func(ctx context.Context, app control.Application) error {
		return app.Observability().CloseConnection(ctx, id)
	})
}

func (c *Controller) closeAllConnections() {
	c.run("关闭全部连接", func(ctx context.Context, app control.Application) error {
		count, err := app.Observability().CloseConnections(ctx, kernel.ConnectionFilter{})
		if err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("当前没有可关闭的连接")
		}
		return nil
	})
}

func (c *Controller) copyLog(line mergedLog) {
	text := line.Time.Local().Format("2006-01-02 15:04:05.000") + " [" + strings.ToUpper(line.Level) + "] " + line.Source + " " + line.Message
	mygo.Clipboard.WriteText(text)
	c.notice = "日志已复制到剪贴板"
	c.noticeErr = false
	c.window.Invalidate()
}

func (c *Controller) setLogFilter(value int) {
	if value < 0 || value > 2 {
		return
	}
	c.logFilter = value
	c.window.Invalidate()
}

func (c *Controller) submitEditor() {
	if c.runtime == nil || c.busy {
		return
	}
	editor := c.editor
	c.editorOpen = false
	c.run("保存", func(ctx context.Context, app control.Application) error {
		return c.saveEditor(ctx, app, editor)
	})
}
