// Package httpdownload implements the control-plane Downloader port.
package httpdownload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	defaultTimeout  = 30 * time.Second
	defaultMaxBytes = int64(16 << 20)
	userAgent       = "boxd/2"
)

// Client performs bounded, conditional HTTP GET requests.
type Client struct {
	httpClient *http.Client
}

var _ control.Downloader = (*Client)(nil)

// New constructs a client with a private HTTP transport.
func New() *Client {
	return &Client{httpClient: &http.Client{Timeout: defaultTimeout}}
}

// NewWithHTTPClient allows hosts and tests to inject a configured client.
func NewWithHTTPClient(client *http.Client) *Client {
	if client == nil {
		return New()
	}
	return &Client{httpClient: client}
}

// Get implements control.Downloader.
func (c *Client) Get(ctx context.Context, request control.DownloadRequest) (control.DownloadResult, error) {
	if err := validateRequest(request); err != nil {
		return control.DownloadResult{}, err
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	maxBytes := request.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(requestCtx, http.MethodGet, request.URL, nil)
	if err != nil {
		return control.DownloadResult{}, kernel.WrapError(kernel.ErrCodeInvalidConfig, "download.get", "create HTTP request", err)
	}
	httpRequest.Header.Set("User-Agent", userAgent)
	httpRequest.Header.Set("Accept", "application/json, text/plain, */*")
	if request.ETag != "" {
		httpRequest.Header.Set("If-None-Match", request.ETag)
	}
	if request.LastModified != "" {
		httpRequest.Header.Set("If-Modified-Since", request.LastModified)
	}

	response, err := c.client().Do(httpRequest)
	if err != nil {
		code := kernel.ErrCodeInternal
		if errors.Is(err, context.DeadlineExceeded) {
			code = kernel.ErrCodeTimeout
		} else if errors.Is(err, context.Canceled) {
			code = kernel.ErrCodeCanceled
		}
		return control.DownloadResult{}, kernel.WrapError(code, "download.get", "download remote content", err)
	}
	defer response.Body.Close()

	result := control.DownloadResult{
		URL:          response.Request.URL.String(),
		ContentType:  response.Header.Get("Content-Type"),
		ETag:         response.Header.Get("ETag"),
		LastModified: response.Header.Get("Last-Modified"),
	}
	if response.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return result, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return control.DownloadResult{}, kernel.NewError(kernel.ErrCodeInternal, "download.get", fmt.Sprintf("remote server returned HTTP %d", response.StatusCode)).
			WithHint("check the subscription or rule set URL and credentials")
	}
	if response.ContentLength > maxBytes {
		return control.DownloadResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "download.get", "remote content exceeds the configured size limit")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return control.DownloadResult{}, kernel.WrapError(kernel.ErrCodeInternal, "download.get", "read remote content", err)
	}
	if int64(len(body)) > maxBytes {
		return control.DownloadResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "download.get", "remote content exceeds the configured size limit")
	}
	result.Body = body
	return result, nil
}

func (c *Client) client() *http.Client {
	if c == nil || c.httpClient == nil {
		return &http.Client{Timeout: defaultTimeout}
	}
	return c.httpClient
}

func validateRequest(request control.DownloadRequest) error {
	parsed, err := url.Parse(strings.TrimSpace(request.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "download.get", "download URL must be an HTTP or HTTPS URL")
	}
	return nil
}
