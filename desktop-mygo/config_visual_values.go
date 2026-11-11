package main

import (
	"encoding/json"
	"strconv"
	"strings"
)

func decodeSectionObject(text string) (map[string]any, bool) {
	value, err := decodeJSONValue(text)
	if err != nil {
		return nil, false
	}
	object, ok := value.(map[string]any)
	return object, ok
}

func jsonPathValue(root map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = root
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[part]
	}
	return current
}

func jsonPathString(root map[string]any, path string) string {
	switch value := jsonPathValue(root, path).(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

func jsonPathBool(root map[string]any, path string) bool {
	value, _ := jsonPathValue(root, path).(bool)
	return value
}

func setJSONPath(root map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, _ := current[part].(map[string]any)
		if next == nil {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	last := parts[len(parts)-1]
	if value == nil {
		delete(current, last)
		return
	}
	current[last] = value
}

func deleteJSONPath(root map[string]any, path string) {
	setJSONPath(root, path, nil)
}

func sectionVisualKey(section, path string) string {
	return "visual-" + section + "-" + strings.ReplaceAll(path, ".", "-")
}
