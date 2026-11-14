package json

import "encoding/json/jsontext"

// Compact 按 jsontext 默认规则验证并压缩 JSON。
func Compact(source string) (string, error) {
	output, err := jsontext.AppendFormat(nil, source)
	if err != nil {
		return "", syntaxErrorWithLocation(source, err)
	}
	return string(output), nil
}
