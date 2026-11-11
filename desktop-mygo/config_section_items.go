package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
)

const configItemEditorHeight = 300

func (a *application) configSectionItemsCards(c *ui.Context, state *configSectionState) {
	for _, itemPath := range configSectionItemPaths(state.spec.Key) {
		a.configSectionItemsCard(c, state, itemPath)
	}
}

func (a *application) configSectionItemsCard(c *ui.Context, state *configSectionState, itemPath configItemPath) {
	items, ok := configSectionObjectItems(state, itemPath.Path)
	if !ok {
		return
	}
	card(c, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			sectionHeading(c, configItemCardTitle(state, itemPath), fmt.Sprintf("%d configured item(s)", len(items)))
			ui.Spacer(c)
			if ui.PrimaryButton(c, "Add item").Key(configItemKey(state, itemPath.Path, "add", -1)).Clicked() {
				a.openConfigSectionItemEditor(state, itemPath, -1)
			}
		})
		if len(items) == 0 {
			emptyState(c, "No items configured", "Add an item or use the raw JSON editor below for advanced configuration.")
			return
		}
		for index, item := range items {
			a.configSectionItemRow(c, state, itemPath, index, item)
		}
	})
}

func (a *application) configSectionItemRow(c *ui.Context, state *configSectionState, itemPath configItemPath,
	index int, item map[string]any) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, configItemLabel(item)).FontSize(12).Bold()
			ui.Text(c, configItemDetail(item)).FontSize(11).TextColor(c.Theme().TextMuted)
		})
		if ui.Button(c, "Edit").Key(configItemKey(state, itemPath.Path, "edit", index)).Clicked() {
			a.openConfigSectionItemEditor(state, itemPath, index)
		}
		if ui.Button(c, "Delete").Key(configItemKey(state, itemPath.Path, "delete", index)).Clicked() {
			a.askDeleteConfigSectionItem(state, itemPath.Path, index, configItemLabel(item))
		}
	})
}

func (a *application) openConfigSectionItemEditor(state *configSectionState, itemPath configItemPath, index int) {
	items, ok := configSectionObjectItems(state, itemPath.Path)
	if !ok {
		return
	}
	state.itemPath = itemPath.Path
	state.itemEditIndex = index
	state.itemErr = ""
	if index >= 0 && index < len(items) {
		body, err := json.MarshalIndent(items[index], "", "  ")
		if err != nil {
			a.model.setActionError(err)
			a.invalidate()
			return
		}
		state.itemDraft = string(body) + "\n"
	} else {
		state.itemDraft = itemPath.Default
	}
	state.itemEditOpen = true
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
		title := "Add " + configItemDialogLabel(state, state.itemPath)
		if state.itemEditIndex >= 0 {
			title = "Edit " + configItemDialogLabel(state, state.itemPath)
		}
		dialogPanel(c, title, "Edit this item as native JSON. Unknown fields are preserved when saved.", func() {
			ui.TextArea(c.Key("config-item-editor-"+state.spec.Key), &state.itemDraft).
				Height(configItemEditorHeight).AutoFocus()
			if state.itemErr != "" {
				ui.Text(c, state.itemErr).FontSize(12).TextColor(c.Theme().Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Key(configItemKey(state, state.itemPath, "cancel", -1)).Clicked() {
					state.itemEditOpen = false
				}
				if ui.PrimaryButton(c, "Save item").Key(configItemKey(state, state.itemPath, "save", -1)).Clicked() {
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
	items, ok := configSectionObjectItems(state, state.itemPath)
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
	if err := state.replaceConfigItems(state.itemPath, items); err != nil {
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

func (a *application) askDeleteConfigSectionItem(state *configSectionState, path string, index int, label string) {
	a.deleteForm = deleteFormState{
		open: true, kind: "config-section-item", id: state.spec.Key,
		ids: []string{strconv.Itoa(index), path}, title: "Delete configuration item",
		message: "Remove " + label + " from " + state.spec.Title + "? The change is only saved when you Apply the section.",
		confirm: "Delete",
	}
}

func (a *application) deleteConfigSectionItem(section string, indexText, path string) {
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
	items, ok := configSectionObjectItems(state, path)
	if !ok || index < 0 || index >= len(items) {
		a.model.setActionError(errors.New("configuration item does not exist"))
		a.invalidate()
		return
	}
	items = append(items[:index], items[index+1:]...)
	if err := state.replaceConfigItems(path, items); err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.model.actionError = ""
	a.model.notice = state.spec.Title + " item removed from the draft"
	a.invalidate()
}

func configSectionObjectItems(state *configSectionState, path string) ([]map[string]any, bool) {
	if state == nil || !state.loaded {
		return nil, false
	}
	value, err := decodeJSONValue(state.editorState().Buffer.String())
	if err != nil {
		return nil, false
	}
	if path != "" {
		root, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value = jsonPathValue(root, path)
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

func (s *configSectionState) replaceConfigItems(path string, items []map[string]any) error {
	if path == "" {
		body, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
		s.editorState().Buffer.Set(string(body) + "\n")
		return nil
	}
	root, ok := decodeSectionObject(s.editorState().Buffer.String())
	if !ok || root == nil {
		return errors.New("current section is not a JSON object")
	}
	setJSONPath(root, path, items)
	body, err := json.MarshalIndent(root, "", "  ")
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
	return item, nil
}
