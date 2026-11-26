// Package logtext contains helpers for presenting structured log messages.
package logtext

import "strings"

// StripTerminalFormatting removes ANSI escape sequences and terminal color
// fragments that lost their leading escape byte while crossing a log sink.
// Kernel loggers sometimes format their output for a terminal, but structured
// log surfaces must store and render plain text.
func StripTerminalFormatting(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); {
		if value[index] != '\x1b' {
			if value[index] == '[' {
				if next, ok := skipBrokenSGR(value, index); ok {
					index = next
					continue
				}
			}
			result.WriteByte(value[index])
			index++
			continue
		}
		index = skipEscapeSequence(value, index)
	}
	return result.String()
}

// CleanMessage normalizes a structured log message for display. It removes
// terminal formatting and the level marker that some engines embed in the
// message even though the level is already stored as structured metadata.
func CleanMessage(level, value string) string {
	message := strings.TrimSpace(StripTerminalFormatting(value))
	for attempts := 0; attempts < 4; attempts++ {
		next := stripLevelMarker(level, message)
		if next == message {
			break
		}
		message = strings.TrimSpace(next)
	}
	return message
}

func stripLevelMarker(level, message string) string {
	for _, label := range levelMarkerLabels(level) {
		if strings.HasPrefix(message, label) {
			remainder := message[len(label):]
			if strings.HasPrefix(remainder, "[") {
				if end := strings.IndexByte(remainder, ']'); end > 1 && digitsOnly(remainder[1:end]) {
					return strings.TrimLeft(remainder[end+1:], " \t")
				}
			}
			if strings.HasPrefix(remainder, ":") {
				return strings.TrimLeft(remainder[1:], " \t")
			}
		}
		bracketed := "[" + label + "]"
		if strings.HasPrefix(message, bracketed) {
			return strings.TrimLeft(message[len(bracketed):], " \t")
		}
	}
	return message
}

func levelMarkerLabels(level string) []string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "warn":
		return []string{"WARN", "WARNING"}
	case "warning":
		return []string{"WARNING", "WARN"}
	case "trace":
		return []string{"TRACE"}
	case "debug":
		return []string{"DEBUG"}
	case "info":
		return []string{"INFO"}
	case "error":
		return []string{"ERROR"}
	case "fatal":
		return []string{"FATAL"}
	case "panic":
		return []string{"PANIC"}
	default:
		return nil
	}
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// skipBrokenSGR removes a color fragment such as [37m or [0m when the ESC byte
// was stripped by an intermediate writer. Legitimate log brackets do not end in
// the SGR final byte m and therefore remain untouched.
func skipBrokenSGR(value string, index int) (int, bool) {
	cursor := index + 1
	if cursor >= len(value) || value[cursor] < '0' || value[cursor] > '9' {
		return 0, false
	}
	for cursor < len(value) && ((value[cursor] >= '0' && value[cursor] <= '9') || value[cursor] == ';') {
		cursor++
	}
	if cursor < len(value) && value[cursor] == 'm' {
		return cursor + 1, true
	}
	return 0, false
}

func skipEscapeSequence(value string, escape int) int {
	index := escape + 1
	if index >= len(value) {
		return index
	}
	switch value[index] {
	case '[':
		return skipCSI(value, index+1)
	case ']', 'P', 'X', '^', '_':
		return skipControlString(value, index+1)
	}
	if value[index] >= 0x20 && value[index] <= 0x7e {
		return index + 1
	}
	return index
}

func skipCSI(value string, index int) int {
	for index < len(value) {
		current := value[index]
		switch {
		case current >= 0x40 && current <= 0x7e:
			return index + 1
		case current >= 0x20 && current <= 0x3f:
			index++
		default:
			return index
		}
	}
	return index
}

func skipControlString(value string, index int) int {
	for index < len(value) {
		if value[index] == '\a' {
			return index + 1
		}
		if value[index] == '\x1b' && index+1 < len(value) && value[index+1] == '\\' {
			return index + 2
		}
		index++
	}
	return index
}
