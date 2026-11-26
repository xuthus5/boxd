// Package updatecheck defines the host-independent contract for checking the
// project's published desktop releases.
package updatecheck

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// ErrNoRelease means that the repository has no published stable release yet.
var ErrNoRelease = errors.New("updatecheck: no stable release published")

// Project describes the public source repository shown by host clients.
type Project struct {
	Name        string
	Slug        string
	URL         string
	ReleasesURL string
	IssuesURL   string
	License     string
}

// DefaultProject returns the canonical BOXD repository metadata.
func DefaultProject() Project {
	return Project{
		Name:        "BOXD",
		Slug:        "xuthus5/boxd",
		URL:         "https://github.com/xuthus5/boxd",
		ReleasesURL: "https://github.com/xuthus5/boxd/releases",
		IssuesURL:   "https://github.com/xuthus5/boxd/issues",
		License:     "Apache-2.0",
	}
}

// WithDefaults fills omitted metadata from DefaultProject while keeping hosts
// free to point a development build at a fork.
func (p Project) WithDefaults() Project {
	defaults := DefaultProject()
	if p.Name == "" {
		p.Name = defaults.Name
	}
	if p.Slug == "" {
		p.Slug = defaults.Slug
	}
	if p.URL == "" {
		p.URL = "https://github.com/" + strings.TrimSuffix(strings.TrimPrefix(p.Slug, "https://github.com/"), "/")
	}
	if p.ReleasesURL == "" {
		p.ReleasesURL = strings.TrimSuffix(p.URL, "/") + "/releases"
	}
	if p.IssuesURL == "" {
		p.IssuesURL = strings.TrimSuffix(p.URL, "/") + "/issues"
	}
	if p.License == "" {
		p.License = defaults.License
	}
	return p
}

// Release is a published, stable project release.
type Release struct {
	Tag         string
	Name        string
	URL         string
	PublishedAt time.Time
}

// Checker returns the newest published stable release.
type Checker interface {
	LatestRelease(context.Context) (Release, error)
}

// CompareVersions compares two human-readable versions using semantic-version
// ordering. A missing patch component is treated as zero, and a leading "v" is
// optional.
func CompareVersions(current, latest string) (int, error) {
	currentSemver, ok := normalizeSemver(current)
	if !ok {
		return 0, fmt.Errorf("invalid current version %q", current)
	}
	latestSemver, ok := normalizeSemver(latest)
	if !ok {
		return 0, fmt.Errorf("invalid release version %q", latest)
	}
	return semver.Compare(latestSemver, currentSemver), nil
}

func normalizeSemver(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if fields := strings.Fields(raw); len(fields) > 0 {
		raw = fields[0]
	}
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "v"), "V")
	if raw == "" {
		return "", false
	}

	core, suffix := raw, ""
	if index := strings.IndexAny(core, "-+"); index >= 0 {
		core, suffix = core[:index], core[index:]
	}
	parts := strings.Split(core, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return "", false
	}
	for _, part := range parts {
		if part == "" {
			return "", false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return "", false
			}
		}
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	value := "v" + strings.Join(parts, ".") + suffix
	return value, semver.IsValid(value)
}
