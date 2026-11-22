package kernel

import (
	"context"
	"time"
)

// RuntimeController exposes engine operations that require a running engine.
type RuntimeController interface {
	OutboundGroups(ctx context.Context) ([]OutboundGroup, error)
	SelectOutbound(ctx context.Context, group, outbound string) error
	OutboundDelay(ctx context.Context, tag, url string) (time.Duration, error)
	ClashMode(ctx context.Context) (ClashMode, error)
	SetClashMode(ctx context.Context, mode string) error
	FlushDNS(ctx context.Context) error
}
