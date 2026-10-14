package diff

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/component"
	"myjson/internal/editor"
	"myjson/internal/editor/format"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

func (e *DiffEditor) Loading() bool {
	return e.transformResults != nil || e.left.Loading() || e.right.Loading()
}

func (e *DiffEditor) TransformEnabled() bool { return !e.comparing && !e.Loading() }

// transform 异步处理左右文档快照，任一侧失败时停止后续处理。
func (e *DiffEditor) transform(transform func(string) (string, error)) {
	if !e.TransformEnabled() {
		return
	}
	snapshots := [2]mygoeditor.Snapshot{e.left.Buffer.Snapshot(), e.right.Buffer.Snapshot()}
	results := make(chan [2]diffPreparedTransform, 1)
	wake := e.Invalidate
	e.transformResults, e.transformCanceled, e.transformPolling = results, false, wake == nil
	go func() {
		var prepared [2]diffPreparedTransform
		for i, snapshot := range snapshots {
			prepared[i].Version = snapshot.Version()
		}
		for i, snapshot := range snapshots {
			source := snapshot.String()
			r := &prepared[i]
			r.Text = source
			if strings.Trim(source, " \t\r\n") == "" {
				continue
			}
			r.Text, r.err = transform(source)
			if r.err != nil {
				r.Diagnostics = json.DiagnosticsForError(source, r.err)
				break
			}
			r.Text = strings.ReplaceAll(r.Text, "\r\n", "\n")
			r.Changed = r.Text != source
			r.Diagnostics = json.Validate(r.Text)
		}
		results <- prepared
		if wake != nil {
			wake()
		}
	}()
}

// refreshTransform 检查两侧版本和错误，全部通过后才提交结果。
func (e *DiffEditor) refreshTransform(c *ui.Context) {
	if e.transformResults == nil {
		return
	}
	select {
	case results := <-e.transformResults:
		e.transformResults = nil
		c.Invalidate()
		if e.transformCanceled {
			return
		}
		panes := [2]*format.FormatEditor{e.left, e.right}
		for i, pane := range panes {
			if pane.Buffer.Version() != results[i].Version {
				return
			}
		}
		for i, r := range results {
			if r.err == nil {
				continue
			}
			panes[i].SetDiagnostics(r.Version, editor.Diagnostics(r.Diagnostics))
			message := r.err.Error()
			if len(r.Diagnostics) != 0 {
				d := r.Diagnostics[0]
				message = locales.JsonErrorLocation(d.Line, d.Column, d.Detail)
			}
			if i == 0 {
				message = locales.DiffLeftError(message)
			} else {
				message = locales.DiffRightError(message)
			}
			c.AddToast(ui.Toast{Title: message, Type: component.ToastError})
			e.activeSide = i
			e.Focus()
			return
		}
		for i, pane := range panes {
			pane.ApplyTransform(results[i].PreparedTransform)
		}
		e.Focus()
	default:
		if e.transformPolling {
			c.After(16 * time.Millisecond)
		}
	}
}
