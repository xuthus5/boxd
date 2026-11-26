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
