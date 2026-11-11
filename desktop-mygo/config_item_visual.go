package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	configVisualPortMin = 1
	configVisualPortMax = 65535
)

func (a *application) configItemVisualForm(c *ui.Context, state *configSectionState, item map[string]any) {
	fields := configItemVisualFields(state.spec.Key, state.itemPath)
	if len(fields) == 0 {
		return
	}
	ui.Text(c, "Common fields").FontSize(12).Bold()
	ui.Text(c, "Changes update the JSON below; unknown fields are preserved.").
		FontSize(11).TextColor(c.Theme().TextMuted)
	for _, field := range fields {
		a.configItemVisualField(c, state, item, field)
	}
}

func (a *application) configItemVisualField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField) {
	key := configItemVisualKey(state, field.Path)
	switch field.Kind {
	case configVisualBool:
		a.configItemVisualBoolField(c, state, item, field, key)
	case configVisualSelect:
		a.configItemVisualSelectField(c, state, item, field, key)
	case configVisualNumber:
		a.configItemVisualNumberField(c, state, item, field, key)
	case configVisualStringList:
		a.configItemVisualStringListField(c, state, item, field, key)
	case configVisualNumberList:
		a.configItemVisualNumberListField(c, state, item, field, key)
	default:
		a.configItemVisualTextField(c, state, item, field, key)
	}
}

func (a *application) configItemVisualBoolField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := jsonPathBool(item, field.Path)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, field.Label).Grow(1).FontSize(12)
		if ui.Switch(c.Key(key), &value).Changed() {
			a.commitConfigItemVisual(state, item, field.Path, value, false)
		}
	})
}

func (a *application) configItemVisualSelectField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := jsonPathString(item, field.Path)
	if value == "" && len(field.Options) > 0 {
		value = field.Options[0]
	}
	ui.Field(c, field.Label, func() {
		if ui.Select(c.Key(key), &value, field.Options).Changed() {
			a.commitConfigItemVisual(state, item, field.Path, value, false)
		}
	})
}

func (a *application) configItemVisualNumberField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := jsonPathString(item, field.Path)
	ui.Field(c, field.Label, func() {
		if ui.TextInput(c.Key(key), &value).Changed() {
			a.commitConfigItemNumber(state, item, field.Path, value)
		}
	})
}

func (a *application) configItemVisualStringListField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := configItemListText(item, field.Path)
	ui.Field(c, field.Label, func() {
		if ui.TextInput(c.Key(key), &value).Placeholder(field.Help).Changed() {
			a.commitConfigItemStringList(state, item, field.Path, value)
		}
	})
}

func (a *application) configItemVisualNumberListField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := configItemListText(item, field.Path)
	ui.Field(c, field.Label, func() {
		if ui.TextInput(c.Key(key), &value).Placeholder(field.Help).Changed() {
			a.commitConfigItemNumberList(state, item, field.Path, value)
		}
	})
}

func (a *application) configItemVisualTextField(c *ui.Context, state *configSectionState,
	item map[string]any, field configVisualField, key string) {
	value := jsonPathString(item, field.Path)
	ui.Field(c, field.Label, func() {
		if ui.TextInput(c.Key(key), &value).Placeholder(field.Help).Changed() {
			trimmed := strings.TrimSpace(value)
			a.commitConfigItemVisual(state, item, field.Path, trimmed, trimmed == "")
		}
	})
}

func (a *application) commitConfigItemNumber(state *configSectionState, item map[string]any, path, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		a.commitConfigItemVisual(state, item, path, nil, true)
		return
	}
	number, err := parseConfigItemPort(value)
	if err != nil {
		a.setConfigItemVisualError(state, path, err)
		return
	}
	a.commitConfigItemVisual(state, item, path, number, false)
}

func (a *application) commitConfigItemStringList(state *configSectionState, item map[string]any, path, value string) {
	parts := configItemListParts(value)
	if len(parts) == 0 {
		a.commitConfigItemVisual(state, item, path, nil, true)
		return
	}
	a.commitConfigItemVisual(state, item, path, parts, false)
}

func (a *application) commitConfigItemNumberList(state *configSectionState, item map[string]any, path, value string) {
	parts := configItemListParts(value)
	if len(parts) == 0 {
		a.commitConfigItemVisual(state, item, path, nil, true)
		return
	}
	values := make([]any, 0, len(parts))
	for _, part := range parts {
		number, err := parseConfigItemPort(part)
		if err != nil {
			a.setConfigItemVisualError(state, path, err)
			return
		}
		values = append(values, number)
	}
	a.commitConfigItemVisual(state, item, path, values, false)
}

func parseConfigItemPort(value string) (any, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "-") {
		return parseConfigItemPortRange(value)
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	if number < configVisualPortMin || number > configVisualPortMax {
		return nil, fmt.Errorf("port %d is outside %d-%d", number, configVisualPortMin, configVisualPortMax)
	}
	return json.Number(strconv.FormatInt(number, 10)), nil
}

func parseConfigItemPortRange(value string) (any, error) {
	bounds := strings.Split(value, "-")
	if len(bounds) != 2 {
		return nil, fmt.Errorf("invalid port range %q", value)
	}
	start, err := strconv.ParseInt(strings.TrimSpace(bounds[0]), 10, 64)
	if err != nil {
		return nil, err
	}
	end, err := strconv.ParseInt(strings.TrimSpace(bounds[1]), 10, 64)
	if err != nil {
		return nil, err
	}
	if start < configVisualPortMin || end > configVisualPortMax || start > end {
		return nil, fmt.Errorf("port range %q is outside %d-%d or reversed", value, configVisualPortMin, configVisualPortMax)
	}
	return fmt.Sprintf("%d-%d", start, end), nil
}

func (a *application) commitConfigItemVisual(state *configSectionState, item map[string]any,
	path string, value any, remove bool) {
	if remove {
		deleteJSONPath(item, path)
	} else {
		setJSONPath(item, path, value)
	}
	body, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		a.setConfigItemVisualError(state, path, err)
		return
	}
	state.itemDraft = string(body) + "\n"
	state.itemErr = ""
	a.model.actionError = ""
	a.model.notice = "Item field updated: " + path
	a.invalidate()
}

func (a *application) setConfigItemVisualError(state *configSectionState, path string, err error) {
	state.itemErr = fmt.Sprintf("%s: %v", path, err)
	a.invalidate()
}

func configItemListText(item map[string]any, path string) string {
	switch value := jsonPathValue(item, path).(type) {
	case []any:
		parts := make([]string, 0, len(value))
		for _, entry := range value {
			parts = append(parts, fmt.Sprint(entry))
		}
		return strings.Join(parts, ", ")
	case []string:
		return strings.Join(value, ", ")
	default:
		return jsonPathString(item, path)
	}
}

func configItemListParts(value string) []string {
	raw := strings.FieldsFunc(value, func(char rune) bool {
		return char == ',' || char == '\n' || char == '\r'
	})
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func configItemVisualKey(state *configSectionState, path string) string {
	key := "config-item-visual-" + state.spec.Key
	if state.itemPath != "" {
		key += "-" + strings.ReplaceAll(state.itemPath, "_", "-")
	}
	return key + "-" + strings.ReplaceAll(path, "_", "-")
}
