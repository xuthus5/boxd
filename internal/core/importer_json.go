package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseLinkJSONInt(raw json.RawMessage, field string) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("invalid %s", field)
	}
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", field, text)
	}
	return value, nil
}

func parseLinkJSONBool(raw json.RawMessage, field string) (bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return false, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil && (number == 0 || number == 1) {
		return number == 1, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return false, fmt.Errorf("invalid %s", field)
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s %q", field, text)
	}
}
