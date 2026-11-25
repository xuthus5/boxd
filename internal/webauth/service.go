// Package webauth implements the administrator authentication used exclusively
// by the browser-facing Web ingress. It is intentionally kept outside the
// control-plane application boundary so desktop hosts never inherit Web
// sessions or password management.
package webauth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

const (
	PasswordMinLength  = 8
	PasswordMaxLength  = 1024
	passwordSaltBytes  = 16
	passwordHashBytes  = 32
	passwordIterations = 600_000
	rotatedLength      = 32
	sessionTokenBytes  = 32
	sessionTTL         = 24 * time.Hour
	passwordAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
)

// Mode describes the browser administrator authentication state.
type Mode string

const (
	ModeLocal    Mode = "local"
	ModePassword Mode = "password"
)

// Status is safe to expose before authentication.
type Status struct {
	Initialized         bool      `json:"initialized"`
	Mode                Mode      `json:"mode"`
	SessionCount        int       `json:"session_count"`
	CredentialUpdatedAt time.Time `json:"credential_updated_at,omitempty"`
}

// Session is issued only after successful initialization, login, or rotation.
type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Rotation is a one-time plaintext result. Callers must never persist the
// password or include it in logs.
type Rotation struct {
	Password string
	Session  Session
}

// Credential is a salted password verifier. Plaintext passwords never cross
// this storage port.
type Credential struct {
	Salt       []byte
	Hash       []byte
	Iterations int
	UpdatedAt  time.Time
}

// Repository persists the single browser administrator credential.
type Repository interface {
	Load(ctx context.Context) (Credential, bool, error)
	Save(ctx context.Context, credential Credential) error
}

// Service owns the browser administrator credential and in-memory sessions.
type Service struct {
	repository Repository
	now        func() time.Time
	random     func([]byte) (int, error)
	iterations int
	ttl        time.Duration

	credentialMu sync.Mutex
	mu           sync.Mutex
	sessions     map[string]time.Time
}

func New(repository Repository) *Service {
	return &Service{
		repository: repository,
		now:        time.Now,
		random:     rand.Read,
		iterations: passwordIterations,
		ttl:        sessionTTL,
		sessions:   make(map[string]time.Time),
	}
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	if s == nil || s.repository == nil {
		return Status{}, capabilityUnavailable("webauth.status")
	}
	credential, initialized, err := s.repository.Load(ctx)
	if err != nil {
		return Status{}, storeError("webauth.status", "load administrator credential", err)
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.purgeExpiredLocked(now)
	status := Status{
		Initialized:  initialized,
		Mode:         ModeLocal,
		SessionCount: len(s.sessions),
	}
	if initialized {
		status.Mode = ModePassword
		status.CredentialUpdatedAt = credential.UpdatedAt.UTC()
	}
	s.mu.Unlock()
	return status, nil
}

func (s *Service) Initialize(ctx context.Context, password string) (Session, error) {
	if s == nil || s.repository == nil {
		return Session{}, capabilityUnavailable("webauth.initialize")
	}
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()
	if err := ValidatePassword(password); err != nil {
		return Session{}, err
	}
	if _, initialized, err := s.repository.Load(ctx); err != nil {
		return Session{}, storeError("webauth.initialize", "load administrator credential", err)
	} else if initialized {
		return Session{}, kernel.NewError(kernel.ErrCodeConflict, "webauth.initialize", "password authentication is already initialized")
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := s.random(salt); err != nil {
		return Session{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.initialize", "generate password salt", err)
	}
	hash, err := derivePassword(password, salt, s.iterations)
	if err != nil {
		return Session{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.initialize", "derive password verifier", err)
	}
	now := s.now().UTC()
	credential := Credential{
		Salt:       salt,
		Hash:       hash,
		Iterations: s.iterations,
		UpdatedAt:  now,
	}
	if err := s.repository.Save(ctx, credential); err != nil {
		return Session{}, storeError("webauth.initialize", "save administrator credential", err)
	}
	return s.createSession(now)
}

func (s *Service) Login(ctx context.Context, password string) (Session, error) {
	if s == nil || s.repository == nil {
		return Session{}, capabilityUnavailable("webauth.login")
	}
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()
	if err := ValidatePassword(password); err != nil {
		return Session{}, unauthorized("webauth.login")
	}
	credential, initialized, err := s.repository.Load(ctx)
	if err != nil {
		return Session{}, storeError("webauth.login", "load administrator credential", err)
	}
	if !initialized {
		return Session{}, kernel.NewError(kernel.ErrCodeConflict, "webauth.login", "password authentication is not initialized").
			WithHint("initialize an administrator password first")
	}
	if len(credential.Salt) == 0 || len(credential.Hash) == 0 || credential.Iterations <= 0 {
		return Session{}, kernel.NewError(kernel.ErrCodeInternal, "webauth.login", "stored administrator credential is invalid")
	}
	candidate, err := derivePassword(password, credential.Salt, credential.Iterations)
	if err != nil {
		return Session{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.login", "derive password verifier", err)
	}
	if subtle.ConstantTimeCompare(candidate, credential.Hash) != 1 {
		return Session{}, unauthorized("webauth.login")
	}
	return s.createSession(s.now().UTC())
}

// RotatePassword generates a new random administrator password, invalidates
// every existing session, and returns a fresh session for the caller. It does
// not accept a caller-supplied password by design.
func (s *Service) RotatePassword(ctx context.Context) (Rotation, error) {
	if s == nil || s.repository == nil {
		return Rotation{}, capabilityUnavailable("webauth.rotate_password")
	}
	s.credentialMu.Lock()
	defer s.credentialMu.Unlock()
	if _, initialized, err := s.repository.Load(ctx); err != nil {
		return Rotation{}, storeError("webauth.rotate_password", "load administrator credential", err)
	} else if !initialized {
		return Rotation{}, kernel.NewError(kernel.ErrCodeConflict, "webauth.rotate_password", "password authentication is not initialized")
	}

	password, err := generatePassword(s.random, rotatedLength)
	if err != nil {
		return Rotation{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.rotate_password", "generate administrator password", err)
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := s.random(salt); err != nil {
		return Rotation{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.rotate_password", "generate password salt", err)
	}
	hash, err := derivePassword(password, salt, s.iterations)
	if err != nil {
		return Rotation{}, kernel.WrapError(kernel.ErrCodeInternal, "webauth.rotate_password", "derive password verifier", err)
	}
	now := s.now().UTC()
	credential := Credential{
		Salt:       salt,
		Hash:       hash,
		Iterations: s.iterations,
		UpdatedAt:  now,
	}
	token, err := s.newSessionToken()
	if err != nil {
		return Rotation{}, err
	}
	if err := s.repository.Save(ctx, credential); err != nil {
		return Rotation{}, storeError("webauth.rotate_password", "save administrator credential", err)
	}
	expiresAt := now.Add(s.ttl)
	s.mu.Lock()
	s.sessions = map[string]time.Time{token: expiresAt}
	s.mu.Unlock()
	return Rotation{
		Password: password,
		Session:  Session{Token: token, ExpiresAt: expiresAt},
	}, nil
}

func (s *Service) Logout(_ context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	return nil
}

func (s *Service) ValidateSession(_ context.Context, token string) (Session, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Session{}, unauthorized("webauth.validate_session")
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	expiresAt, exists := s.sessions[token]
	if !exists {
		return Session{}, unauthorized("webauth.validate_session")
	}
	return Session{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) createSession(now time.Time) (Session, error) {
	token, err := s.newSessionToken()
	if err != nil {
		return Session{}, err
	}
	expiresAt := now.Add(s.ttl)
	s.mu.Lock()
	s.sessions[token] = expiresAt
	s.mu.Unlock()
	return Session{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) newSessionToken() (string, error) {
	var random [sessionTokenBytes]byte
	if _, err := s.random(random[:]); err != nil {
		return "", kernel.WrapError(kernel.ErrCodeInternal, "webauth.session", "generate session token", err)
	}
	return base64.RawURLEncoding.EncodeToString(random[:]), nil
}

func (s *Service) purgeExpiredLocked(now time.Time) {
	for token, expiresAt := range s.sessions {
		if !expiresAt.After(now) {
			delete(s.sessions, token)
		}
	}
}

// ValidatePassword enforces the minimum browser administrator password policy.
func ValidatePassword(password string) error {
	if len(password) < PasswordMinLength {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "webauth.password", "password must contain at least 8 characters")
	}
	if len(password) > PasswordMaxLength {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "webauth.password", "password is too long")
	}
	return nil
}

func generatePassword(random func([]byte) (int, error), length int) (string, error) {
	if length <= 0 {
		return "", kernel.NewError(kernel.ErrCodeInternal, "webauth.password", "random password length must be positive")
	}
	result := make([]byte, length)
	limit := 256 - (256 % len(passwordAlphabet))
	for index := 0; index < length; {
		var value [1]byte
		if _, err := random(value[:]); err != nil {
			return "", err
		}
		if int(value[0]) >= limit {
			continue
		}
		result[index] = passwordAlphabet[int(value[0])%len(passwordAlphabet)]
		index++
	}
	return string(result), nil
}

func derivePassword(password string, salt []byte, iterations int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iterations, passwordHashBytes)
}

func capabilityUnavailable(op string) error {
	return kernel.NewError(kernel.ErrCodeFeatureDisabled, op, "web authentication is not configured").
		WithHint("start the Web ingress with a credential repository")
}

func storeError(op, message string, err error) error {
	return kernel.WrapError(kernel.ErrCodeInternal, op, message, err)
}

func unauthorized(op string) error {
	return kernel.NewError(kernel.ErrCodeUnauthorized, op, "authentication failed").
		WithHint("provide a valid administrator password or session token")
}
