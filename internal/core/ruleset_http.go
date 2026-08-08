package core

import (
	"errors"
	"io"
	"time"
)

const (
	ruleSetHTTPTimeout          = 60 * time.Second
	ruleSetInstallerHTTPTimeout = 60 * time.Second
	maxRuleSetBodyBytes         = 16 << 20
	maxRuleSetEtagBytes         = 8 << 10
)

var ErrRuleSetContentTooLarge = errors.New("rule-set content is too large")

func readRuleSetBody(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("rule-set response body is nil")
	}
	content, err := io.ReadAll(io.LimitReader(reader, maxRuleSetBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxRuleSetBodyBytes {
		return nil, ErrRuleSetContentTooLarge
	}
	return content, nil
}
