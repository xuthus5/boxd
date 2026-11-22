package control

import (
	"context"
	"time"
)

// SecurityMode describes how host adapters protect the control plane.
type SecurityMode string

const (
	SecurityModeLocal    SecurityMode = "local"
	SecurityModePassword SecurityMode = "password"
)

// SecurityStatus is safe to expose before authentication.
type SecurityStatus struct {
	Initialized         bool         `json:"initialized"`
	Mode                SecurityMode `json:"mode"`
	SessionCount        int          `json:"session_count"`
	CredentialUpdatedAt time.Time    `json:"credential_updated_at,omitempty"`
}

// SecuritySession is returned only after successful initialization or login.
type SecuritySession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SecurityAPI owns local credentials and short-lived access sessions. Session
// tokens are intended for HTTP, Wails, and MyGo host adapters.
type SecurityAPI interface {
	Status(ctx context.Context) (SecurityStatus, error)
	Initialize(ctx context.Context, password string) (SecuritySession, error)
	Login(ctx context.Context, password string) (SecuritySession, error)
	Logout(ctx context.Context, token string) error
	ValidateSession(ctx context.Context, token string) (SecuritySession, error)
}
