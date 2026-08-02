package main

import (
	"context"
	"errors"
)

// DialogService 提供原生文件对话框能力（打开/保存文件）。
// 通过 Wails Dialog 管理器实现，GTK4 走 xdg-desktop-portal。
type DialogService struct{}

// NewDialogService 构造文件对话框服务。
func NewDialogService() *DialogService {
	return &DialogService{}
}

// OpenConfig 打开文件选择器选取 sing-box 配置文件。
func (s *DialogService) OpenConfig(_ context.Context) (string, error) {
	return s.openFile("Select sing-box config", "JSON files (*.json)", "*.json")
}

// OpenJSON 打开文件选择器选取任意 JSON 文件（导入用）。
func (s *DialogService) OpenJSON(_ context.Context) (string, error) {
	return s.openFile("Select JSON file", "JSON files (*.json)", "*.json")
}

// SaveJSON 打开保存对话框导出 JSON 文件。
func (s *DialogService) SaveJSON(_ context.Context, filename string) (string, error) {
	return s.saveFile("Save JSON file", filename, "JSON files (*.json)", "*.json")
}

// SaveBackup 打开保存对话框导出备份归档。
func (s *DialogService) SaveBackup(_ context.Context, filename string) (string, error) {
	return s.saveFile("Save backup archive", filename, "GZip archives (*.tar.gz)", "*.tar.gz")
}

// openFile 打开文件选择器。
func (s *DialogService) openFile(title, filterName, filterPattern string) (string, error) {
	if globalApp == nil || globalApp.Dialog == nil {
		return "", errors.New("file dialog is not available")
	}
	dialog := globalApp.Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(true).
		CanChooseDirectories(false)
	if filterName != "" {
		dialog = dialog.AddFilter(filterName, filterPattern)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// saveFile 打开保存对话框。
func (s *DialogService) saveFile(title, filename, filterName, filterPattern string) (string, error) {
	if globalApp == nil || globalApp.Dialog == nil {
		return "", errors.New("file dialog is not available")
	}
	saveDialog := globalApp.Dialog.SaveFile().
		SetMessage(title)
	if filterName != "" {
		saveDialog = saveDialog.AddFilter(filterName, filterPattern)
	}
	if filename != "" {
		saveDialog = saveDialog.SetDirectory(filename)
	}
	path, err := saveDialog.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}
