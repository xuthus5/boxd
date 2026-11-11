package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	configDiffDisplayLimit = 20
	configDiffValueLimit   = 96
)

type configDiffKind string

const (
	configDiffAdded   configDiffKind = "added"
	configDiffRemoved configDiffKind = "removed"
	configDiffChanged configDiffKind = "changed"
)

type configDiffItem struct {
	Path   string
	Kind   configDiffKind
	Before string
	After  string
}

func (a *application) configDiffCard(c *ui.Context) {
	renderConfigDiffCard(c, "Configuration diff", a.config.baseline,
		a.configEditorState().Buffer.String(), a.config.loaded)
}

func renderConfigDiffCard(c *ui.Context, title, baseline, draft string, loaded bool) {
	card(c, func() {
		sectionHeading(c, title, "Semantic changes compared with the loaded configuration")
		if !loaded {
			ui.Text(c, "Load a configuration to inspect changes.").FontSize(12).TextColor(c.Theme().TextMuted)
			return
		}
		items, err := compareConfigJSON(baseline, draft)
		if err != nil {
			ui.Text(c, err.Error()).FontSize(12).TextColor(c.Theme().Danger)
			return
		}
		if len(items) == 0 {
			ui.Text(c, "No changes compared with the loaded configuration.").FontSize(12).TextColor(c.Theme().Success)
			return
		}
		ui.Textf(c, "%d changed path(s)", len(items)).FontSize(12).TextColor(c.Theme().TextMuted)
		for index, item := range items {
			if index >= configDiffDisplayLimit {
				ui.Textf(c, "…and %d more changed paths", len(items)-index).
					FontSize(11).TextColor(c.Theme().TextMuted)
				break
			}
			configDiffRow(c, item)
		}
	})
}

func configDiffRow(c *ui.Context, item configDiffItem) {
	color := c.Theme().Warning
	if item.Kind == configDiffAdded {
		color = c.Theme().Success
	}
	if item.Kind == configDiffRemoved {
		color = c.Theme().Danger
	}
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Badge(c, strings.ToUpper(string(item.Kind))).Background(color.Alpha(0.15)).TextColor(color)
		ui.Text(c, item.Path).FontSize(12).Bold().Grow(1)
		ui.Text(c, configDiffValueLabel(item)).FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func configDiffValueLabel(item configDiffItem) string {
	switch item.Kind {
	case configDiffAdded:
		return "+ " + item.After
	case configDiffRemoved:
		return "- " + item.Before
	default:
		return item.Before + " → " + item.After
	}
}

func compareConfigJSON(before, after string) ([]configDiffItem, error) {
	left, err := decodeJSONValue(before)
	if err != nil {
		return nil, fmt.Errorf("loaded configuration is invalid: %w", err)
	}
	right, err := decodeJSONValue(after)
	if err != nil {
		return nil, errors.New("current configuration is not valid JSON")
	}
	items := make([]configDiffItem, 0)
	diffJSONValue("", left, right, &items)
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items, nil
}

func diffJSONValue(path string, before, after any, items *[]configDiffItem) {
	left, leftIsMap := before.(map[string]any)
	right, rightIsMap := after.(map[string]any)
	if leftIsMap && rightIsMap {
		diffJSONMap(path, left, right, items)
		return
	}
	leftArray, leftIsArray := before.([]any)
	rightArray, rightIsArray := after.([]any)
	if leftIsArray && rightIsArray {
		diffJSONArray(path, leftArray, rightArray, items)
		return
	}
	if jsonValueEqual(before, after) {
		return
	}
	*items = append(*items, configDiffItem{Path: firstText(path, "$"), Kind: configDiffChanged,
		Before: diffValue(before), After: diffValue(after)})
}

func diffJSONMap(path string, before, after map[string]any, items *[]configDiffItem) {
	for _, key := range sortedJSONKeys(before, after) {
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		leftValue, leftOK := before[key]
		rightValue, rightOK := after[key]
		switch {
		case !leftOK:
			*items = append(*items, configDiffItem{Path: childPath, Kind: configDiffAdded, After: diffValue(rightValue)})
		case !rightOK:
			*items = append(*items, configDiffItem{Path: childPath, Kind: configDiffRemoved, Before: diffValue(leftValue)})
		default:
			diffJSONValue(childPath, leftValue, rightValue, items)
		}
	}
}

func diffJSONArray(path string, before, after []any, items *[]configDiffItem) {
	limit := len(before)
	if len(after) > limit {
		limit = len(after)
	}
	for index := 0; index < limit; index++ {
		childPath := fmt.Sprintf("%s[%d]", firstText(path, "$"), index)
		if index >= len(before) {
			*items = append(*items, configDiffItem{Path: childPath, Kind: configDiffAdded, After: diffValue(after[index])})
			continue
		}
		if index >= len(after) {
			*items = append(*items, configDiffItem{Path: childPath, Kind: configDiffRemoved, Before: diffValue(before[index])})
			continue
		}
		diffJSONValue(childPath, before[index], after[index], items)
	}
}

func sortedJSONKeys(left, right map[string]any) []string {
	keys := make([]string, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for _, source := range []map[string]any{left, right} {
		for key := range source {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func jsonValueEqual(left, right any) bool {
	leftBody, leftErr := json.Marshal(left)
	rightBody, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftBody) == string(rightBody)
}

func diffValue(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "unknown"
	}
	return truncateText(string(body), configDiffValueLimit)
}
