package main

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestShortError(t *testing.T) {
	if got := shortError(nil); got != "" {
		t.Fatalf("shortError(nil) = %q", got)
	}
	got := shortError(errors.New("request https://example.test/path?token=secret\nsecond line"))
	if strings.Contains(got, "https://") || strings.Contains(got, "\n") {
		t.Fatalf("shortError leaked URL or newline: %q", got)
	}
	long := strings.Repeat("x", maxErrorLength+20)
	got = shortError(errors.New(long))
	if utf8.RuneCountInString(got) > maxErrorLength || !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated error = %q (runes %d)", got, utf8.RuneCountInString(got))
	}
}

func TestFormatBytesAndRate(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{name: "negative", in: -1, want: "0 B"},
		{name: "bytes", in: 1023, want: "1023 B"},
		{name: "kilobytes", in: 2048, want: "2.0 KB"},
		{name: "megabytes", in: 5 * 1024 * 1024, want: "5.0 MB"},
		{name: "gigabytes", in: 2 * 1024 * 1024 * 1024, want: "2.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBytes(tt.in); got != tt.want {
				t.Fatalf("formatBytes(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	if got := formatRate(1536); got != "1.5 KB/s" {
		t.Fatalf("formatRate = %q", got)
	}
	if got := formatRate(-1); got != "0 B/s" {
		t.Fatalf("negative rate = %q", got)
	}
}

func TestFormatTimeAndRelative(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 30, 0, 0, time.Local)
	if got := formatTime(time.Time{}); got != "Never" {
		t.Fatalf("zero full time = %q", got)
	}
	if got := formatShortTime(time.Time{}); got != "Never" {
		t.Fatalf("zero short time = %q", got)
	}
	if got := formatTime(now); got != "2026-11-12 10:30:00" {
		t.Fatalf("formatTime = %q", got)
	}
	if got := formatShortTime(now); got != "10:30:00" {
		t.Fatalf("formatShortTime = %q", got)
	}
	relative := []struct {
		when time.Time
		want string
	}{
		{when: time.Time{}, want: "Never"},
		{when: now.Add(-30 * time.Second), want: "Just now"},
		{when: now.Add(-2 * time.Minute), want: "2 min ago"},
		{when: now.Add(-2 * time.Hour), want: "2 h ago"},
		{when: now.Add(-48 * time.Hour), want: "2026-11-10"},
	}
	for _, tt := range relative {
		if got := formatRelative(tt.when, now); got != tt.want {
			t.Errorf("formatRelative(%v) = %q, want %q", tt.when, got, tt.want)
		}
	}
}

func TestTextHelpers(t *testing.T) {
	if got := truncateText("abcdef", 4); got != "abc…" {
		t.Fatalf("truncateText = %q", got)
	}
	if got := truncateText("abcdef", 0); got != "abcdef" {
		t.Fatalf("zero limit = %q", got)
	}
	if got := truncateText("abcdef", 1); got != "a" {
		t.Fatalf("one limit = %q", got)
	}
	if got := yesNo(true); got != "Yes" {
		t.Fatalf("yesNo(true) = %q", got)
	}
	if got := yesNo(false); got != "No" {
		t.Fatalf("yesNo(false) = %q", got)
	}
	if got := kernelStatusLabel(true, false); got != "Running" {
		t.Fatalf("status = %q", got)
	}
	if got := kernelStatusLabel(false, true); got != "Working" {
		t.Fatalf("busy status = %q", got)
	}
	if got := kernelStatusLabel(false, false); got != "Stopped" {
		t.Fatalf("stopped status = %q", got)
	}
}
