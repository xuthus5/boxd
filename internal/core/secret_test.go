package core

import (
	"strings"
	"testing"

	"go.etcd.io/bbolt"
)

func newSettingsDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	path := t.TempDir() + "/secret.db"
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() { _ = db.Close() }
}

func TestValidateJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "short", true},
		{"valid", "1234567890123456", false},
		{"long", strings.Repeat("a", 40), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	s1, err := generateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(s1) < minJWTSecretLen {
		t.Fatalf("generated secret too short: %d", len(s1))
	}
	s2, err := generateJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Fatal("generated secrets should differ")
	}
}

func TestSettingsJWTSecretGetSet(t *testing.T) {
	db, cleanup := newSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	if sm.JWTSecret() != "" {
		t.Fatal("expected empty secret initially")
	}

	if err := sm.SetJWTSecret("my-super-secret-value"); err != nil {
		t.Fatal(err)
	}
	if got := sm.JWTSecret(); got != "my-super-secret-value" {
		t.Fatalf("got %q, want %q", got, "my-super-secret-value")
	}
}

func TestSetJWTSecretRejectsShort(t *testing.T) {
	db, cleanup := newSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	if err := sm.SetJWTSecret("short"); err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestEnsureJWTSecretAutoGenerates(t *testing.T) {
	db, cleanup := newSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	got, generated, err := sm.EnsureJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < minJWTSecretLen {
		t.Fatalf("generated secret too short: %d", len(got))
	}
	if !generated {
		t.Fatal("should be generated when no stored secret exists")
	}
	// 应已落盘
	if sm.JWTSecret() != got {
		t.Fatal("EnsureJWTSecret did not persist generated secret")
	}
}

func TestEnsureJWTSecretReusesStored(t *testing.T) {
	db, cleanup := newSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	// 预置一个合法密钥
	stored := "pre-existing-valid-secret"
	if err := sm.SetJWTSecret(stored); err != nil {
		t.Fatal(err)
	}
	got, generated, err := sm.EnsureJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if got != stored {
		t.Fatalf("got %q, want %q", got, stored)
	}
	if generated {
		t.Fatal("should not be generated when stored secret exists")
	}
}

func TestEnsureJWTSecretRejectsInvalidStored(t *testing.T) {
	db, cleanup := newSettingsDB(t)
	defer cleanup()

	sm := NewSettingsManager(db)
	if err := sm.Set(jwtSecretKey, "too-short"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := sm.EnsureJWTSecret(); err == nil {
		t.Fatal("expected error for invalid stored secret")
	}
}

func TestEnsureJWTSecretGeneratedUniqueness(t *testing.T) {
	a, cleanup := newSettingsDB(t)
	defer cleanup()
	smA := NewSettingsManager(a)
	g1, _, err := smA.EnsureJWTSecret()
	if err != nil {
		t.Fatal(err)
	}

	b, cleanup2 := newSettingsDB(t)
	defer cleanup2()
	smB := NewSettingsManager(b)
	g2, _, err := smB.EnsureJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if g1 == g2 {
		t.Fatal("generated secrets across databases should differ")
	}
}
