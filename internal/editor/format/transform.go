package format

import (
	"errors"
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/editor"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

// transform 异步处理当前文档快照，并在完成后通知界面刷新。
func (e *FormatEditor) transform(transform func(string) (string, error)) {
	if e.Loading() {
		return
	}
	snapshot := e.Buffer.Snapshot()
	results := make(chan transformResult, 1)
	wake := e.Invalidate
	e.pendingTransform = results
	e.transformPolling = wake == nil

	go func() {
		text, err := transform(snapshot.String())
		results <- transformResult{
			version: snapshot.Version(),
			text:    text,
			err:     err,
		}

		if wake != nil {
			wake()
		}
	}()
}

// refreshTransform 仅应用版本匹配的异步结果，并报告处理错误。
func (e *FormatEditor) refreshTransform(c *ui.Context) {
	select {
	case r := <-e.pendingTransform:
		e.pendingTransform = nil
		e.transformPolling = false
		c.Invalidate()
		if r.version == e.Buffer.Version() {
			if r.err != nil {
				message := r.err.Error()
				if located, ok := errors.AsType[*json.LocationError](r.err); ok {
					message = locales.JsonErrorLocation(located.Line, located.Column, located.Cause.Error())
				}
				c.AddToast(ui.Toast{Title: message, Type: component.ToastError})
			} else {
				e.Buffer.Set(r.text)
				e.State.Select(0, 0)
				e.ScrollTo(0)
				e.focus.Focus()
			}
		}
	default:
		if e.pendingTransform != nil && e.transformPolling {
			c.After(16 * time.Millisecond)
		}
	}
}

// ApplyTransform 在 UI 线程提交版本匹配的结果，并同步诊断和位置。
// 多文档宿主须先检查所有文档的版本和错误，再提交任一文档。
func (e *FormatEditor) ApplyTransform(result PreparedTransform) bool {
	if result.Version != e.Buffer.Version() {
		return false
	}
	if result.Changed {
		e.Buffer.Set(result.Text)
		e.Select(0, 0)
		e.ScrollTo(0)
	}
	e.version = e.Buffer.Version()
	e.deadline = time.Time{}
	e.SetDiagnostics(e.version, editor.Diagnostics(result.Diagnostics))
	return true
}
