package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// webProcessRecovery 在 WebKit 渲染进程终止后自动重载主窗口 UI。
// Linux 上 Wails 不会在渲染进程死亡后自行恢复页面，窗口会永久失去
// 响应（表现为卡死）；这里主动触发 reload 让 WebKit 重建渲染进程。
type webProcessRecovery struct {
	mu sync.Mutex

	// reloadFn 实际执行 UI 重载的函数，main.go 注入。
	reloadFn func()

	// now 注入时钟，便于测试。
	now func() time.Time

	// cooldown 内只触发一次重载，避免同一次崩溃被重复处理。
	cooldown time.Duration

	// rateWindow 内最多 maxReloads 次重载，超出说明渲染进程
	// 持续崩溃（reload 循环），停止自动重载并记录日志。
	rateWindow time.Duration
	maxReloads int
	reloadLog  []time.Time
}

const (
	defaultReloadCooldown  = 3 * time.Second
	defaultReloadRateLimit = 4
	defaultRateWindow      = 5 * time.Minute
)

// webRecovery 全局恢复器，供 GTK 信号回调与托盘使用。
var webRecovery = newWebProcessRecovery(nil)

func newWebProcessRecovery(reloadFn func()) *webProcessRecovery {
	return &webProcessRecovery{
		reloadFn:   reloadFn,
		now:        time.Now,
		cooldown:   defaultReloadCooldown,
		rateWindow: defaultRateWindow,
		maxReloads: defaultReloadRateLimit,
	}
}

// setReloadFn 注入重载函数（main.go 在拿到 app 实例后调用）。
func (r *webProcessRecovery) setReloadFn(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reloadFn = fn
}

// onTerminated 渲染进程终止回调：按冷却与限流规则触发重载。
// 返回是否实际执行了重载。
func (r *webProcessRecovery) onTerminated() bool {
	r.mu.Lock()
	now := r.now()
	if !r.allowReloadLocked(now) {
		r.mu.Unlock()
		return false
	}
	r.reloadLog = append(r.reloadLog, now)
	reloadFn := r.reloadFn
	r.mu.Unlock()

	if reloadFn == nil {
		log.Printf("web process terminated but no reload fn configured")
		return false
	}
	log.Printf("web process terminated, reloading UI (reloads in window: %d)", len(r.reloadLog))
	reloadFn()
	return true
}

// allowReloadLocked 判断当前时间点是否允许重载，调用方需持有 mu。
func (r *webProcessRecovery) allowReloadLocked(now time.Time) bool {
	if len(r.reloadLog) > 0 && now.Sub(r.reloadLog[len(r.reloadLog)-1]) < r.cooldown {
		return false
	}
	recent := 0
	for _, ts := range r.reloadLog {
		if now.Sub(ts) < r.rateWindow {
			recent++
		}
	}
	if recent >= r.maxReloads {
		log.Printf("web process keeps terminating (%d reloads in %v); auto reload suspended", recent, r.rateWindow)
		return false
	}
	return true
}

// reloadMainWindow 重载主窗口页面（托盘"Reload UI"与自动恢复共用）。
func reloadMainWindow(app *application.App) {
	if app == nil {
		return
	}
	if window, ok := app.Window.GetByName("main"); ok {
		window.ForceReload()
	}
}

// enableWebKitWorkarounds 设置 WebKitGTK 渲染相关的环境变量。
// WEBKIT_DISABLE_DMABUF_RENDERER=1 规避 Intel i915 + Mesa 组合下
// 睡眠唤醒/长时间运行后渲染进程崩溃的问题；需在 webview 创建前设置。
func enableWebKitWorkarounds() {
	const key = "WEBKIT_DISABLE_DMABUF_RENDERER"
	if _, present := os.LookupEnv(key); present {
		return
	}
	if err := os.Setenv(key, "1"); err != nil {
		log.Printf("set %s failed: %v", key, err)
	}
}
