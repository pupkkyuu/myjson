package json

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// lineColumn converts a byte offset into one-based line and rune column numbers.
// The caller is responsible for choosing an offset within source.
func lineColumn(source string, offset int) (int, int) {
	prefix := source[:offset]
	line := strings.Count(prefix, "\n") + 1
	column := utf8.RuneCountInString(prefix[strings.LastIndex(prefix, "\n")+1:]) + 1
	return line, column
}

func lineColumnBytes(source []byte, offset int) (int, int) {
	prefix := source[:offset]
	line := bytes.Count(prefix, []byte{'\n'}) + 1
	column := utf8.RuneCount(prefix[bytes.LastIndexByte(prefix, '\n')+1:]) + 1
	return line, column
}
