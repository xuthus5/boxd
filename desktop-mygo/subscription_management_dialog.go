package main

import "github.com/egoist/mygo/ui"

func (a *application) subscriptionFormDialog(c *ui.Context) {
	ui.Modal(c, &a.subForm.open, func() {
		title := "Add subscription"
		if a.subForm.edit {
			title = "Edit subscription"
		}
		dialogPanel(c, title, "Subscription changes are validated, saved and synchronized into managed outbounds.", func() {
			ui.Field(c, "Name", func() {
				ui.TextInput(c.Key("sub-name"), &a.subForm.name).AutoFocus()
			})
			ui.Field(c, "Subscription URL", func() {
				ui.TextInput(c.Key("sub-url"), &a.subForm.url).Placeholder("https://example.com/subscription")
			})
			ui.Row(c).Gap(14).AlignItems(ui.End).Children(func() {
				ui.Field(c, "Refresh interval (minutes)", func() {
					ui.NumberInput(c.Key("sub-interval"), &a.subForm.interval, 1, 10080, 1)
				}).Width(220)
				ui.Field(c, "URL-test policy", func() {
					ui.Select(c.Key("sub-urltest-policy"), &a.subForm.urlTestMode,
						[]string{urlTestModeInherit, urlTestModeEnabled, urlTestModeDisabled})
				})
			})
			if a.subForm.urlTestMode == urlTestModeEnabled {
				a.subscriptionURLTestFields(c)
			}
			if a.subForm.err != "" {
				ui.Text(c, a.subForm.err).FontSize(12).TextColor(c.Theme().Danger)
			}
			a.subscriptionDialogButtons(c)
		})
	})
}

func (a *application) subscriptionURLTestFields(c *ui.Context) {
	ui.Row(c).Gap(10).Children(func() {
		ui.Field(c, "Test URL", func() {
			ui.TextInput(c.Key("sub-test-url"), &a.subForm.testURL).Placeholder("https://www.gstatic.com/generate_204")
		}).Grow(1)
		ui.Field(c, "Interval", func() {
			ui.TextInput(c.Key("sub-test-interval"), &a.subForm.testInterval).Placeholder("3m")
		}).Width(120)
		ui.Field(c, "Tolerance", func() {
			ui.NumberInput(c.Key("sub-tolerance"), &a.subForm.tolerance, 0, 65535, 1)
		}).Width(130)
	})
}

func (a *application) subscriptionDialogButtons(c *ui.Context) {
	ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
		if ui.Button(c, "Cancel").Key("sub-cancel").Clicked() {
			a.subForm.open = false
		}
		label := "Create"
		if a.subForm.edit {
			label = "Save"
		}
		if ui.PrimaryButton(c, label).Key("sub-save").Disabled(a.model.busy).Clicked() {
			a.submitSubscriptionForm()
		}
	})
}
