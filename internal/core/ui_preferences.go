package core

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/xuthus5/boxd/internal/model"
)

const uiPreferencesKey = "ui_preferences"

var (
	uiPreferenceThemes    = []string{"light", "dark", "system"}
	uiPreferenceLanguages = []string{"zh", "en"}
	uiPreferenceLogLevels = []string{"all", "debug", "info", "warn", "error"}
)

// DefaultUIPreferences returns the panel defaults when nothing is stored yet.
func DefaultUIPreferences() model.UIPreferences {
	return model.UIPreferences{
		Theme:           "system",
		Language:        "zh",
		MinimumLogLevel: "all",
	}
}

// ValidateUIPreferences rejects unsupported preference values.
func ValidateUIPreferences(prefs model.UIPreferences) error {
	if !slices.Contains(uiPreferenceThemes, prefs.Theme) {
		return fmt.Errorf("theme must be one of %v", uiPreferenceThemes)
	}
	if !slices.Contains(uiPreferenceLanguages, prefs.Language) {
		return fmt.Errorf("language must be one of %v", uiPreferenceLanguages)
	}
	if !slices.Contains(uiPreferenceLogLevels, prefs.MinimumLogLevel) {
		return fmt.Errorf("minimumLogLevel must be one of %v", uiPreferenceLogLevels)
	}
	return nil
}

// UIPreferences loads panel preferences from the database, falling back to defaults.
func (m *SettingsManager) UIPreferences() (model.UIPreferences, error) {
	value, err := m.value(uiPreferencesKey)
	if err != nil {
		return model.UIPreferences{}, fmt.Errorf("reading ui preferences: %w", err)
	}
	if value == "" {
		return DefaultUIPreferences(), nil
	}

	var prefs model.UIPreferences
	if err := json.Unmarshal([]byte(value), &prefs); err != nil {
		return model.UIPreferences{}, fmt.Errorf("decoding ui preferences: %w", err)
	}
	if err := ValidateUIPreferences(prefs); err != nil {
		return model.UIPreferences{}, fmt.Errorf("validating ui preferences: %w", err)
	}
	return prefs, nil
}

// SetUIPreferences validates and persists panel preferences.
func (m *SettingsManager) SetUIPreferences(prefs model.UIPreferences) error {
	if err := ValidateUIPreferences(prefs); err != nil {
		return err
	}
	data, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("encoding ui preferences: %w", err)
	}
	if err := m.Set(uiPreferencesKey, string(data)); err != nil {
		return fmt.Errorf("saving ui preferences: %w", err)
	}
	return nil
}
