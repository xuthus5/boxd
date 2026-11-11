package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xuthus5/boxd/internal/model"
)

const configEditorHeight = 520

type configPageState struct {
	editor             *editor.State
	baseline           string
	loaded             bool
	loading            bool
	attempted          bool
	diagnostics        model.ConfigDiagnostics
	diagnosticsLoaded  bool
	diagnosticsLoading bool
	diagnosticsTried   bool
}

func (a *application) configView(c *ui.Context) {
	if !a.config.loaded && !a.config.loading && !a.config.attempted {
		a.loadConfig()
	}
	a.pageColumn(c, func() {
		pageHeading(c, "Config", "Inspect, validate and apply the shared sing-box configuration.")
		card(c, func() {
			a.configToolbar(c)
			ui.Divider(c)
			if a.config.loading {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "Loading configuration…").TextColor(c.Theme().TextMuted)
				})
				return
			}
			editor.View(c, a.configEditorState()).Key("config-editor").
				FillWidth().Height(configEditorHeight)
		})
		a.configDiagnosticsCard(c)
		a.configDiffCard(c)
	})
}

func (a *application) configToolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
		sectionHeading(c, "Raw configuration", "JSON is validated before it is written and applied")
		ui.Spacer(c)
		dirty := a.configEditorState().Buffer.Modified()
		if dirty {
			ui.Badge(c, "Unsaved").Background(t.Warning.Alpha(0.2)).TextColor(t.Warning)
		}
		if ui.Button(c, "Reset").Key("config-reset").Disabled(!a.configCanReset()).Clicked() {
			a.resetConfigDraft()
		}
		if ui.Button(c, "Import").Key("config-import").Disabled(a.config.loading || a.model.busy).Clicked() {
			a.importConfig()
		}
		if ui.Button(c, "Export").Key("config-export").Clicked() {
			a.exportConfig()
		}
		if ui.Button(c, "Reload").Key("config-reload").Disabled(a.config.loading || a.model.busy).Clicked() {
			a.loadConfig()
		}
		if ui.Button(c, "Format").Key("config-format").Disabled(a.config.loading).Clicked() {
			a.formatConfig()
		}
		if ui.Button(c, "Validate").Key("config-validate").Disabled(a.config.loading || a.model.busy).Clicked() {
			a.validateConfig()
		}
		if ui.PrimaryButton(c, "Apply").Key("config-apply").Disabled(a.config.loading || a.model.busy).Clicked() {
			a.applyConfig()
		}
	})
}

func (a *application) configCanReset() bool {
	return a.config.loaded && a.config.baseline != "" && a.configEditorState().Buffer.String() != a.config.baseline
}

func (a *application) configEditorState() *editor.State {
	if a.config.editor == nil {
		buffer := editor.NewBuffer("")
		buffer.SetLanguage(editor.LanguageFor("config.json"))
		a.config.editor = &editor.State{Buffer: buffer}
	}
	return a.config.editor
}

func formatConfigJSON(input string) (string, error) {
	var value any
	if !json.Valid([]byte(input)) {
		return "", errors.New("invalid JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return body.String(), nil
}
