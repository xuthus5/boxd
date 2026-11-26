package updatecheck_test

import (
	"testing"

	"github.com/xuthus5/boxd/internal/updatecheck"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
	}{
		{name: "older", current: "0.1.0", latest: "v0.2.0", want: 1},
		{name: "same", current: "v1.2.3", latest: "1.2.3", want: 0},
		{name: "newer local build", current: "1.3.0", latest: "v1.2.3", want: -1},
		{name: "development build", current: "0.1.0-dev", latest: "v0.1.0", want: 1},
		{name: "nightly channel", current: "0.0.0-nightly.7", latest: "v0.0.0-nightly.8", want: 1},
		{name: "missing patch", current: "1.2", latest: "v1.2.1", want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := updatecheck.CompareVersions(test.current, test.latest)
			if err != nil {
				t.Fatalf("CompareVersions() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", test.current, test.latest, got, test.want)
			}
		})
	}
}

func TestCompareVersionsRejectsMalformedBuildValues(t *testing.T) {
	if _, err := updatecheck.CompareVersions("development", "v1.0.0"); err == nil {
		t.Fatal("CompareVersions() accepted an invalid current version")
	}
	if _, err := updatecheck.CompareVersions("v1.0.0", "latest"); err == nil {
		t.Fatal("CompareVersions() accepted an invalid release version")
	}
}
