package core

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
	"golang.org/x/crypto/argon2"
)

const (
	adminPasswordHashKey    = "admin_password_hash"
	adminPasswordDefaultKey = "admin_password_default"
	defaultAdminPassword    = "admin123"
	credentialVersion       = 1
	credentialMemory        = 64 * 1024
	credentialIterations    = 3
	credentialParallelism   = 2
	credentialSaltLength    = 16
	credentialKeyLength     = 32
)

var (
	ErrCurrentPasswordInvalid = errors.New("current password is invalid")
	ErrWeakPassword           = errors.New("new password does not meet security requirements")
)

type passwordRecord struct {
	Version     int    `json:"version"`
	Memory      uint32 `json:"memory"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
	Salt        string `json:"salt"`
	Hash        string `json:"hash"`
}

func (m *SettingsManager) EnsureAdminCredential(username, initialPassword string) (bool, error) {
	if m.Get(adminPasswordHashKey) != "" {
		if m.VerifyAdminPassword(defaultAdminPassword) && !m.AdminPasswordIsDefault() {
			if err := m.Set(adminPasswordDefaultKey, "true"); err != nil {
				return false, fmt.Errorf("persist default password state: %w", err)
			}
		}
		return m.AdminPasswordIsDefault(), nil
	}

	isDefault := initialPassword == "" || initialPassword == defaultAdminPassword
	if isDefault {
		initialPassword = defaultAdminPassword
	}
	record, err := hashAdminPassword(initialPassword)
	if err != nil {
		return false, fmt.Errorf("hash administrator password: %w", err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("encode administrator password: %w", err)
	}
	err = m.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(settingsBucket)
		if bucket.Get([]byte(adminPasswordHashKey)) != nil {
			return nil
		}
		if err := bucket.Put([]byte(adminPasswordHashKey), encoded); err != nil {
			return err
		}
		return bucket.Put([]byte(adminPasswordDefaultKey), []byte(fmt.Sprintf("%t", isDefault)))
	})
	return isDefault, err
}

func (m *SettingsManager) VerifyAdminPassword(password string) bool {
	record, err := decodePasswordRecord(m.Get(adminPasswordHashKey))
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), record.salt(), record.Iterations, record.Memory, record.Parallelism, credentialKeyLength)
	expected, err := base64.RawStdEncoding.DecodeString(record.Hash)
	if err != nil || len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func (m *SettingsManager) AdminPasswordIsDefault() bool {
	return m.Get(adminPasswordDefaultKey) == "true"
}

func (m *SettingsManager) ChangeAdminPassword(username, currentPassword, newPassword string) error {
	if !m.VerifyAdminPassword(currentPassword) {
		return ErrCurrentPasswordInvalid
	}
	if err := validateAdminPassword(username, newPassword); err != nil {
		return err
	}
	record, err := hashAdminPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode administrator password: %w", err)
	}
	secret, err := generateJWTSecret()
	if err != nil {
		return fmt.Errorf("rotate jwt secret: %w", err)
	}
	return m.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(settingsBucket)
		if err := bucket.Put([]byte(adminPasswordHashKey), encoded); err != nil {
			return err
		}
		if err := bucket.Put([]byte(adminPasswordDefaultKey), []byte("false")); err != nil {
			return err
		}
		return bucket.Put([]byte(jwtSecretKey), []byte(secret))
	})
}

func validateAdminPassword(username, password string) error {
	weak := map[string]struct{}{"admin123": {}, "password": {}, "12345678": {}, "qwerty123": {}}
	if len(password) < 12 || strings.EqualFold(password, username) {
		return ErrWeakPassword
	}
	if _, found := weak[strings.ToLower(password)]; found {
		return ErrWeakPassword
	}
	return nil
}

func hashAdminPassword(password string) (passwordRecord, error) {
	salt := make([]byte, credentialSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return passwordRecord{}, err
	}
	hash := argon2.IDKey([]byte(password), salt, credentialIterations, credentialMemory, credentialParallelism, credentialKeyLength)
	return passwordRecord{Version: credentialVersion, Memory: credentialMemory, Iterations: credentialIterations, Parallelism: credentialParallelism, Salt: base64.RawStdEncoding.EncodeToString(salt), Hash: base64.RawStdEncoding.EncodeToString(hash)}, nil
}

func decodePasswordRecord(encoded string) (passwordRecord, error) {
	var record passwordRecord
	if err := json.Unmarshal([]byte(encoded), &record); err != nil {
		return passwordRecord{}, err
	}
	if record.Version != credentialVersion ||
		record.Memory != credentialMemory ||
		record.Iterations != credentialIterations ||
		record.Parallelism != credentialParallelism {
		return passwordRecord{}, fmt.Errorf("unsupported administrator password record")
	}
	if _, err := base64.RawStdEncoding.DecodeString(record.Salt); err != nil {
		return passwordRecord{}, err
	}
	return record, nil
}

func (r passwordRecord) salt() []byte {
	salt, _ := base64.RawStdEncoding.DecodeString(r.Salt)
	return salt
}
