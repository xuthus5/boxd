package core

import (
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

var urlTestDefaultsValidationCases = []struct {
	name    string
	config  model.URLTestDefaults
	wantErr bool
}{
	{name: "valid config", config: DefaultURLTestDefaults()},
	{name: "invalid scheme", config: model.URLTestDefaults{Enabled: true, URL: "ftp://example.com/test", Interval: "3m", Tolerance: 50}, wantErr: true},
	{name: "relative url", config: model.URLTestDefaults{Enabled: true, URL: "/generate_204", Interval: "3m", Tolerance: 50}, wantErr: true},
	{name: "malformed url", config: model.URLTestDefaults{Enabled: true, URL: "http://%", Interval: "3m", Tolerance: 50}, wantErr: true},
	{name: "invalid interval", config: model.URLTestDefaults{Enabled: true, URL: "https://example.com/test", Interval: "later", Tolerance: 50}, wantErr: true},
	{name: "zero interval", config: model.URLTestDefaults{Enabled: true, URL: "https://example.com/test", Interval: "0s", Tolerance: 50}, wantErr: true},
}

func TestDefaultURLTestDefaults(t *testing.T) {
	expected := model.URLTestDefaults{
		Enabled:   true,
		URL:       "https://www.gstatic.com/generate_204",
		Interval:  "3m",
		Tolerance: 50,
	}

	actual := DefaultURLTestDefaults()
	if actual != expected {
		t.Fatalf("defaults = %#v, want %#v", actual, expected)
	}
}

func TestValidateURLTestDefaults(t *testing.T) {
	for _, tt := range urlTestDefaultsValidationCases {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLTestDefaults(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateURLTestDefaults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURLTestOverrides(t *testing.T) {
	invalidURL := "file:///tmp/test"
	invalidInterval := "0s"
	validURL := "https://example.com/generate_204"
	validInterval := "30s"

	tests := []struct {
		name      string
		overrides *model.URLTestOverrides
		wantErr   bool
	}{
		{name: "nil overrides"},
		{
			name: "valid partial overrides",
			overrides: &model.URLTestOverrides{
				URL: &validURL, Interval: &validInterval,
			},
		},
		{
			name:      "invalid url override",
			overrides: &model.URLTestOverrides{URL: &invalidURL},
			wantErr:   true,
		},
		{
			name:      "invalid interval override",
			overrides: &model.URLTestOverrides{Interval: &invalidInterval},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLTestOverrides(tt.overrides)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateURLTestOverrides() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveURLTest(t *testing.T) {
	defaults := DefaultURLTestDefaults()
	disabled := false
	customURL := "https://example.com/generate_204"
	zeroTolerance := uint16(0)

	actual := ResolveURLTest(defaults, &model.URLTestOverrides{
		Enabled:   &disabled,
		URL:       &customURL,
		Tolerance: &zeroTolerance,
	})

	expected := model.URLTestDefaults{
		Enabled:   false,
		URL:       customURL,
		Interval:  defaults.Interval,
		Tolerance: 0,
	}
	if actual != expected {
		t.Fatalf("resolved = %#v, want %#v", actual, expected)
	}

	customInterval := "45s"
	actual = ResolveURLTest(defaults, &model.URLTestOverrides{Interval: &customInterval})
	if actual.Interval != customInterval {
		t.Fatalf("resolved interval = %q, want %q", actual.Interval, customInterval)
	}
}

func TestSettingsManagerURLTestDefaults(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	settings := NewSettingsManager(db)
	defaults, err := settings.URLTestDefaults()
	if err != nil {
		t.Fatalf("reading missing defaults: %v", err)
	}
	if defaults != DefaultURLTestDefaults() {
		t.Fatalf("missing defaults = %#v, want %#v", defaults, DefaultURLTestDefaults())
	}

	custom := model.URLTestDefaults{
		Enabled:   false,
		URL:       "https://example.com/generate_204",
		Interval:  "5m",
		Tolerance: 100,
	}
	if err := settings.SetURLTestDefaults(custom); err != nil {
		t.Fatalf("saving defaults: %v", err)
	}

	stored, err := settings.URLTestDefaults()
	if err != nil {
		t.Fatalf("loading defaults: %v", err)
	}
	if stored != custom {
		t.Fatalf("stored defaults = %#v, want %#v", stored, custom)
	}

	if err := settings.Set(urlTestDefaultsKey, "{"); err != nil {
		t.Fatalf("saving invalid json fixture: %v", err)
	}
	if _, err := settings.URLTestDefaults(); err == nil {
		t.Fatal("expected invalid stored json error")
	}
}

func TestSettingsManagerRejectsInvalidURLTestDefaults(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	settings := NewSettingsManager(db)
	invalid := DefaultURLTestDefaults()
	invalid.Interval = "0s"
	if err := settings.SetURLTestDefaults(invalid); err == nil {
		t.Fatal("expected invalid defaults error")
	}
	if err := settings.Set(urlTestDefaultsKey, `{"enabled":true,"url":"file:///tmp/test","interval":"3m","tolerance":50}`); err != nil {
		t.Fatalf("saving invalid fixture: %v", err)
	}
	if _, err := settings.URLTestDefaults(); err == nil {
		t.Fatal("expected invalid stored defaults error")
	}
}

func TestSettingsManagerURLTestDatabaseErrors(t *testing.T) {
	t.Run("missing bucket", func(t *testing.T) {
		db, cleanup := setupSettingsDB(t)
		defer cleanup()
		settings := NewSettingsManager(db)
		if err := db.Update(func(tx *bbolt.Tx) error { return tx.DeleteBucket(settingsBucket) }); err != nil {
			t.Fatalf("deleting settings bucket: %v", err)
		}
		if _, err := settings.URLTestDefaults(); err == nil {
			t.Fatal("expected missing bucket error")
		}
		if _, err := settings.URLTestManagedGroups(); err == nil {
			t.Fatal("expected managed groups missing bucket error")
		}
	})

	t.Run("closed database", func(t *testing.T) {
		db, cleanup := setupSettingsDB(t)
		defer cleanup()
		settings := NewSettingsManager(db)
		if err := db.Close(); err != nil {
			t.Fatalf("closing database: %v", err)
		}
		if err := settings.SetURLTestDefaults(DefaultURLTestDefaults()); err == nil {
			t.Fatal("expected closed database error")
		}
		if err := settings.SetURLTestManagedGroups([]string{"group"}); err == nil {
			t.Fatal("expected managed groups closed database error")
		}
	})
}

func TestSettingsManagerURLTestManagedGroups(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()

	settings := NewSettingsManager(db)
	groups, err := settings.URLTestManagedGroups()
	if err != nil {
		t.Fatalf("reading missing managed groups: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("missing managed groups = %#v", groups)
	}

	expected := []string{"alpha", "beta"}
	if err := settings.SetURLTestManagedGroups(expected); err != nil {
		t.Fatalf("saving managed groups: %v", err)
	}
	groups, err = settings.URLTestManagedGroups()
	if err != nil {
		t.Fatalf("loading managed groups: %v", err)
	}
	if len(groups) != len(expected) || groups[0] != expected[0] || groups[1] != expected[1] {
		t.Fatalf("managed groups = %#v, want %#v", groups, expected)
	}

	if err := settings.Set(urlTestManagedGroupsKey, `{`); err != nil {
		t.Fatalf("saving invalid managed groups fixture: %v", err)
	}
	if _, err := settings.URLTestManagedGroups(); err == nil {
		t.Fatal("expected invalid managed groups error")
	}
}
