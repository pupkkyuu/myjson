package json

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

type Diagnostic struct {
	Start, End   int
	Line, Column int
	Detail       string
}

// Validate 返回首个语法错误；输入有效或为空时返回 nil。
func Validate(source string) []Diagnostic {
	return ValidateBytes([]byte(source))
}

// ValidateBytes 返回首个语法错误；输入有效或仅含 JSON 空白时返回 nil。
// 校验期间只读取 source，不复制或修改其内容。
func ValidateBytes(source []byte) []Diagnostic {
	if len(bytes.Trim(source, " \t\r\n")) == 0 {
		return nil
	}
	decoder := jsontext.NewDecoder(bytes.NewBuffer(source))
	if _, err := decoder.ReadValue(); err != nil {
		return diagnosticsForError(source, err)
	}
	// Decoder accepts a stream of values; an editor document must contain one.
	tail := bytes.TrimLeft(source[int(decoder.InputOffset()):], " \t\r\n")
	if len(tail) != 0 {
		return diagnosticsForError(source, &jsontext.SyntacticError{
			ByteOffset: int64(len(source) - len(tail)),
			Err:        errors.New("unexpected data after top-level value"),
		})
	}
	return nil
}

// DiagnosticsForError 将 JSON 解析错误转换为源文本诊断。
// 支持经过包装的 jsontext 错误，包括格式化错误。
func DiagnosticsForError(source string, err error) []Diagnostic {
	return diagnosticsForError(source, err)
}

func diagnosticsForError[Source string | []byte](source Source, err error) []Diagnostic {
	if err == nil {
		return nil
	}
	start := 0
	detail := err.Error()
	if located, ok := errors.AsType[*LocationError](err); ok {
		detail = located.Cause.Error()
	}
	if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		detail = syntax.Error()
		start = min(max(int(syntax.ByteOffset), 0), len(source))
		anchor := source
		lineBreak := start < len(source) && (source[start] == '\n' || source[start] == '\r')
		if lineBreak {
			anchor = source[:start]
		}
		if errors.Is(syntax, io.ErrUnexpectedEOF) || lineBreak {
			switch anchor := any(anchor).(type) {
			case string:
				start = max(0, len(strings.TrimRight(anchor, " \t\r\n"))-1)
			case []byte:
				start = max(0, len(bytes.TrimRight(anchor, " \t\r\n"))-1)
			}
		}
	}
	for start > 0 && start < len(source) && !utf8.RuneStart(source[start]) {
		start--
	}
	_, size := utf8.DecodeRune([]byte(source[start:min(start+utf8.UTFMax, len(source))]))
	end := min(start+size, len(source))
	var line, column int
	switch source := any(source).(type) {
	case string:
		line, column = lineColumn(source, start)
	case []byte:
		line, column = lineColumnBytes(source, start)
	}
	return []Diagnostic{{
		Start: start, End: end,
		Line: line, Column: column, Detail: detail,
	}}
}
