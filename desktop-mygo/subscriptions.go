package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) subscriptionsView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Subscriptions", "Create, edit, refresh and select subscription exits without leaving the native app.")
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c, "Add subscription").Key("sub-add").Disabled(a.model.busy).Clicked() {
				a.openAddSubscription()
			}
			if ui.Button(c, "Refresh all").Key("sub-refresh-all").Disabled(a.model.busy || len(a.model.subs) == 0).Clicked() {
				a.refreshAllSubscriptions()
			}
		})
		ui.Row(c).Gap(12).Children(func() {
			metricCard(c, "Subscriptions", fmt.Sprint(len(a.model.subs)), "Configured in the shared database", c.Theme().Accent)
			metricCard(c, "Healthy", fmt.Sprint(countHealthySubs(a.model.subs)), "No recorded refresh error", c.Theme().Success)
			metricCard(c, "Errors", fmt.Sprint(countFailedSubs(a.model.subs)), "Need attention", c.Theme().Danger)
		})
		if len(a.model.subs) == 0 {
			card(c, func() {
				emptyState(c, "No subscriptions", "Add a subscription to import its nodes into the managed outbound configuration.")
				if ui.PrimaryButton(c, "Add subscription").Key("sub-empty-add").Clicked() {
					a.openAddSubscription()
				}
			})
			return
		}
		a.subscriptionsTable(c)
	})
}

func (a *application) subscriptionsTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Name", MinWidth: 190},
		{Title: "Status", Width: 100},
		{Title: "Traffic", Width: 150},
		{Title: "Last updated", Width: 135},
		{Title: "Exit", Width: 130},
		{Title: "Actions", Width: 260},
	}
	ui.Table(c, &a.subsList, columns, len(a.model.subs), func(row, col int) {
		if row < 0 || row >= len(a.model.subs) {
			return
		}
		sub := a.model.subs[row]
		switch col {
		case 0:
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, sub.Name).NoWrap().Bold()
				ui.Text(c, truncateText(sub.URL, 56)).FontSize(11).TextColor(c.Theme().TextMuted).NoWrap()
			})
		case 1:
			ui.Text(c, subscriptionStatus(sub)).NoWrap().TextColor(subscriptionStatusColor(c, sub))
		case 2:
			ui.Text(c, subscriptionTraffic(sub)).NoWrap()
		case 3:
			ui.Text(c, formatRelative(sub.LastUpdated, time.Now())).NoWrap().TextColor(c.Theme().TextMuted)
		case 4:
			ui.Text(c, firstText(sub.Selected, "Auto")).NoWrap().TextColor(c.Theme().TextMuted)
		case 5:
			a.subscriptionRowActions(c, sub)
		}
	}).FillWidth().Height(500).Label("Subscription list")
}

func (a *application) subscriptionRowActions(c *ui.Context, sub model.Subscription) {
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, "Refresh").Key("refresh-sub-" + sub.ID).Disabled(a.model.busy).Clicked() {
			a.refreshSubscription(sub.ID)
		}
		if ui.Button(c, "Select").Key("select-sub-" + sub.ID).
			Disabled(a.model.busy || len(sub.Outbounds) == 0).Clicked() {
			a.openSubscriptionSelector(sub)
		}
		if ui.Button(c, "Edit").Key("edit-sub-" + sub.ID).Disabled(a.model.busy).Clicked() {
			a.openEditSubscription(sub)
		}
		if ui.Button(c, "Delete").Key("delete-sub-" + sub.ID).Disabled(a.model.busy).Clicked() {
			a.askDeleteSubscription(sub)
		}
	})
}

func subscriptionStatus(sub model.Subscription) string {
	if sub.Error != "" {
		return "Error"
	}
	if sub.LastUpdated.IsZero() {
		return "Pending"
	}
	return "Healthy"
}

func subscriptionStatusColor(c *ui.Context, sub model.Subscription) ui.Color {
	switch subscriptionStatus(sub) {
	case "Error":
		return c.Theme().Danger
	case "Pending":
		return c.Theme().Warning
	default:
		return c.Theme().Success
	}
}

func subscriptionTraffic(sub model.Subscription) string {
	if sub.Traffic == nil {
		return "—"
	}
	used := sub.Traffic.Upload + sub.Traffic.Download
	if sub.Traffic.Total > 0 {
		return formatBytes(used) + " / " + formatBytes(sub.Traffic.Total)
	}
	return formatBytes(used)
}

func countHealthySubs(subs []model.Subscription) int {
	count := 0
	for _, sub := range subs {
		if subscriptionStatus(sub) == "Healthy" {
			count++
		}
	}
	return count
}

func countFailedSubs(subs []model.Subscription) int {
	count := 0
	for _, sub := range subs {
		if subscriptionStatus(sub) == "Error" {
			count++
		}
	}
	return count
}
