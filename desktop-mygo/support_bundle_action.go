package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
)

func (a *application) exportSupportBundle() {
	if a.backend == nil || a.win == nil || !a.beginAction("Preparing support bundle") {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		collection, err := a.backend.SupportBundle(ctx)
		cancel()
		path := ""
		unavailable := 0
		if err == nil {
			bundle := buildSupportBundle(collection, time.Now().UTC())
			unavailable = countUnavailableSupportBundleSources(bundle.Requests)
			var text string
			text, err = formatSupportBundle(bundle)
			if err == nil {
				path, err = mygo.Dialog.Save(mygo.SaveDialogOptions{
					Parent: a.win, Title: "Export boxd support bundle",
					DefaultPath: supportBundleFilename(time.Now()),
					Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
				})
			}
			if err == nil && path != "" {
				err = writeConfigFile(path, []byte(text))
			}
		}
		a.reportSupportBundleResult(path, unavailable, err)
	}()
}

func (a *application) reportSupportBundleResult(path string, unavailable int, err error) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		a.model.busy = false
		a.model.busyLabel = ""
		if err != nil {
			a.model.setActionError(fmt.Errorf("support bundle export failed: %w", err))
			return
		}
		if path == "" {
			return
		}
		a.model.actionError = ""
		if unavailable > 0 {
			a.model.notice = fmt.Sprintf("Support bundle saved to %s (%d sources unavailable)", path, unavailable)
			return
		}
		a.model.notice = "Support bundle saved to " + path
	})
}
