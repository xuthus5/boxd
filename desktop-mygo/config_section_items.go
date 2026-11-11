package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

const configItemEditorHeight = 300

func (a *application) configSectionItemsCard(c *ui.Context, state *configSectionState) {
	items, ok := configSectionObjectItems(state)
	if !ok {
		return
	}
	card(c, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			sectionHeading(c, state.spec.Title+" items", fmt.Sprintf("%d configured item(s)", len(items)))
			ui.Spacer(c)
			if ui.PrimaryButton(c, "Add item").Key("config-item-add-" + state.spec.Key).Clicked() {
				a.openConfigSectionItemEditor(state, -1)
			}
		})
		if len(items) == 0 {
			emptyState(c, "No items configured", "Add an item or use the raw JSON editor below for advanced configuration.")
			return
		}
		for index, item := range items {
			a.configSectionItemRow(c, state, index, item)
		}
	})
}

func (a *application) configSectionItemRow(c *ui.Context, state *configSectionState, index int, item map[string]any) {
	tag := firstText(jsonPathString(item, "tag"), "(untagged)")
	typ := firstText(jsonPathString(item, "type"), "unknown type")
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, tag).FontSize(12).Bold()
			ui.Text(c, typ+" · "+configItemDetail(item)).FontSize(11).TextColor(c.Theme().TextMuted)
		})
		if ui.Button(c, "Edit").Key(fmt.Sprintf("config-item-edit-%s-%d", state.spec.Key, index)).Clicked() {
			a.openConfigSectionItemEditor(state, index)
		}
		if ui.Button(c, "Delete").Key(fmt.Sprintf("config-item-delete-%s-%d", state.spec.Key, index)).Clicked() {
			a.askDeleteConfigSectionItem(state, index, tag)
		}
	})
}

func (a *application) openConfigSectionItemEditor(state *configSectionState, index int) {
	items, ok := configSectionObjectItems(state)
	if !ok {
		return
	}
	state.itemErr = ""
	state.itemEditIndex = index
	if index >= 0 && index < len(items) {
		body, err := json.MarshalIndent(items[index], "", "  ")
		if err != nil {
			a.model.setActionError(err)
			a.invalidate()
			return
		}
		state.itemDraft = string(body) + "\n"
	} else {
		state.itemDraft = "{\n  \"tag\": \"\",\n  \"type\": \"\"\n}\n"
	}
	state.itemEditOpen = true
	state.itemErr = ""
	a.invalidate()
}

func (a *application) configSectionItemDialog(c *ui.Context) {
	for _, state := range a.configSections {
		if state.itemEditOpen && a.router.Path() == state.spec.Route {
			configSectionItemDialog(c, a, state)
			return
		}
	}
}

func configSectionItemDialog(c *ui.Context, app *application, state *configSectionState) {
	ui.Modal(c, &state.itemEditOpen, func() {
		title := "Add " + strings.TrimSuffix(state.spec.Title, "s") + " item"
		if state.itemEditIndex >= 0 {
			title = "Edit " + strings.TrimSuffix(state.spec.Title, "s") + " item"
		}
		dialogPanel(c, title, "Edit this item as native JSON. Unknown fields are preserved when saved.", func() {
			ui.TextArea(c.Key("config-item-editor-"+state.spec.Key), &state.itemDraft).
				Height(configItemEditorHeight).AutoFocus()
			if state.itemErr != "" {
				ui.Text(c, state.itemErr).FontSize(12).TextColor(c.Theme().Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Key("config-item-cancel-" + state.spec.Key).Clicked() {
					state.itemEditOpen = false
				}
				if ui.PrimaryButton(c, "Save item").Key("config-item-save-" + state.spec.Key).Clicked() {
					app.saveConfigSectionItem(state)
				}
			})
		})
	})
}

func (a *application) saveConfigSectionItem(state *configSectionState) {
	item, err := parseConfigSectionItem(state.itemDraft)
	if err != nil {
		state.itemErr = err.Error()
		a.invalidate()
		return
	}
	items, ok := configSectionObjectItems(state)
	if !ok {
		state.itemErr = "current section is not an array of objects"
		a.invalidate()
		return
	}
	if state.itemEditIndex >= 0 && state.itemEditIndex < len(items) {
		items[state.itemEditIndex] = item
	} else {
		items = append(items, item)
	}
	if err := state.replaceConfigItems(items); err != nil {
		state.itemErr = err.Error()
		a.invalidate()
		return
	}
	state.itemEditOpen = false
	state.itemErr = ""
	a.model.actionError = ""
	a.model.notice = state.spec.Title + " item saved to the draft"
	a.invalidate()
}

func (a *application) askDeleteConfigSectionItem(state *configSectionState, index int, tag string) {
	a.deleteForm = deleteFormState{
		open: true, kind: "config-section-item", id: state.spec.Key,
		ids: []string{strconv.Itoa(index)}, title: "Delete configuration item",
		message: "Remove " + tag + " from " + state.spec.Title + "? The change is only saved when you Apply the section.",
		confirm: "Delete",
	}
}

func (a *application) deleteConfigSectionItem(section string, indexText string) {
	state := a.configSections[section]
	if state == nil {
		return
	}
	index, err := strconv.Atoi(indexText)
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	items, ok := configSectionObjectItems(state)
	if !ok || index < 0 || index >= len(items) {
		a.model.setActionError(errors.New("configuration item does not exist"))
		a.invalidate()
		return
	}
	items = append(items[:index], items[index+1:]...)
	if err := state.replaceConfigItems(items); err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.model.actionError = ""
	a.model.notice = state.spec.Title + " item removed from the draft"
	a.invalidate()
}

func configSectionObjectItems(state *configSectionState) ([]map[string]any, bool) {
	if state == nil || !state.loaded {
		return nil, false
	}
	value, err := decodeJSONValue(state.editorState().Buffer.String())
	if err != nil {
		return nil, false
	}
	rawItems, ok := value.([]any)
	if !ok {
		return nil, false
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		items = append(items, item)
	}
	return items, true
}

func (s *configSectionState) replaceConfigItems(items []map[string]any) error {
	body, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	s.editorState().Buffer.Set(string(body) + "\n")
	return nil
}

func parseConfigSectionItem(text string) (map[string]any, error) {
	value, err := decodeJSONValue(text)
	if err != nil {
		return nil, err
	}
	item, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("configuration item must be a JSON object")
	}
	body, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	var normalized map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func configItemDetail(item map[string]any) string {
	for _, key := range []string{"listen", "server", "address", "path"} {
		if value := strings.TrimSpace(jsonPathString(item, key)); value != "" {
			return key + "=" + value
		}
	}
	return fmt.Sprintf("%d field(s)", len(item))
}
