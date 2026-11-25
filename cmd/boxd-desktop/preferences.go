package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const desktopPreferencesFile = "desktop-preferences.json"

type desktopPreferences struct {
	Theme string `json:"theme,omitempty"`
}

func (p desktopPreferences) isDark() bool {
	return strings.EqualFold(strings.TrimSpace(p.Theme), "dark")
}

func preferenceTheme(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

func loadDesktopPreferences(dataDir string) (desktopPreferences, error) {
	body, err := os.ReadFile(filepath.Join(dataDir, desktopPreferencesFile))
	if errors.Is(err, os.ErrNotExist) {
		return desktopPreferences{Theme: "light"}, nil
	}
	if err != nil {
		return desktopPreferences{Theme: "light"}, err
	}
	var preferences desktopPreferences
	if err := json.Unmarshal(body, &preferences); err != nil {
		return desktopPreferences{Theme: "light"}, fmt.Errorf("decode desktop preferences: %w", err)
	}
	if preferences.Theme == "" {
		preferences.Theme = "light"
	}
	return preferences, nil
}

func saveDesktopPreferences(dataDir string, preferences desktopPreferences) error {
	body, err := json.MarshalIndent(preferences, "", "  ")
	if err != nil {
		return fmt.Errorf("encode desktop preferences: %w", err)
	}
	body = append(body, '\n')
	file, err := os.CreateTemp(dataDir, ".desktop-preferences-*.tmp")
	if err != nil {
		return fmt.Errorf("create desktop preferences: %w", err)
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure desktop preferences: %w", err)
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return fmt.Errorf("write desktop preferences: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close desktop preferences: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(dataDir, desktopPreferencesFile)); err != nil {
		return fmt.Errorf("replace desktop preferences: %w", err)
	}
	return nil
}
