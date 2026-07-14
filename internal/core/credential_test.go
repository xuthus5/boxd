package core

import (
	"errors"
	"strings"
	"testing"
)

func TestEnsureAdminCredentialUsesInitialPasswordOnce(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)

	isDefault, err := sm.EnsureAdminCredential("admin", "first-password-123")
	if err != nil {
		t.Fatalf("EnsureAdminCredential() error = %v", err)
	}
	if isDefault {
		t.Fatal("custom initial password marked as default")
	}
	if !sm.VerifyAdminPassword("first-password-123") {
		t.Fatal("initial password was not persisted")
	}
	if stored := sm.Get(adminPasswordHashKey); strings.Contains(stored, "first-password-123") {
		t.Fatal("stored credential contains plaintext password")
	}

	if _, err := sm.EnsureAdminCredential("admin", "second-password-456"); err != nil {
		t.Fatalf("second EnsureAdminCredential() error = %v", err)
	}
	if !sm.VerifyAdminPassword("first-password-123") || sm.VerifyAdminPassword("second-password-456") {
		t.Fatal("existing database credential was overwritten")
	}
}

func TestEnsureAdminCredentialFallsBackToDefaultPassword(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)

	isDefault, err := sm.EnsureAdminCredential("admin", "")
	if err != nil {
		t.Fatalf("EnsureAdminCredential() error = %v", err)
	}
	if !isDefault || !sm.AdminPasswordIsDefault() {
		t.Fatal("default password state was not persisted")
	}
	if !sm.VerifyAdminPassword(defaultAdminPassword) {
		t.Fatal("default password does not verify")
	}
}

func TestEnsureAdminCredentialMarksExplicitDefaultPassword(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)

	isDefault, err := sm.EnsureAdminCredential("admin", defaultAdminPassword)
	if err != nil {
		t.Fatalf("EnsureAdminCredential() error = %v", err)
	}
	if !isDefault || !sm.AdminPasswordIsDefault() {
		t.Fatal("explicit admin123 must be marked as the default password")
	}
}

func TestEnsureAdminCredentialMigratesDefaultPasswordState(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)
	if _, err := sm.EnsureAdminCredential("admin", defaultAdminPassword); err != nil {
		t.Fatal(err)
	}
	if err := sm.Set(adminPasswordDefaultKey, "false"); err != nil {
		t.Fatal(err)
	}

	isDefault, err := sm.EnsureAdminCredential("admin", "ignored-password")
	if err != nil {
		t.Fatalf("EnsureAdminCredential() error = %v", err)
	}
	if !isDefault || !sm.AdminPasswordIsDefault() {
		t.Fatal("existing admin123 credential state was not migrated")
	}
}

func TestChangeAdminPasswordValidatesAndRotatesSecret(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)
	if _, err := sm.EnsureAdminCredential("admin", "current-password-123"); err != nil {
		t.Fatal(err)
	}
	if err := sm.SetJWTSecret("initial-secret-value-1234"); err != nil {
		t.Fatal(err)
	}

	err := sm.ChangeAdminPassword("admin", "wrong-password", "replacement-password-456")
	if !errors.Is(err, ErrCurrentPasswordInvalid) {
		t.Fatalf("wrong current password error = %v", err)
	}
	if !sm.VerifyAdminPassword("current-password-123") {
		t.Fatal("credential changed after invalid current password")
	}

	err = sm.ChangeAdminPassword("admin", "current-password-123", "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak password error = %v", err)
	}

	err = sm.ChangeAdminPassword("admin", "current-password-123", "replacement-password-456")
	if err != nil {
		t.Fatalf("ChangeAdminPassword() error = %v", err)
	}
	if sm.VerifyAdminPassword("current-password-123") || !sm.VerifyAdminPassword("replacement-password-456") {
		t.Fatal("new password verification state is incorrect")
	}
	if sm.AdminPasswordIsDefault() {
		t.Fatal("changed password remains marked as default")
	}
	if got := sm.JWTSecret(); got == "initial-secret-value-1234" || len(got) < minJWTSecretLen {
		t.Fatalf("JWT secret was not rotated: %q", got)
	}
}

func TestVerifyAdminPasswordRejectsCorruptRecords(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	sm := NewSettingsManager(db)
	for _, record := range []string{
		"not-json",
		`{"version":99,"memory":65536,"iterations":3,"parallelism":2,"salt":"AA","hash":"AA"}`,
		`{"version":1,"memory":65536,"iterations":3,"parallelism":2,"salt":"!","hash":"AA"}`,
		`{"version":1,"memory":65536,"iterations":3,"parallelism":2,"salt":"AA","hash":"!"}`,
	} {
		if err := sm.Set(adminPasswordHashKey, record); err != nil {
			t.Fatal(err)
		}
		if sm.VerifyAdminPassword("anything") {
			t.Fatalf("corrupt record verified: %s", record)
		}
	}
}
