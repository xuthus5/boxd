package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *application) configSectionVisualForm(c *ui.Context, state *configSectionState) {
	object, ok := decodeSectionObject(state.editorState().Buffer.String())
	if !ok {
		return
	}
	fields := configVisualFields(state.spec.Key)
	if len(fields) == 0 {
		return
	}
	card(c, func() {
		sectionHeading(c, "Visual editor", "Common fields are edited natively; raw JSON below remains authoritative for advanced options")
		for _, field := range fields {
			a.configVisualField(c, state, object, field)
		}
	})
}

func (a *application) configVisualField(c *ui.Context, state *configSectionState, object map[string]any, field configVisualField) {
	key := sectionVisualKey(state.spec.Key, field.Path)
	switch field.Kind {
	case configVisualBool:
		value := jsonPathBool(object, field.Path)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, field.Label).Grow(1).FontSize(12)
			if ui.Switch(c.Key(key), &value).Changed() {
				a.commitVisualField(state, object, field.Path, value, false)
			}
		})
	case configVisualSelect:
		value := jsonPathString(object, field.Path)
		if value == "" && len(field.Options) > 0 {
			value = field.Options[0]
		}
		ui.Field(c, field.Label, func() {
			if ui.Select(c.Key(key), &value, field.Options).Changed() {
				a.commitVisualField(state, object, field.Path, value, false)
			}
		})
	case configVisualNumber:
		value := jsonPathString(object, field.Path)
		ui.Field(c, field.Label, func() {
			if ui.TextInput(c.Key(key), &value).Changed() {
				a.commitVisualNumber(state, object, field.Path, value)
			}
		})
	case configVisualSecret:
		value := jsonPathString(object, field.Path)
		ui.Field(c, field.Label, func() {
			if ui.TextInput(c.Key(key), &value).Password().Changed() {
				a.commitVisualText(state, object, field.Path, value)
			}
		})
	default:
		value := jsonPathString(object, field.Path)
		ui.Field(c, field.Label, func() {
			if ui.TextInput(c.Key(key), &value).Placeholder(field.Help).Changed() {
				a.commitVisualText(state, object, field.Path, value)
			}
		})
	}
}

func (a *application) commitVisualText(state *configSectionState, object map[string]any, path, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		a.commitVisualField(state, object, path, nil, true)
		return
	}
	a.commitVisualField(state, object, path, value, false)
}

func (a *application) commitVisualNumber(state *configSectionState, object map[string]any, path, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		a.commitVisualField(state, object, path, nil, true)
		return
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.commitVisualField(state, object, path, json.Number(strconv.FormatInt(number, 10)), false)
}

func (a *application) commitVisualField(state *configSectionState, object map[string]any, path string, value any, remove bool) {
	if remove {
		deleteJSONPath(object, path)
	} else {
		setJSONPath(object, path, value)
	}
	body, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	state.editorState().Buffer.Set(string(body) + "\n")
	a.model.actionError = ""
	a.model.notice = state.spec.Title + " field updated: " + path
	a.invalidate()
}
