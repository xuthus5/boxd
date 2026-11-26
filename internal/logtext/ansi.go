// Package logtext contains helpers for presenting structured log messages.
package logtext

import "strings"

// StripTerminalFormatting removes ANSI escape sequences from a log message.
// Kernel loggers sometimes format their output for a terminal, but structured
// log surfaces must store and render plain text.
func StripTerminalFormatting(value string) string {
	if !strings.ContainsRune(value, '\x1b') {
		return value
	}
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); {
		if value[index] != '\x1b' {
			result.WriteByte(value[index])
			index++
			continue
		}
		index = skipEscapeSequence(value, index)
	}
	return result.String()
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
