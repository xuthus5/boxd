package core

import (
	"bufio"
	"bytes"
	"fmt"
	"net/netip"
	"strings"

	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func convertCIDRRuleSet(tag string, content []byte) ([]byte, error) {
	prefixes, err := parseRuleSetCIDRs(tag, content)
	if err != nil {
		return nil, err
	}
	ruleSet := option.PlainRuleSet{Rules: []option.HeadlessRule{{
		Type:           C.RuleTypeDefault,
		DefaultOptions: option.DefaultHeadlessRule{IPCIDR: prefixes},
	}}}
	var data bytes.Buffer
	if err := srs.Write(&data, ruleSet, C.RuleSetVersionCurrent); err != nil {
		return nil, fmt.Errorf("compile %s: %w", tag, err)
	}
	return data.Bytes(), nil
}

func parseRuleSetCIDRs(tag string, content []byte) ([]string, error) {
	prefixes := make([]string, 0)
	var ipv4, ipv6 bool
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			return nil, fmt.Errorf("parse %s CIDR %q: %w", tag, line, err)
		}
		ipv4 = ipv4 || prefix.Addr().Is4()
		ipv6 = ipv6 || prefix.Addr().Is6()
		prefixes = append(prefixes, prefix.Masked().String())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s CIDRs: %w", tag, err)
	}
	if !ipv4 || !ipv6 {
		return nil, fmt.Errorf("source %s must contain both IPv4 and IPv6 prefixes", tag)
	}
	return uniqueStrings(prefixes), nil
}
