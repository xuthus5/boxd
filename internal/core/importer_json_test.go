package core

import (
	"encoding/json"
	"testing"
)

func TestParseLinkJSONInt(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want int
	}{
		{raw: "", want: 0},
		{raw: "null", want: 0},
		{raw: "443", want: 443},
		{raw: `"8443"`, want: 8443},
	} {
		got, err := parseLinkJSONInt(json.RawMessage(test.raw), "port")
		if err != nil || got != test.want {
			t.Errorf("parseLinkJSONInt(%q) = %d, %v", test.raw, got, err)
		}
	}
	for _, raw := range []string{`"bad"`, "1.5", "{}"} {
		if _, err := parseLinkJSONInt(json.RawMessage(raw), "port"); err == nil {
			t.Errorf("parseLinkJSONInt(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestParseLinkJSONBool(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want bool
	}{
		{raw: "", want: false},
		{raw: "null", want: false},
		{raw: "true", want: true},
		{raw: "1", want: true},
		{raw: `"false"`, want: false},
	} {
		got, err := parseLinkJSONBool(json.RawMessage(test.raw), "flag")
		if err != nil || got != test.want {
			t.Errorf("parseLinkJSONBool(%q) = %v, %v", test.raw, got, err)
		}
	}
	for _, raw := range []string{"2", `"maybe"`, "{}"} {
		if _, err := parseLinkJSONBool(json.RawMessage(raw), "flag"); err == nil {
			t.Errorf("parseLinkJSONBool(%q) unexpectedly succeeded", raw)
		}
	}
}
