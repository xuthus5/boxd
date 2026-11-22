package controlplane

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

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	passwordMinLength  = 8
	passwordMaxLength  = 1024
	passwordSaltBytes  = 16
	passwordHashBytes  = 32
	passwordIterations = 600_000
	sessionTokenBytes  = 32
	sessionTTL         = 24 * time.Hour
)

// SecurityService manages the local administrator credential and in-memory
// access sessions. Credentials are salted and hashed before persistence.
type SecurityService struct {
	repository control.SecurityRepository
	now        func() time.Time
	random     func([]byte) (int, error)
	iterations int
	ttl        time.Duration

	initMu   sync.Mutex
	mu       sync.Mutex
	sessions map[string]time.Time
}

var _ control.SecurityAPI = (*SecurityService)(nil)

func NewSecurityService(repository control.SecurityRepository) *SecurityService {
	return &SecurityService{
		repository: repository,
		now:        time.Now,
		random:     rand.Read,
		iterations: passwordIterations,
		ttl:        sessionTTL,
		sessions:   make(map[string]time.Time),
	}
}

func (s *SecurityService) Status(ctx context.Context) (control.SecurityStatus, error) {
	if s.repository == nil {
		return control.SecurityStatus{}, capabilityUnavailable("security.status", "security repository")
	}
	credential, initialized, err := s.repository.Load(ctx)
	if err != nil {
		return control.SecurityStatus{}, wrapStoreError("security.status", "load security credential", err)
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.purgeExpiredLocked(now)
	status := control.SecurityStatus{
		Initialized:  initialized,
		Mode:         control.SecurityModeLocal,
		SessionCount: len(s.sessions),
	}
	if initialized {
		status.Mode = control.SecurityModePassword
		status.CredentialUpdatedAt = credential.UpdatedAt.UTC()
	}
	s.mu.Unlock()
	return status, nil
}

func (s *SecurityService) Initialize(ctx context.Context, password string) (control.SecuritySession, error) {
	if s.repository == nil {
		return control.SecuritySession{}, capabilityUnavailable("security.initialize", "security repository")
	}
	s.initMu.Lock()
	defer s.initMu.Unlock()
	if err := validatePassword(password); err != nil {
		return control.SecuritySession{}, err
	}
	if _, initialized, err := s.repository.Load(ctx); err != nil {
		return control.SecuritySession{}, wrapStoreError("security.initialize", "load security credential", err)
	} else if initialized {
		return control.SecuritySession{}, kernel.NewError(kernel.ErrCodeConflict, "security.initialize", "password authentication is already initialized")
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := s.random(salt); err != nil {
		return control.SecuritySession{}, kernel.WrapError(kernel.ErrCodeInternal, "security.initialize", "generate password salt", err)
	}
	hash, err := derivePassword(password, salt, s.iterations)
	if err != nil {
		return control.SecuritySession{}, kernel.WrapError(kernel.ErrCodeInternal, "security.initialize", "derive password verifier", err)
	}
	now := s.now().UTC()
	credential := control.SecurityCredential{
		Salt:       salt,
		Hash:       hash,
		Iterations: s.iterations,
		UpdatedAt:  now,
	}
	if err := s.repository.Save(ctx, credential); err != nil {
		return control.SecuritySession{}, wrapStoreError("security.initialize", "save security credential", err)
	}
	return s.createSession(now)
}

func (s *SecurityService) Login(ctx context.Context, password string) (control.SecuritySession, error) {
	if s.repository == nil {
		return control.SecuritySession{}, capabilityUnavailable("security.login", "security repository")
	}
	if err := validatePassword(password); err != nil {
		return control.SecuritySession{}, unauthorized("security.login")
	}
	credential, initialized, err := s.repository.Load(ctx)
	if err != nil {
		return control.SecuritySession{}, wrapStoreError("security.login", "load security credential", err)
	}
	if !initialized {
		return control.SecuritySession{}, kernel.NewError(kernel.ErrCodeConflict, "security.login", "password authentication is not initialized").
			WithHint("initialize an administrator password first")
	}
	if len(credential.Salt) == 0 || len(credential.Hash) == 0 || credential.Iterations <= 0 {
		return control.SecuritySession{}, kernel.NewError(kernel.ErrCodeInternal, "security.login", "stored security credential is invalid")
	}
	candidate, err := derivePassword(password, credential.Salt, credential.Iterations)
	if err != nil {
		return control.SecuritySession{}, kernel.WrapError(kernel.ErrCodeInternal, "security.login", "derive password verifier", err)
	}
	if subtle.ConstantTimeCompare(candidate, credential.Hash) != 1 {
		return control.SecuritySession{}, unauthorized("security.login")
	}
	return s.createSession(s.now().UTC())
}

func (s *SecurityService) Logout(_ context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	return nil
}

func (s *SecurityService) ValidateSession(_ context.Context, token string) (control.SecuritySession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return control.SecuritySession{}, unauthorized("security.validate_session")
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	expiresAt, exists := s.sessions[token]
	if !exists {
		return control.SecuritySession{}, unauthorized("security.validate_session")
	}
	return control.SecuritySession{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *SecurityService) createSession(now time.Time) (control.SecuritySession, error) {
	var random [sessionTokenBytes]byte
	if _, err := s.random(random[:]); err != nil {
		return control.SecuritySession{}, kernel.WrapError(kernel.ErrCodeInternal, "security.session", "generate session token", err)
	}
	token := base64.RawURLEncoding.EncodeToString(random[:])
	expiresAt := now.Add(s.ttl)
	s.mu.Lock()
	s.sessions[token] = expiresAt
	s.mu.Unlock()
	return control.SecuritySession{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *SecurityService) purgeExpiredLocked(now time.Time) {
	for token, expiresAt := range s.sessions {
		if !expiresAt.After(now) {
			delete(s.sessions, token)
		}
	}
}

func validatePassword(password string) error {
	if len(password) < passwordMinLength {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "security.password", "password must contain at least 8 characters")
	}
	if len(password) > passwordMaxLength {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "security.password", "password is too long")
	}
	return nil
}

func derivePassword(password string, salt []byte, iterations int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iterations, passwordHashBytes)
}

func unauthorized(op string) error {
	return kernel.NewError(kernel.ErrCodeUnauthorized, op, "authentication failed").
		WithHint("provide a valid administrator password or session token")
}
