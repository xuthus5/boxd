package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/xuthus5/boxd/kernel"
)

const (
	defaultDownloadTimeout = 30
	defaultDownloadBytes   = int64(16 << 20)
)

func capabilityUnavailable(op, capability string) error {
	return kernel.NewError(kernel.ErrCodeFeatureDisabled, op, capability+" is unavailable").
		WithHint("configure the required adapter in the composition root")
}

func scopedStableID(prefix string, values ...any) string {
	hasher := sha256.New()
	for _, value := range values {
		_, _ = fmt.Fprintf(hasher, "%v\x00", value)
	}
	return prefix + "-" + hex.EncodeToString(hasher.Sum(nil)[:8])
}
