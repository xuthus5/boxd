package main

import "testing"

func TestParseDeepLink(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty", raw: "  "},
		{name: "valid host", raw: "boxd://import?link=hysteria2%3A%2F%2Fexample", want: "hysteria2://example"},
		{name: "valid path", raw: "boxd:///import?link=vless%3A%2F%2Fexample", want: "vless://example"},
		{name: "unknown action", raw: "boxd://settings"},
		{name: "wrong scheme", raw: "https://import?link=x", wantErr: true},
		{name: "missing link", raw: "boxd://import", wantErr: true},
		{name: "blank link", raw: "boxd://import?link=%20", wantErr: true},
		{name: "invalid escape", raw: "boxd://import?link=%zz", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDeepLink(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDeepLink(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseDeepLink(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
