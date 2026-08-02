package main

import (
	"context"
	"testing"
)

func TestDialogServiceUnavailable(t *testing.T) {
	old := globalApp
	t.Cleanup(func() { globalApp = old })
	globalApp = nil

	svc := NewDialogService()
	if _, err := svc.OpenConfig(context.Background()); err == nil {
		t.Fatal("expected error when app is nil")
	}
	if _, err := svc.OpenJSON(context.Background()); err == nil {
		t.Fatal("expected error when app is nil")
	}
	if _, err := svc.SaveJSON(context.Background(), "x.json"); err == nil {
		t.Fatal("expected error when app is nil")
	}
	if _, err := svc.SaveBackup(context.Background(), "x.tar.gz"); err == nil {
		t.Fatal("expected error when app is nil")
	}
}

func TestFocusMainWindowNil(t *testing.T) {
	focusMainWindow(nil)
}
