package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

const (
	urlTestDefaultsKey      = "subscription_urltest_defaults"
	urlTestManagedGroupsKey = "subscription_urltest_managed_groups"
	defaultURLTestURL       = "https://www.gstatic.com/generate_204"
	defaultURLTestInterval  = "3m"
	defaultURLTestTolerance = uint16(50)
)

func DefaultURLTestDefaults() model.URLTestDefaults {
	return model.URLTestDefaults{
		Enabled:   true,
		URL:       defaultURLTestURL,
		Interval:  defaultURLTestInterval,
		Tolerance: defaultURLTestTolerance,
	}
}

func ValidateURLTestDefaults(config model.URLTestDefaults) error {
	if err := validateURLTestURL(config.URL); err != nil {
		return err
	}
	return validateURLTestInterval(config.Interval)
}

func ValidateURLTestOverrides(overrides *model.URLTestOverrides) error {
	if overrides == nil {
		return nil
	}
	if overrides.URL != nil {
		if err := validateURLTestURL(*overrides.URL); err != nil {
			return err
		}
	}
	if overrides.Interval != nil {
		return validateURLTestInterval(*overrides.Interval)
	}
	return nil
}

func ResolveURLTest(
	defaults model.URLTestDefaults,
	overrides *model.URLTestOverrides,
) model.URLTestDefaults {
	if overrides == nil {
		return defaults
	}
	if overrides.Enabled != nil {
		defaults.Enabled = *overrides.Enabled
	}
	if overrides.URL != nil {
		defaults.URL = *overrides.URL
	}
	if overrides.Interval != nil {
		defaults.Interval = *overrides.Interval
	}
	if overrides.Tolerance != nil {
		defaults.Tolerance = *overrides.Tolerance
	}
	return defaults
}

func (m *SettingsManager) URLTestDefaults() (model.URLTestDefaults, error) {
	value, err := m.value(urlTestDefaultsKey)
	if err != nil {
		return model.URLTestDefaults{}, fmt.Errorf("reading urltest defaults: %w", err)
	}
	if value == "" {
		return DefaultURLTestDefaults(), nil
	}

	var config model.URLTestDefaults
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return model.URLTestDefaults{}, fmt.Errorf("decoding urltest defaults: %w", err)
	}
	if err := ValidateURLTestDefaults(config); err != nil {
		return model.URLTestDefaults{}, fmt.Errorf("validating urltest defaults: %w", err)
	}
	return config, nil
}

func (m *SettingsManager) SetURLTestDefaults(config model.URLTestDefaults) error {
	if err := ValidateURLTestDefaults(config); err != nil {
		return err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encoding urltest defaults: %w", err)
	}
	if err := m.Set(urlTestDefaultsKey, string(data)); err != nil {
		return fmt.Errorf("saving urltest defaults: %w", err)
	}
	return nil
}

func (m *SettingsManager) URLTestManagedGroups() ([]string, error) {
	value, err := m.value(urlTestManagedGroupsKey)
	if err != nil {
		return nil, fmt.Errorf("reading managed urltest groups: %w", err)
	}
	if value == "" {
		return []string{}, nil
	}

	groups := []string{}
	if err := json.Unmarshal([]byte(value), &groups); err != nil {
		return nil, fmt.Errorf("decoding managed urltest groups: %w", err)
	}
	return groups, nil
}

func (m *SettingsManager) SetURLTestManagedGroups(groups []string) error {
	data, err := json.Marshal(groups)
	if err != nil {
		return fmt.Errorf("encoding managed urltest groups: %w", err)
	}
	if err := m.Set(urlTestManagedGroupsKey, string(data)); err != nil {
		return fmt.Errorf("saving managed urltest groups: %w", err)
	}
	return nil
}

func (m *SettingsManager) value(key string) (string, error) {
	var value string
	err := m.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(settingsBucket)
		if bucket == nil {
			return fmt.Errorf("settings bucket is missing")
		}
		data := bucket.Get([]byte(key))
		if data != nil {
			value = string(data)
		}
		return nil
	})
	return value, err
}

func validateURLTestURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing urltest url: %w", err)
	}
	isHTTP := parsed.Scheme == "http" || parsed.Scheme == "https"
	if !isHTTP || parsed.Host == "" {
		return fmt.Errorf("urltest url must be an absolute http or https url")
	}
	return nil
}

func validateURLTestInterval(rawInterval string) error {
	interval, err := time.ParseDuration(rawInterval)
	if err != nil {
		return fmt.Errorf("parsing urltest interval: %w", err)
	}
	if interval <= 0 {
		return fmt.Errorf("urltest interval must be positive")
	}
	return nil
}
