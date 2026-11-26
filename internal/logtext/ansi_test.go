package logtext_test

import (
	"testing"

	"github.com/xuthus5/boxd/internal/logtext"
)

func TestStripTerminalFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "colored level",
			input: "\x1b[37mTRACE\x1b[0m[0364] outbound: connected",
			want:  "TRACE[0364] outbound: connected",
		},
		{
			name:  "extended color",
			input: "\x1b[38;5;196merror\x1b[0m: failed",
			want:  "error: failed",
		},
		{
			name:  "hyperlink",
			input: "open \x1b]8;;https://example.com\aexample\x1b]8;;\adoc",
			want:  "open exampledoc",
		},
		{
			name:  "malformed trailing sequence",
			input: "message\x1b[38;5;",
			want:  "message",
		},
		{
			name:  "escape byte removed upstream",
			input: "[37mTRACE[0m[0364] outbound: connected",
			want:  "TRACE[0364] outbound: connected",
		},
		{
			name:  "legitimate bracketed text",
			input: "dns/local[dns-bootstrap]: outbound/vless[42 12ms] connected",
			want:  "dns/local[dns-bootstrap]: outbound/vless[42 12ms] connected",
		},
		{
			name:  "unicode is preserved",
			input: "雪 \x1b[32m正常\x1b[0m",
			want:  "雪 正常",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := logtext.StripTerminalFormatting(test.input); got != test.want {
				t.Fatalf("StripTerminalFormatting(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestCleanMessageRemovesDuplicateStructuredLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		input string
		want  string
	}{
		{
			name:  "terminal colored level",
			level: "trace",
			input: "\x1b[37mTRACE\x1b[0m[0364] outbound: connected",
			want:  "outbound: connected",
		},
		{
			name:  "escape byte removed upstream",
			level: "trace",
			input: "[37mTRACE[0m[0364] outbound: connected",
			want:  "outbound: connected",
		},
		{
			name:  "plain structured prefix",
			level: "debug",
			input: "DEBUG[18] dns/local[dns-bootstrap]: ready",
			want:  "dns/local[dns-bootstrap]: ready",
		},
		{
			name:  "bracketed level",
			level: "info",
			input: "[INFO] kernel initialized",
			want:  "kernel initialized",
		},
		{
			name:  "colon level alias",
			level: "warning",
			input: "WARN: outbound timeout",
			want:  "outbound timeout",
		},
		{
			name:  "unrelated bracket content is preserved",
			level: "info",
			input: "dns/local[dns-bootstrap]: systemd-resolved available",
			want:  "dns/local[dns-bootstrap]: systemd-resolved available",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := logtext.CleanMessage(test.level, test.input); got != test.want {
				t.Fatalf("CleanMessage(%q, %q) = %q, want %q", test.level, test.input, got, test.want)
			}
		})
	}
}
