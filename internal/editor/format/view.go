package format

import (
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/component"
	"myjson/internal/editor"
)

func (e *FormatEditor) View(c *ui.Context) {
	e.Update(c)
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		e.Format()
	}
	if e.pendingTransform != nil && e.transformPolling {
		c.After(16 * time.Millisecond)
	}

	view := e.Render(c)
	component.Toasts(c, component.ToastOptions{Bottom: true}).
		Margin(0, 0, max(4, c.Theme().Space(2)), 0).
		AttachTo(view, ui.AnchorBottom, ui.AnchorBottom)
}

// Update 接收后台结果并调度诊断，输入区隐藏时也可继续调用。
func (e *FormatEditor) Update(c *ui.Context) {
	e.State.Theme = editor.Theme(c.Theme())
	e.refreshTransform(c)
	e.refreshDiagnostics(c)
}

// Render 绘制输入区，快捷键和提示容器由宿主管理。
func (e *FormatEditor) Render(c *ui.Context) ui.Element {
	return mygoeditor.View(c.Key("format-editor"), &e.State).Fill().Bind(&e.focus)
}
