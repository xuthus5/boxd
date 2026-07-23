package core

import (
	"errors"
	"os"
	"testing"
)

func TestClassifyKernelError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		msg  string
		err  error
		want string
	}{
		{name: "missing file", err: os.ErrNotExist, want: KernelErrorConfigMissing},
		{name: "permission", err: os.ErrPermission, want: KernelErrorPermission},
		{name: "restart", msg: "restart failed after config save: boom", want: KernelErrorRestartFailed},
		{name: "decode", msg: "decode config at outbounds[0].server: invalid", want: KernelErrorConfigInvalid},
		{name: "invalid outbound", msg: "invalid outbound", want: KernelErrorConfigInvalid},
		{name: "start", msg: "start failed: address already in use", want: KernelErrorStartFailed},
		{name: "listen", msg: "listen tcp :1080: bind: address already in use", want: KernelErrorStartFailed},
		{name: "unknown", msg: "something else", want: KernelErrorUnknown},
		{name: "empty", msg: "", want: KernelErrorUnknown},
		{name: "wrapped missing", err: errors.Join(os.ErrNotExist), want: KernelErrorConfigMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyKernelError(tt.msg, tt.err)
			if got != tt.want {
				t.Fatalf("ClassifyKernelError(%q, %v)=%q want %q", tt.msg, tt.err, got, tt.want)
			}
		})
	}
}
