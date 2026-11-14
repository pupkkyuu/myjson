package json

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
)

// Format 按 jsontext 默认规则验证并格式化 JSON。
func Format(source string) (string, error) {
	return FormatWithIndent(source, "  ")
}

// FormatWithIndent 使用指定的每级缩进格式化 JSON；缩进只能包含空格或制表符。
func FormatWithIndent(source, indent string) (string, error) {
	output, err := jsontext.AppendFormat(nil, source, jsontext.WithIndent(indent))
	if err != nil {
		return "", syntaxErrorWithLocation(source, err)
	}
	return string(output), nil
}

type LocationError struct {
	Line, Column int
	Cause        error
}

func (e *LocationError) Error() string {
	return fmt.Sprintf("line %d, column %d: %v", e.Line, e.Column, e.Cause)
}

func (e *LocationError) Unwrap() error { return e.Cause }

func syntaxErrorWithLocation(source string, err error) error {
	if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		offset := min(max(int(syntax.ByteOffset), 0), len(source))
		line, column := lineColumn(source, offset)
		return &LocationError{Line: line, Column: column, Cause: err}
	}
	return err
}
