package githubrelease_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxd/internal/adapters/outbound/githubrelease"
	"github.com/xuthus5/boxd/internal/updatecheck"
)

func TestLatestReleaseSelectsHighestStableVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/repos/xuthus5/boxd/releases" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("per_page") != "30" {
			t.Errorf("per_page = %q", request.URL.Query().Get("per_page"))
		}
		if request.Header.Get("User-Agent") != "BOXD desktop updater" {
			t.Errorf("user agent = %q", request.Header.Get("User-Agent"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{"tag_name":"v1.2.0","name":"Stable 1.2","html_url":"https://github.com/xuthus5/boxd/releases/tag/v1.2.0","draft":false,"prerelease":false,"published_at":"2026-11-20T10:00:00Z"},
			{"tag_name":"nightly","name":"Nightly","html_url":"https://github.com/xuthus5/boxd/releases/tag/nightly","draft":false,"prerelease":true,"published_at":"2026-11-25T10:00:00Z"},
			{"tag_name":"v1.3.0","name":"Stable 1.3","html_url":"https://github.com/xuthus5/boxd/releases/tag/v1.3.0","draft":false,"prerelease":false,"published_at":"2026-11-22T10:00:00Z"},
			{"tag_name":"v9.9.9","name":"Draft","html_url":"https://github.com/xuthus5/boxd/releases/tag/v9.9.9","draft":true,"prerelease":false,"published_at":"2026-11-23T10:00:00Z"}
		]`))
	}))
	defer server.Close()

	client := githubrelease.NewWithHTTPClient("xuthus5/boxd", server.URL, server.Client())
	release, err := client.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease() error = %v", err)
	}
	if release.Tag != "v1.3.0" || release.Name != "Stable 1.3" {
		t.Fatalf("release = %+v", release)
	}
}

func TestLatestReleaseReportsNoStableRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`[{"tag_name":"nightly","draft":false,"prerelease":true}]`))
	}))
	defer server.Close()

	client := githubrelease.NewWithHTTPClient("xuthus5/boxd", server.URL, server.Client())
	_, err := client.LatestRelease(context.Background())
	if !errors.Is(err, updatecheck.ErrNoRelease) {
		t.Fatalf("LatestRelease() error = %v, want ErrNoRelease", err)
	}
}

func TestLatestReleaseSurfacesAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer server.Close()

	client := githubrelease.NewWithHTTPClient("xuthus5/boxd", server.URL, server.Client())
	_, err := client.LatestRelease(context.Background())
	if err == nil || err.Error() != "GitHub releases API returned HTTP 403: API rate limit exceeded" {
		t.Fatalf("LatestRelease() error = %v", err)
	}
}
