package app

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor"
	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
	"myjson/internal/toolbar"
)

// cycleEditor 按侧栏顺序循环切换标签；无活动编辑器时按方向选择首项或末项。
func (app *Application) cycleEditor(direction int) {
	count := len(app.editors)
	if count < 2 {
		return
	}
	index := -1
	if app.active != nil {
		for i, e := range app.editors {
			if e.ID() == app.active.ID() {
				index = i
				break
			}
		}
	}
	if index == -1 && direction < 0 {
		index = 0
	}
	app.activateEditor(app.editors[(index+direction+count)%count].ID())
}

// selectEditor 切换活动文档，同时保留侧栏焦点。
func (app *Application) selectEditor(id string) {
	for _, e := range app.editors {
		if e.ID() == id {
			app.active = e
			return
		}
	}
}

func (app *Application) activateEditor(id string) {
	for _, e := range app.editors {
		if e.ID() != id {
			continue
		}
		app.active = e
		if focusable, ok := e.(interface{ Focus() }); ok {
			focusable.Focus()
		}
		return
	}
}

func (app *Application) nextEditorID() string {
	app.next++
	return "editor-" + strconv.FormatUint(app.next, 10)
}

func (app *Application) closeEditor(id string) {
	for i, e := range app.editors {
		if e.ID() != id {
			continue
		}
		wasActive := app.active != nil && app.active.ID() == id
		app.editors = slices.Delete(app.editors, i, i+1)
		if len(app.editors) == 0 {
			app.active = nil
		} else if wasActive {
			app.activateEditor(app.editors[min(i, len(app.editors)-1)].ID())
		}
		return
	}
}

func (app *Application) addFormatEditor(c *ui.Context) {
	app.createFormatEditor(c.Services().Invalidate)
}

func (app *Application) createFormatEditor(invalidate func()) {
	e := app.appendFormatEditor(invalidate)
	app.active = e
	e.Focus()
}

func (app *Application) appendFormatEditor(invalidate func()) *format.FormatEditor {
	e := format.NewFormatEditor(app.nextEditorID(), "", *app.settingState.Preferences())
	e.Invalidate = invalidate
	app.editors = append(app.editors, e)
	return e
}

func (app *Application) addDiffEditor(c *ui.Context) {
	app.createDiffEditor(c.Services().Invalidate)
}

func (app *Application) createDiffEditor(invalidate func()) {
	e := diff.NewDiffEditor(app.nextEditorID(), "", "", *app.settingState.Preferences())
	e.Invalidate = invalidate
	app.active = e
	app.editors = append(app.editors, e)
	e.Focus()
}

func (app *Application) applyToolbarAction(c *ui.Context, active editor.Editor, action toolbar.Action) {
	switch action {
	case toolbar.Format:
		active.Format()
	case toolbar.Compact:
		active.Compact()
	case toolbar.Escape:
		active.Escape()
	case toolbar.Unescape:
		active.Unescape()
	case toolbar.SortFields:
		active.SortFields()
	case toolbar.FoldAll:
		active.FoldAll()
	case toolbar.UnfoldAll:
		active.UnfoldAll()
	case toolbar.ToggleWrap:
		active.ToggleWrap()
	case toolbar.Search:
		active.Search()
	case toolbar.CopyAll:
		active.CopyAll(c.WriteClipboard)
	case toolbar.Clear:
		active.Clear()
	case toolbar.ToggleComparison, toolbar.PreviousDifference, toolbar.NextDifference:
		if comparison, ok := active.(editor.ComparisonEditor); ok {
			switch action {
			case toolbar.ToggleComparison:
				comparison.ToggleComparison()
			case toolbar.PreviousDifference:
				comparison.PreviousDifference()
			default:
				comparison.NextDifference()
			}
		}
	}
	c.Invalidate()
}
