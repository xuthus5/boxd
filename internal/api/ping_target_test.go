package api

import (
	"strings"
	"testing"
)

func TestIsValidPingTarget(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// 合法 IP
		{"1.2.3.4", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"2001:db8::1", true},
		// 合法域名
		{"example.com", true},
		{"sub.example.com", true},
		{"my-server.example.org", true},
		// 非法输入
		{"", false},
		{"localhost", false}, // 无点
		{"example.com; rm -rf /", false},
		{"$(whoami)", false},
		{"example.com|cat", false},
		{strings.Repeat("a", 254), false}, // 超长
		{"exa mple.com", false},           // 含空格
		{"example\n.com", false},          // 含换行
		{"example.com`id`", false},        // 反引号
		{"ex{ample}.com", false},          // 大括号
	}

	for _, tt := range tests {
		got := isValidPingTarget(tt.input)
		if got != tt.want {
			t.Errorf("isValidPingTarget(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
