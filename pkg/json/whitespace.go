package json

import "strings"

// IsBlank reports whether source contains only JSON whitespace or is empty.
// JSON whitespace consists of space, tab, carriage return, and line feed.
func IsBlank(source string) bool {
	return strings.Trim(source, " \t\r\n") == ""
}
