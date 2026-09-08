package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRuleSetConvertPreservesConditions(t *testing.T) {
	t.Parallel()
	content := []byte("# comment\nfull: exact.example\ndomain: example.com\nexample.com\n\n" +
		"keyword: search\nregexp: ^ads\\.\nfull:\n")
	data, err := convertRuleSetData(RuleSetSource{Tag: "test"}, content)
	if err != nil {
		t.Fatal(err)
	}
	var parsed sourceRuleSetFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	want := []sourceRule{{
		Domain: []string{"exact.example"}, DomainSuffix: []string{"example.com"},
		DomainKeyword: []string{"search"}, DomainRegex: []string{`^ads\.`},
	}}
	if !reflect.DeepEqual(want, parsed.Rules) {
		t.Fatalf("rules = %#v, want %#v", parsed.Rules, want)
	}
}

func TestRuleSetConvertSourceErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
	}{
		{name: "no conditions", content: "# comment\nfull:\nkeyword:\nregexp:\ndomain:\n"},
		{name: "line limit", content: strings.Repeat("x", 1<<20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := convertRuleSetData(RuleSetSource{Tag: "test"}, []byte(tt.content)); err == nil {
				t.Fatal("expected invalid source error")
			}
		})
	}
}
