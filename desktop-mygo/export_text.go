package main

import (
	"time"

	"github.com/egoist/mygo"
)

func (a *application) exportTextFile(title, defaultName, extension, text string) {
	if a.win == nil {
		return
	}
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: title, DefaultPath: defaultName,
			Filters: []mygo.FileFilter{{Name: extension + " file", Extensions: []string{extension}}},
		})
		if err == nil && path != "" {
			err = writeConfigFile(path, []byte(text))
		}
		a.win.Update(func() {
			if err != nil {
				a.model.setActionError(err)
				return
			}
			if path != "" {
				a.model.actionError = ""
				a.model.notice = defaultName + " exported to " + path
			}
		})
	}()
}

func exportFilename(prefix, extension string) string {
	return prefix + "-" + time.Now().UTC().Format("20060102T150405Z") + "." + extension
}
