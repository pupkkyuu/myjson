package format

import (
	"strings"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/pkg/json"
)

// SkipBlank 让结构化 JSON 操作保留空白文档；转义类操作不使用此策略。
func SkipBlank(transform func(string) (string, error)) func(string) (string, error) {
	return func(source string) (string, error) {
		if json.IsBlank(source) {
			return source, nil
		}
		return transform(source)
	}
}

// PrepareTransform 在后台准备结果；失败诊断对应原文，成功诊断对应规范化后的输出。
// 调用方须检查错误及快照版本后再提交。
func PrepareTransform(snapshot mygoeditor.Snapshot, transform func(string) (string, error)) (PreparedTransform, error) {
	source := snapshot.String()
	result := PreparedTransform{Version: snapshot.Version(), Text: source}
	text, err := transform(source)
	if err != nil {
		result.Diagnostics = json.DiagnosticsForError(source, err)
		return result, err
	}
	// 与 Buffer.Set 一致，先规范化换行，再比较变化并定位诊断。
	result.Text = strings.ReplaceAll(text, "\r\n", "\n")
	result.Changed = result.Text != source
	result.Diagnostics = json.Validate(result.Text)
	return result, nil
}
