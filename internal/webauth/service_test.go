package webauth

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"
	"time"
)

type memoryRepository struct {
	credential  Credential
	initialized bool
}

func (r *memoryRepository) Load(context.Context) (Credential, bool, error) {
	return r.credential, r.initialized, nil
}

func (r *memoryRepository) Save(_ context.Context, credential Credential) error {
	r.credential = credential
	r.initialized = true
	return nil
}

func TestCredentialSessionAndRandomRotationLifecycle(t *testing.T) {
	ctx := context.Background()
	repository := &memoryRepository{}
	service := New(repository)
	service.iterations = 1_000

	status, err := service.Status(ctx)
	if err != nil || status.Initialized || status.Mode != ModeLocal {
		t.Fatalf("initial Status() = %+v, %v", status, err)
	}
	if _, err := service.Initialize(ctx, "short"); err == nil {
		t.Fatal("Initialize() accepted a short password")
	}
	oldSession, err := service.Initialize(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	oldToken := oldSession.Token
	if oldToken == "" || len(repository.credential.Hash) == 0 {
		t.Fatalf("initialized session = %+v, credential = %+v", oldSession, repository.credential)
	}
	if _, err := service.Login(ctx, "wrong password"); err == nil {
		t.Fatal("Login() accepted a wrong password")
	}
	if _, err := service.Login(ctx, "correct horse battery staple"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	rotation, err := service.RotatePassword(ctx)
	if err != nil {
		t.Fatalf("RotatePassword() error = %v", err)
	}
	if len(rotation.Password) != rotatedLength || rotation.Session.Token == "" {
		t.Fatalf("rotation = %+v", rotation)
	}
	if strings.ContainsAny(rotation.Password, "IO0l1") {
		t.Fatalf("rotated password contains ambiguous characters: %q", rotation.Password)
	}
	if _, err := service.ValidateSession(ctx, oldToken); err == nil {
		t.Fatal("rotation did not invalidate the previous session")
	}
	if _, err := service.Login(ctx, "correct horse battery staple"); err == nil {
		t.Fatal("rotation left the previous password valid")
	}
	if _, err := service.Login(ctx, rotation.Password); err != nil {
		t.Fatalf("Login() with rotated password error = %v", err)
	}
	if session, err := service.ValidateSession(ctx, rotation.Session.Token); err != nil || session.Token != rotation.Session.Token {
		t.Fatalf("ValidateSession() = %+v, %v", session, err)
	}
}

func TestRandomPasswordUsesWholeAlphabet(t *testing.T) {
	var seen [256]bool
	for range 64 {
		password, err := generatePassword(rand.Read, 32)
		if err != nil {
			t.Fatalf("generatePassword() error = %v", err)
		}
		if len(password) != 32 {
			t.Fatalf("password length = %d, want 32", len(password))
		}
		for index := range password {
			seen[password[index]] = true
		}
	}
	for index := range passwordAlphabet {
		if !seen[passwordAlphabet[index]] {
			t.Fatalf("random password never used alphabet character %q", passwordAlphabet[index])
		}
	}
}

func TestSessionExpires(t *testing.T) {
	ctx := context.Background()
	service := New(&memoryRepository{})
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	service.iterations = 1_000
	session, err := service.Initialize(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(sessionTTL + time.Second) }
	if _, err := service.ValidateSession(ctx, session.Token); err == nil {
		t.Fatal("expired session remained valid")
	}
}
