package format

import (
	"strings"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/pkg/json"
)

func (e *FormatEditor) Format() {
	indent := strings.Repeat(" ", max(1, e.TabSize))
	e.transform(SkipBlank(func(source string) (string, error) { return json.FormatWithIndent(source, indent) }))
}

func (e *FormatEditor) Compact() {
	e.transform(SkipBlank(json.Compact))
}

func (e *FormatEditor) Escape() {
	e.transform(json.Escape)
}

func (e *FormatEditor) Unescape() {
	e.transform(json.Unescape)
}

func (e *FormatEditor) SortFields() {
	indent := strings.Repeat(" ", max(1, e.TabSize))
	e.transform(SkipBlank(func(source string) (string, error) { return json.SortFieldsWithIndent(source, indent) }))
}

func (e *FormatEditor) FoldAll() {
	e.Run(mygoeditor.FoldAll)
	e.focus.Focus()
}

func (e *FormatEditor) UnfoldAll() {
	e.Run(mygoeditor.UnfoldAll)
	e.focus.Focus()
}

func (e *FormatEditor) Focus() {
	e.focus.Focus()
}

func (e *FormatEditor) ToggleWrap() {
	e.SoftWrap = !e.SoftWrap
	e.focus.Focus()
}

func (e *FormatEditor) WrapEnabled() bool {
	return e.SoftWrap
}

func (e *FormatEditor) Search() {
	e.Run(mygoeditor.Find)
}

func (e *FormatEditor) CopyEnabled() bool {
	return true
}

func (e *FormatEditor) CopyAll(writeClipboard func(string)) {
	writeClipboard(e.Buffer.String())
	e.focus.Focus()
}

func (e *FormatEditor) Clear() {
	e.pendingTransform = nil
	e.transformPolling = false
	if e.Buffer.Len() != 0 {
		e.Buffer.Set("")
	}
	e.State.Select(0, 0)
	e.ScrollTo(0)
	e.focus.Focus()
}

// CancelFocus 撤销尚未执行的聚焦请求。
func (e *FormatEditor) CancelFocus() { e.focus.CancelFocus() }

func (e *FormatEditor) Loading() bool { return e.pendingTransform != nil }
