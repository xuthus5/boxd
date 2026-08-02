package service

import (
	"context"
	"runtime"

	singboxconstant "github.com/sagernet/sing-box/constant"
)

// Kernel 提供内核版本、内存与 GC 等用例逻辑。
type Kernel struct{ version string }

// KernelVersion 返回 boxd 与 sing-box 内核版本。
func (s *Kernel) KernelVersion(_ context.Context) map[string]string {
	return map[string]string{
		"version":        s.version,
		"kernel_version": singboxconstant.Version,
	}
}

// KernelMemory 返回 Go 运行时内存统计。
func (s *Kernel) KernelMemory(_ context.Context) map[string]any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]any{
		"alloc":         m.Alloc,
		"total":         m.TotalAlloc,
		"sys":           m.Sys,
		"num_gc":        m.NumGC,
		"heap_inuse":    m.HeapInuse,
		"stack_inuse":   m.StackInuse,
		"num_goroutine": runtime.NumGoroutine(),
	}
}

// KernelGC 触发一次完整垃圾回收。
func (s *Kernel) KernelGC(_ context.Context) error {
	runtime.GC()
	return nil
}
