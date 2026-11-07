package json

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"strings"
)

// Escape 将整个输入编码为 JSON 字符串，包含外层引号。
// 输入可以是普通文本，无需是有效的 JSON 文档。
func Escape(source string) (string, error) {
	output, err := json.Marshal(source)
	return string(output), err
}

// Unescape 解码一层 JSON 字符串；有效的非字符串 JSON 保持不变。
// 其他输入也可作为不带外层引号的转义字符串处理。
// 解码后的文本无需是 JSON。
func Unescape(source string) (string, error) {
	input := strings.TrimSpace(source)
	if jsontext.Value(input).IsValid() {
		if input[0] != '"' {
			return source, nil
		}
		var output string
		err := json.Unmarshal([]byte(input), &output)
		return output, err
	}
	var output string
	if err := json.Unmarshal([]byte(`"`+source+`"`), &output); err != nil {
		if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
			original := *syntax
			original.ByteOffset = max(0, min(syntax.ByteOffset-1, int64(len(source))))
			err = &original
		}
		return "", syntaxErrorWithLocation(source, err)
	}
	return output, nil
}
