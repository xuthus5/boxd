package main

import (
	"context"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) openAddSubscription() {
	if a.backend == nil || a.model.busy {
		return
	}
	a.subForm.reset()
}

func (a *application) openEditSubscription(sub model.Subscription) {
	if a.backend == nil || a.model.busy {
		return
	}
	a.subForm.beginEdit(sub)
}

func (a *application) submitSubscriptionForm() {
	if a.backend == nil || a.model.busy {
		return
	}
	input, err := a.subForm.input()
	if err != nil {
		a.subForm.err = shortError(err)
		return
	}
	a.subForm.err = ""
	if a.subForm.edit {
		a.runActionWithCallback("Updating subscription", func(ctx context.Context) error {
			return a.backend.UpdateSubscription(ctx, a.subForm.id, input)
		}, "Subscription updated", func() { a.subForm.open = false })
		return
	}
	a.runActionWithCallback("Creating subscription", func(ctx context.Context) error {
		_, err := a.backend.CreateSubscription(ctx, input)
		return err
	}, "Subscription created", func() { a.subForm.open = false })
}

func (a *application) askDeleteSubscription(sub model.Subscription) {
	if a.backend == nil || a.model.busy {
		return
	}
	a.deleteForm = deleteFormState{
		open: true, kind: "subscription", id: sub.ID, title: "Delete subscription “" + sub.Name + "”?",
		message: "Its generated nodes and managed outbound will be removed from the active configuration.", confirm: "Delete",
	}
}

func (a *application) deleteSubscription(id string) {
	a.runActionWithCallback("Deleting subscription", func(ctx context.Context) error {
		return a.backend.DeleteSubscription(ctx, id)
	}, "Subscription deleted", func() { a.requestRefresh() })
}

func (a *application) refreshAllSubscriptions() {
	if a.backend == nil || a.model.busy {
		return
	}
	a.runAction("Refreshing subscriptions", func(ctx context.Context) error {
		_, err := a.backend.RefreshAllSubscriptions(ctx)
		return err
	}, "Subscriptions refreshed")
}

func (a *application) openSubscriptionSelector(sub model.Subscription) {
	if a.backend == nil || a.model.busy || len(sub.Outbounds) == 0 {
		return
	}
	options := make([]string, 0, len(sub.Outbounds))
	for _, outbound := range sub.Outbounds {
		options = append(options, outbound.Tag)
	}
	selected := sub.Selected
	if selected == "" {
		selected = options[0]
	}
	a.selectorForm = selectorFormState{
		open: true, subscriptionID: sub.ID, name: sub.Name, selected: selected, options: options,
	}
}

func (a *application) submitSubscriptionSelector() {
	tag := strings.TrimSpace(a.selectorForm.selected)
	if tag == "" {
		return
	}
	a.runActionWithCallback("Selecting exit", func(ctx context.Context) error {
		return a.backend.SelectSubscription(ctx, a.selectorForm.subscriptionID, tag)
	}, "Subscription exit updated", func() { a.selectorForm.open = false })
}
