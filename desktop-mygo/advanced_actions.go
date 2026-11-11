package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) ensureConfigHistory() {
	if a.configHistory.attempted {
		return
	}
	a.loadConfigHistory(false)
}

func (a *application) refreshConfigHistory() {
	a.loadConfigHistory(true)
}

func (a *application) loadConfigHistory(force bool) {
	if a.backend == nil || a.configHistory.loading || (a.configHistory.attempted && !force) {
		return
	}
	a.configHistory.loading = true
	a.configHistory.attempted = true
	if a.win == nil {
		events, err := a.fetchConfigHistory()
		a.applyConfigHistory(events, err)
		return
	}
	go func() {
		events, err := a.fetchConfigHistory()
		a.win.Update(func() { a.applyConfigHistory(events, err) })
	}()
}

func (a *application) fetchConfigHistory() ([]model.ConfigApplyEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.ConfigApplyHistory(ctx)
}

func (a *application) applyConfigHistory(events []model.ConfigApplyEvent, err error) {
	a.configHistory.loading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.configHistory.events = events
	a.configHistory.loaded = true
}

func (a *application) openConfigSnapshot(event model.ConfigApplyEvent) {
	if a.backend == nil || a.configHistory.previewLoad {
		return
	}
	a.configHistory.previewID = event.ID
	a.configHistory.previewSource = event.Source
	a.configHistory.previewLoad = true
	if a.win == nil {
		text, err := a.fetchConfigSnapshot(event.ID)
		a.applyConfigSnapshot(text, err)
		return
	}
	go func() {
		text, err := a.fetchConfigSnapshot(event.ID)
		a.win.Update(func() { a.applyConfigSnapshot(text, err) })
	}()
}

func (a *application) fetchConfigSnapshot(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.ConfigSnapshot(ctx, id)
}

func (a *application) applyConfigSnapshot(text string, err error) {
	a.configHistory.previewLoad = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.configHistory.previewState().Buffer.Reset(text)
	a.configHistory.previewState().Buffer.MarkSaved()
	a.configHistory.previewOpen = true
	a.model.actionError = ""
}

func (a *application) askRestoreConfigSnapshot(id string) {
	if id == "" || a.model.busy {
		return
	}
	a.configHistory.previewOpen = false
	a.deleteForm = deleteFormState{
		open: true, kind: "config-restore", id: id,
		title:   "Restore this configuration?",
		message: "The snapshot will be validated, applied and the kernel may restart. The current configuration is saved in history first.",
		confirm: "Restore",
	}
}

func (a *application) restoreConfigSnapshot(id string) {
	a.runActionWithCallback("Restoring configuration", func(ctx context.Context) error {
		return a.backend.RestoreConfigSnapshot(ctx, id)
	}, "Configuration restored", func() {
		a.configHistory.previewOpen = false
		a.loadConfig()
		a.loadConfigHistory(true)
		a.loadConfigDiagnostics()
		a.requestRefresh()
	})
}

func (a *application) ensureRuleSets() {
	if a.ruleSets.attempted {
		return
	}
	a.loadRuleSets(false)
}

func (a *application) refreshRuleSets() {
	a.loadRuleSets(true)
}

func (a *application) loadRuleSets(force bool) {
	if a.backend == nil || a.ruleSets.loading || (a.ruleSets.attempted && !force) {
		return
	}
	a.ruleSets.loading = true
	a.ruleSets.attempted = true
	if a.win == nil {
		items, auto, err := a.fetchRuleSets()
		a.applyRuleSets(items, auto, err)
		return
	}
	go func() {
		items, auto, err := a.fetchRuleSets()
		a.win.Update(func() { a.applyRuleSets(items, auto, err) })
	}()
}

func (a *application) fetchRuleSets() ([]model.RuleSetStatusItem, model.RuleSetAutoUpdate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	items, err := a.backend.RuleSetStatus(ctx)
	if err != nil {
		return nil, model.RuleSetAutoUpdate{}, err
	}
	auto, err := a.backend.RuleSetAutoUpdate(ctx)
	return items, auto, err
}

func (a *application) applyRuleSets(items []model.RuleSetStatusItem, auto model.RuleSetAutoUpdate, err error) {
	a.ruleSets.loading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.ruleSets.items = items
	a.ruleSets.loaded = true
	if !a.ruleSets.autoLoaded {
		a.ruleSets.savedAuto = auto
		a.ruleSets.draftAuto = auto
		a.ruleSets.autoLoaded = true
	}
}

func (a *application) updateRuleSets(tags []string) {
	req := core.RuleSetUpdateRequest{Tags: tags}
	label := "Updating all rule sets"
	if len(tags) == 1 {
		label = "Updating rule set " + tags[0]
	}
	a.runActionWithCallback(label, func(ctx context.Context) error {
		_, err := a.backend.UpdateRuleSets(ctx, req)
		return err
	}, "Rule sets updated", func() { a.loadRuleSets(true) })
}

func (a *application) setRuleSetAutoEnabled(enabled bool) {
	if a.model.busy {
		return
	}
	previous := a.ruleSets.savedAuto
	next := a.ruleSets.draftAuto
	next.Enabled = enabled
	a.ruleSets.draftAuto = next
	a.runActionWithCallbacks("Saving rule-set auto update", func(ctx context.Context) error {
		return a.backend.SetRuleSetAutoUpdate(ctx, next)
	}, "Rule-set auto update saved", func() {
		a.ruleSets.draftAuto = previous
	}, func() {
		a.ruleSets.savedAuto = next
	})
}

func (a *application) saveRuleSetAutoInterval(interval string) {
	if a.model.busy {
		return
	}
	previous := a.ruleSets.savedAuto
	next := a.ruleSets.draftAuto
	next.Interval = strings.TrimSpace(interval)
	a.ruleSets.draftAuto = next
	a.runActionWithCallbacks("Saving rule-set update interval", func(ctx context.Context) error {
		return a.backend.SetRuleSetAutoUpdate(ctx, next)
	}, "Rule-set update interval saved", func() {
		a.ruleSets.draftAuto = previous
	}, func() {
		a.ruleSets.savedAuto = next
	})
}

func (a *application) savePassword() {
	form := &a.securityForm
	if form.next != form.confirm {
		a.model.setActionError(errors.New("new password and confirmation do not match"))
		a.invalidate()
		return
	}
	if strings.TrimSpace(form.next) == "" {
		a.model.setActionError(errors.New("new password is required"))
		a.invalidate()
		return
	}
	current, next := form.current, form.next
	a.runActionWithCallback("Changing password", func(ctx context.Context) error {
		return a.backend.ChangePassword(ctx, current, next)
	}, "Password changed", func() {
		a.securityForm = securityFormState{}
	})
}

func (a *application) exportBackup() {
	if a.backend == nil || a.win == nil || a.model.busy {
		return
	}
	go func() {
		defaultName := "boxd-backup-" + time.Now().UTC().Format("20060102T150405Z") + ".tar.gz"
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: "Export boxd backup", DefaultPath: defaultName,
			Filters: []mygo.FileFilter{{Name: "Backup archive", Extensions: []string{"tar.gz", "tgz"}}},
		})
		if err == nil && path != "" {
			ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
			err = a.backend.CreateBackup(ctx, path)
			cancel()
		}
		a.win.Update(func() {
			if err != nil {
				a.model.setActionError(err)
				return
			}
			if path != "" {
				a.model.actionError = ""
				a.model.notice = "Backup ready: " + path
			}
		})
	}()
}
