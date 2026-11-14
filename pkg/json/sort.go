package json

import "encoding/json/jsontext"

// SortFields 递归排序对象键，输出以 2 空格缩进的 JSON，保留数组顺序。
// 使用 jsontext 默认规则验证并规范化字符串，保留数字原文。
// 键按 UTF-16 字典序排列，与 JavaScript 字符串比较规则一致。
func SortFields(source string) (string, error) {
	return SortFieldsWithIndent(source, "  ")
}

// SortFieldsWithIndent 递归排序对象键，并使用指定缩进格式化。
func SortFieldsWithIndent(source, indent string) (string, error) {
	output := jsontext.Value(source)
	if err := output.Format(jsontext.ReorderRawObjects(true), jsontext.WithIndent(indent)); err != nil {
		return "", syntaxErrorWithLocation(source, err)
	}
	return string(output), nil
}
