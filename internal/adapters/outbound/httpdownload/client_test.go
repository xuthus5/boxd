package httpdownload_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/httpdownload"
	"github.com/xuthus5/boxd/kernel"
)

func TestGetSupportsConditionalRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"v1"`)
		writer.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		if request.Header.Get("If-None-Match") == `"v1"` {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"outbounds":[]}`))
	}))
	defer server.Close()

	client := httpdownload.New()
	first, err := client.Get(context.Background(), control.DownloadRequest{URL: server.URL})
	if err != nil {
		t.Fatalf("Get(first) error = %v", err)
	}
	if first.NotModified || first.ETag != `"v1"` || string(first.Body) != `{"outbounds":[]}` {
		t.Fatalf("first result = %+v", first)
	}
	second, err := client.Get(context.Background(), control.DownloadRequest{URL: server.URL, ETag: first.ETag})
	if err != nil {
		t.Fatalf("Get(second) error = %v", err)
	}
	if !second.NotModified || len(second.Body) != 0 {
		t.Fatalf("second result = %+v", second)
	}
}

func TestGetEnforcesSizeLimitAndValidatesURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", 64)))
	}))
	defer server.Close()

	client := httpdownload.New()
	if _, err := client.Get(context.Background(), control.DownloadRequest{URL: server.URL, MaxBytes: 8}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("oversized Get() error = %v", err)
	}
	if _, err := client.Get(context.Background(), control.DownloadRequest{URL: "file:///tmp/secret"}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("invalid URL Get() error = %v", err)
	}
}

func hasCode(err error, code kernel.ErrorCode) bool {
	var kernelError *kernel.Error
	return errors.As(err, &kernelError) && kernelError.Code == code
}
