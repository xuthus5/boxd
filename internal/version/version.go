package version

import "fmt"

// These values are overridden by release builds through -ldflags.
var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns the human-readable build identity used by CLI diagnostics.
func String() string {
	return fmt.Sprintf("boxd %s (commit %s, built %s)", Version, Commit, Date)
}
