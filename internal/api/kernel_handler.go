package api

import (
	"net/http"
	"runtime"

	singboxconstant "github.com/sagernet/sing-box/constant"
)

// KernelHandler 暴露内核级全局信息与维护操作：版本、内存、手动 GC。
// 这些能力不依赖运行时实例，可独立于 sing-box 是否启动使用。
type KernelHandler struct {
	version string
}

// NewKernelHandler 用 boxui 编译期版本号构造 handler。
func NewKernelHandler(version string) *KernelHandler {
	return &KernelHandler{version: version}
}

// Version GET /api/runtime/version
// 返回 boxui 应用版本与 sing-box 内核版本。
func (h *KernelHandler) Version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":        h.version,
		"kernel_version": singboxconstant.Version,
	})
}

// Memory GET /api/runtime/memory —— 暴露 Go 运行时内存统计，供仪表盘展示。
func (h *KernelHandler) Memory(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	writeJSON(w, http.StatusOK, map[string]any{
		"alloc":       m.Alloc,
		"total":       m.TotalAlloc,
		"sys":         m.Sys,
		"num_gc":      m.NumGC,
		"heap_inuse":  m.HeapInuse,
		"stack_inuse": m.StackInuse,
	})
}

// GC POST /api/runtime/gc —— 触发一次完整垃圾回收。
func (h *KernelHandler) GC(w http.ResponseWriter, r *http.Request) {
	runtime.GC()
	writeJSON(w, http.StatusOK, nil)
}
