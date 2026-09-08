package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/route/rule"
)

func cachedRuleSetValid(ctx context.Context, path, format string) (bool, error) {
	content, err := readCachedRuleSet(path)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrRuleSetContentTooLarge) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read cache: %w", err)
	}
	// 缓存语法错误需要恢复快照，I/O 错误则保留现场并中止初始化。
	if err := validateRuleSetData(ctx, content, format); err != nil {
		return false, nil
	}
	return true, nil
}

func readCachedRuleSet(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("rule-set cache is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return readRuleSetBody(file)
}

func validateRuleSetData(ctx context.Context, data []byte, format string) error {
	var parsed option.PlainRuleSetCompat
	var err error
	if format == "binary" || format == "cidr" {
		parsed, err = srs.Read(bytes.NewReader(data), false)
	} else {
		err = json.Unmarshal(data, &parsed)
	}
	if err != nil {
		return err
	}
	if len(parsed.Options.Rules) == 0 {
		return fmt.Errorf("rule-set has no rules")
	}
	for _, options := range parsed.Options.Rules {
		if _, err := rule.NewHeadlessRule(ctx, options); err != nil {
			return err
		}
	}
	return nil
}
