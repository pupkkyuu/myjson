package diff

import (
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/component"
	"myjson/internal/editor"
	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func (e *DiffEditor) View(c *ui.Context) {
	e.refreshTransform(c)
	panes := [2]*format.FormatEditor{e.left, e.right}
	for _, pane := range panes {
		pane.Invalidate = e.Invalidate
		pane.Update(c)
	}
	e.syncVersions()
	if !e.comparing && c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		e.Format()
	}
	if e.transformResults != nil && e.transformPolling {
		c.After(16 * time.Millisecond)
	}
	focus, focusSide := e.requestFocus, e.activeSide
	e.requestFocus = false
	if focus || e.comparing {
		for _, pane := range panes {
			pane.CancelFocus()
		}
	}

	view := ui.Row(c.Key("diff-editor")).Fill().MinWidth(0)
	view.Children(func() {
		renderPane := func(i int) func() {
			return func() {
				column := ui.Column(c.Key(i)).Fill().MinWidth(0)
				column.Children(func() { panes[i].Render(c) })
				if !e.comparing && column.FocusWithin() {
					e.activeSide = i
				}
			}
		}
		e.inputSplitView(c.Key("inputs"), renderPane(0), renderPane(1)).Hidden(e.comparing)
		if !e.comparing && focus {
			panes[focusSide].Focus()
			e.activeSide = focusSide
		}
		if e.comparing {
			e.preview.Theme = editor.Theme(c.Theme())
			e.preview.Wrap = e.WrapEnabled()
			preview := mygoeditor.DiffView(c.Key("preview"), &e.preview).Fill().MinHeight(0).
				Radius(0, 0, 13, 13).Clip().Focusable().FocusRing(false).Label(locales.DiffPreviewLabel())
			if focus {
				preview.Focus()
			}
			if preview.FocusWithin() && !preview.Focused() {
				e.activeSide = int(e.preview.ActiveSide())
			}
		}
	})
	component.Toasts(c, component.ToastOptions{Bottom: true}).
		Margin(0, 0, max(4, c.Theme().Space(2)), 0).
		AttachTo(view, ui.AnchorBottom, ui.AnchorBottom)
}
