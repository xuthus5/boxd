package core

import (
	"errors"
	"strings"
	"testing"
)

type ruleSetReadError struct{}

func (ruleSetReadError) Read([]byte) (int, error) {
	return 0, errors.New("rule-set read failed")
}

func TestReadRuleSetBodyLimitsAndErrors(t *testing.T) {
	if _, err := readRuleSetBody(nil); err == nil {
		t.Fatal("expected nil reader error")
	}
	if _, err := readRuleSetBody(ruleSetReadError{}); err == nil {
		t.Fatal("expected reader error")
	}
	content, err := readRuleSetBody(strings.NewReader(strings.Repeat("x", maxRuleSetBodyBytes)))
	if err != nil || len(content) != maxRuleSetBodyBytes {
		t.Fatalf("boundary content len=%d err=%v", len(content), err)
	}
	if _, err := readRuleSetBody(strings.NewReader(strings.Repeat("x", maxRuleSetBodyBytes+1))); !errors.Is(err, ErrRuleSetContentTooLarge) {
		t.Fatalf("oversized error = %v", err)
	}
}
