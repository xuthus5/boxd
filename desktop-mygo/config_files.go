package main

import (
	"errors"
	"io"
	"os"

	"github.com/egoist/mygo"
)

const maxConfigImportBytes = 16 << 20

func readConfigFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(body)) > limit {
		return nil, errors.New("configuration file is too large")
	}
	return body, nil
}

func writeConfigFile(path string, body []byte) error {
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (a *application) importConfig() {
	if a.win == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.win, Title: "Import configuration",
			Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err != nil || len(paths) == 0 {
			a.reportConfigFileResult("", err)
			return
		}
		path := paths[0]
		body, err := readConfigFile(path, maxConfigImportBytes)
		a.win.Update(func() {
			a.config.loading = false
			if err != nil {
				a.model.setActionError(err)
				return
			}
			a.configEditorState().Buffer.Reset(string(body))
			a.config.loaded = true
			a.config.attempted = true
			a.model.actionError = ""
			a.model.notice = "Configuration imported; review and apply"
		})
	}()
}

func (a *application) exportConfig() {
	if a.win == nil {
		return
	}
	body := []byte(a.configEditorState().Buffer.String())
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: "Export configuration", DefaultPath: "config.json",
			Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		if err == nil && path != "" {
			err = writeConfigFile(path, body)
		}
		a.reportConfigFileResult(path, err)
	}()
}

func (a *application) reportConfigFileResult(path string, err error) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		if err != nil {
			a.model.setActionError(err)
			return
		}
		if path == "" {
			return
		}
		a.model.actionError = ""
		a.model.notice = "Configuration file ready: " + path
	})
}
