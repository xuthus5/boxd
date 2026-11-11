package main

import "github.com/egoist/mygo/ui"

func (a *application) nodeFormDialog(c *ui.Context) {
	ui.Modal(c, &a.nodeForm.open, func() {
		title := "Import proxy link"
		subtitle := "Paste a supported proxy URL. The node is saved and the managed outbound is synchronized."
		if a.nodeForm.edit {
			title = "Edit imported node"
			subtitle = "Update the managed outbound. The kernel is reloaded when the synchronized configuration changes."
		}
		dialogPanel(c, title, subtitle, func() {
			if a.nodeForm.edit {
				a.nodeEditFields(c)
			} else {
				ui.Field(c, "Proxy link", func() {
					ui.TextInput(c.Key("node-link"), &a.nodeForm.link).AutoFocus().Placeholder("vless://…")
				})
			}
			if a.nodeForm.err != "" {
				ui.Text(c, a.nodeForm.err).FontSize(12).TextColor(c.Theme().Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Key("node-cancel").Clicked() {
					a.nodeForm.open = false
				}
				label := "Import"
				if a.nodeForm.edit {
					label = "Save"
				}
				if ui.PrimaryButton(c, label).Key("node-save").Disabled(a.model.busy).Clicked() {
					a.submitNodeForm()
				}
			})
		})
	})
}

func (a *application) nodeEditFields(c *ui.Context) {
	ui.Row(c).Gap(10).Children(func() {
		ui.Field(c, "Tag", func() {
			ui.TextInput(c.Key("node-tag"), &a.nodeForm.tag).AutoFocus()
		}).Grow(1)
		ui.Field(c, "Type", func() {
			ui.TextInput(c.Key("node-type"), &a.nodeForm.typ).Placeholder("vless")
		}).Grow(1)
	})
	ui.Row(c).Gap(10).Children(func() {
		ui.Field(c, "Server", func() {
			ui.TextInput(c.Key("node-server"), &a.nodeForm.server).Placeholder("example.com")
		}).Grow(1)
		ui.Field(c, "Port", func() {
			ui.NumberInput(c.Key("node-port"), &a.nodeForm.port, 1, 65535, 1)
		}).Width(130)
	})
	ui.Field(c, "Outbound JSON", func() {
		ui.TextArea(c.Key("node-config"), &a.nodeForm.config).Height(220)
	})
}
