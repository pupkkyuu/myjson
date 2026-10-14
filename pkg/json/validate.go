package json

import (
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
	if strings.Trim(source, " \t\r\n") == "" {
		return nil
	}
	data := []byte(source)
	if jsontext.Value(data).IsValid() {
		return nil
	}
	document := jsontext.Value(data)
	return DiagnosticsForError(source, document.Format())
}

// DiagnosticsForError 将 JSON 解析错误转换为源文本诊断。
// 支持经过包装的 jsontext 错误，包括格式化错误。
func DiagnosticsForError(source string, err error) []Diagnostic {
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
			start = max(0, len(strings.TrimRight(anchor, " \t\r\n"))-1)
		}
	}
	for start > 0 && start < len(source) && !utf8.RuneStart(source[start]) {
		start--
	}
	_, size := utf8.DecodeRuneInString(source[start:])
	end := min(start+size, len(source))
	prefix := source[:start]
	line := strings.Count(prefix, "\n") + 1
	column := utf8.RuneCountInString(prefix[strings.LastIndex(prefix, "\n")+1:]) + 1
	return []Diagnostic{{
		Start: start, End: end,
		Line: line, Column: column, Detail: detail,
	}}
}
