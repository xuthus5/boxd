package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	maxErrorLength = 240
	urlPatternText = `https?://[^\s]+`
)

var sensitiveURLPattern = regexp.MustCompile(urlPatternText)

func shortError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	message = strings.ReplaceAll(message, "\n", " ")
	message = sensitiveURLPattern.ReplaceAllString(message, "[redacted-url]")
	return truncateText(message, maxErrorLength)
}

func formatBytes(value int64) string {
	if value < 0 {
		value = 0
	}
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	scaled := float64(value)
	index := -1
	for scaled >= unit && index < len(units)-1 {
		scaled /= unit
		index++
	}
	return fmt.Sprintf("%.1f %s", scaled, units[index])
}

func formatRate(value float32) string {
	if value < 0 {
		value = 0
	}
	return formatBytes(int64(value)) + "/s"
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "Never"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func formatShortTime(value time.Time) string {
	if value.IsZero() {
		return "Never"
	}
	return value.Local().Format("15:04:05")
}

func formatRelative(value time.Time, now time.Time) string {
	if value.IsZero() {
		return "Never"
	}
	delta := now.Sub(value)
	switch {
	case delta < time.Minute:
		return "Just now"
	case delta < time.Hour:
		return fmt.Sprintf("%d min ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(delta.Hours()))
	default:
		return value.Local().Format("2006-01-02")
	}
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func kernelStatusLabel(running bool, busy bool) string {
	if busy {
		return "Working"
	}
	if running {
		return "Running"
	}
	return "Stopped"
}

func truncateText(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}
