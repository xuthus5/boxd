package main

import "github.com/egoist/mygo/ui"

func (a *application) selectorDialog(c *ui.Context) {
	ui.Modal(c, &a.selectorForm.open, func() {
		dialogPanel(c, "Select subscription exit", "Choose the outbound used by "+a.selectorForm.name+" in selector mode.", func() {
			ui.Field(c, "Outbound", func() {
				ui.Select(c.Key("selector-outbound"), &a.selectorForm.selected, a.selectorForm.options)
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Cancel").Key("selector-cancel").Clicked() {
					a.selectorForm.open = false
				}
				if ui.PrimaryButton(c, "Apply").Key("selector-save").Disabled(a.model.busy).Clicked() {
					a.submitSubscriptionSelector()
				}
			})
		})
	})
}

func (a *application) deleteDialog(c *ui.Context) {
	if !a.deleteForm.open {
		return
	}
	confirm := a.deleteForm.confirm
	if confirm == "" {
		confirm = "Delete"
	}
	if ui.AlertDialog(c, &a.deleteForm.open, a.deleteForm.title, a.deleteForm.message, "Cancel", confirm) == 1 {
		a.confirmDelete()
	}
}
