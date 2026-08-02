package main

import (
	"testing"
)

func TestParseBoxdURL(t *testing.T) {
	tests := []struct {
		raw    string
		action string
		query  string
		ok     bool
	}{
		{raw: "boxd://import?link=https%3A%2F%2Fexample.com%2Fsub", action: "import", query: "https://example.com/sub", ok: true},
		{raw: "boxd://show", action: "show", ok: true},
		{raw: "boxd://unknown", action: "unknown", ok: true},
		{raw: "https://example.com", ok: false},
		{raw: "", ok: false},
		{raw: "not-a-url", ok: false},
	}
	for _, test := range tests {
		action, params, ok := ParseBoxdURL(test.raw)
		if ok != test.ok {
			t.Fatalf("%q ok = %v, want %v", test.raw, ok, test.ok)
		}
		if !ok {
			continue
		}
		if action != test.action {
			t.Fatalf("%q action = %q, want %q", test.raw, action, test.action)
		}
		if test.query != "" {
			if got := params.Get("link"); got != test.query {
				t.Fatalf("%q link = %q, want %q", test.raw, got, test.query)
			}
		}
	}
}

func TestParseBoxdURLInvalid(t *testing.T) {
	if _, _, ok := ParseBoxdURL("boxd://%zz"); ok {
		t.Fatal("expected parse failure for invalid url")
	}
}

func TestParseBoxdURLNoHost(t *testing.T) {
	action, params, ok := ParseBoxdURL("boxd:///import?link=abc")
	if !ok {
		t.Fatal("expected ok")
	}
	_ = action
	_ = params
}

func TestURLHandlerNilApp(t *testing.T) {
	h := NewURLHandler(nil, nil)
	h.Register() // 不应 panic
	h.handle("boxd://import?link=abc")
	h.importLink("https://example.com/sub")
	h.focusWindow()
}

func TestURLHandlerNilRT(t *testing.T) {
	h := NewURLHandler(nil, nil)
	// importLink 在 rt nil 时应安全返回
	h.importLink("https://example.com/sub")
}

func TestURLHandlerHandleInvalid(t *testing.T) {
	h := NewURLHandler(nil, nil)
	h.handle("not-a-boxd-url")
}

func TestURLHandlerHandleUnknownAction(t *testing.T) {
	h := NewURLHandler(nil, nil)
	h.handle("boxd://whatever")
}
