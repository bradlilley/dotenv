// Copyright (c) 2025 Brad Lilley. All rights reserved.
// Use of this source code is governed by the Conduit CMS License
// that can be found in the LICENSE file.

package env

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func Load(filename string, override ...bool) (err error) {
	// Override is only variadic to make it optional.
	// If more than one boolean is set, return an error.
	if len(override) > 1 {
		return errors.New("too many arguments in call to env.Load")
	}

	lines, err := Parse(filename)
	if err != nil {
		return
	}

	err = setEnvVars(lines, len(override) > 0 && override[0])
	if err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}

	return nil
}

func Parse(filename string) (lines map[string]string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %w", filename, err)
	}
	defer file.Close()

	lines = make(map[string]string, 100)

	err = scanFile(file, lines)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	err = processLines(lines)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	return lines, nil
}

func scanFile(r io.Reader, lines map[string]string) error {
	scanner := bufio.NewScanner(r)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("line %d: %q key defined without \"=\" separator or value", lineNum, line)
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		// Empty keys are not allowed (e.g. =VALUE)
		if key == "" {
			return fmt.Errorf("line %d: %q value defined without key", lineNum, line)
		}

		lines[key] = stripInlineComments(val)
	}

	return scanner.Err()
}

func processLines(lines map[string]string) (err error) {
	for key, val := range lines {
		if doubleQuoted(val) {
			unquoted := stripQuotes(val)
			processed, err := processEscapeSequences(unquoted)
			if err != nil {
				// TODO: Evalute if %s is truly safe to use here.
				// What if someone does something like TESTING="value with %d literal percent d"?
				return fmt.Errorf("error processing escape sequences in %s=%s key-value pair: %w", key, val, err)
			}
			lines[key] = expandVariables(processed, lines)
		} else if singleQuoted(val) {
			lines[key] = stripQuotes(val)
		} else {
			lines[key] = expandVariables(val, lines)
		}
	}

	return nil
}

func processEscapeSequences(s string) (string, error) {
	var result strings.Builder
	result.Grow(len(s))

	// Convert input to runes for proper UTF-8 handling
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		// Check for an escape sequence
		if runes[i] == '\\' {
			// Check if this is the last character
			if i == len(runes)-1 {
				return "", fmt.Errorf("string ends with an incomplete escape sequence \"\\\" (trailing backslash)")
			}

			// Process the escape sequence
			// Move to the character after the backslash
			i++
			switch runes[i] {
			case 'n':
				result.WriteRune('\n') // Newline
			case 't':
				result.WriteRune('\t') // Tab
			case 'r':
				result.WriteRune('\r') // Carriage return
			case '"':
				result.WriteRune('"') // Double quote
			case '\'':
				result.WriteRune('\'') // Single quote
			case '\\':
				result.WriteRune('\\') // Backslash
			case '$':
				// Keep the dollar sign backslash to prevent variable expansion
				result.WriteString("\\$")
			default:
				return "", fmt.Errorf("invalid escape sequence \"\\%c\" at position %d", runes[i], i)
			}
		} else {
			// Regular character
			result.WriteRune(runes[i])
		}
	}

	return result.String(), nil
}

func expandVariables(s string, m map[string]string) string {
	// Return early if there's nothing to expand
	if !strings.Contains(s, "$") {
		return s
	}

	// Prepare literal dollar signs ($$) for expansion from original escaped dollars (\\$).
	// This is a workaround for os.Expand() not supporting escape sequences.
	s = strings.ReplaceAll(s, `\$`, "$$")

	return os.Expand(s, func(k string) string {
		// Replace $$ with $, completing \$ escape sequence
		if k == "$" {
			return "$"
		}
		if val, exists := m[k]; exists {
			// Strip quotes before returning because there's no guarentee
			// all inputs have been stripped yet
			return stripQuotes(val)
		}
		return ""
	})
}

func setEnvVars(lines map[string]string, override bool) (err error) {
	for key, val := range lines {
		_, exists := os.LookupEnv(key)
		if !exists || override {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}
		}
	}
	return nil
}

func doubleQuoted(s string) bool {
	return len(s) >= 2 && strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")
}

func singleQuoted(s string) bool {
	return len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")
}

func stripInlineComments(s string) string {
	s = strings.TrimSpace(s)

	// Return early if the string is empty
	if len(s) == 0 {
		return s
	}

	// Return early if the string doesn't contain a hash
	if !strings.ContainsRune(s, '#') {
		return s
	}

	singleQuoted := s[0] == '\''
	doubleQuoted := s[0] == '"'

	// If a string is quoted, find the closing quote
	// and remove everything that comes after it
	if singleQuoted || doubleQuoted {
		// Value: literal ' or "
		quoteType := s[0]

		for i := len(s) - 1; i > 0; i-- {
			if s[i] == quoteType {
				return s[:i+1]
			}
		}

		return s
	}

	// If a string isn't quoted, find the first hash
	// and remove it and everything that comes after
	for i := 0; i < len(s); i++ {
		if s[i] == '#' {
			// Trim again because comments can have white space before them.
			// This means there could be empty white space at the end of
			// the string after removing the comment.
			return strings.TrimSpace(s[:i])
		}
	}

	// No # found, return the original string
	return s
}

func stripQuotes(s string) string {
	if doubleQuoted(s) || singleQuoted(s) {
		return s[1 : len(s)-1]
	}
	return s
}
