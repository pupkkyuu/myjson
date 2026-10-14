package app

import (
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
	"myjson/internal/sidebar"
	"myjson/internal/toolbar"
)

func (app *Application) toolbar(c *ui.Context) {
	active := app.active
	options := toolbar.Options{}
	if active != nil {
		options.Title = active.Title()
		options.WrapEnabled = active.WrapEnabled()
		options.CopyEnabled = active.CopyEnabled()
		options.Loading = active.Loading()
		if options.Loading {
			if app.loadingEditorID != active.ID() {
				app.loadingEditorID = active.ID()
				app.loadingSince = c.Now()
			}
			const delay = 150 * time.Millisecond
			if remaining := delay - c.Now().Sub(app.loadingSince); remaining > 0 {
				c.After(remaining)
			} else {
				options.ShowLoading = true
			}
		} else {
			app.loadingEditorID = ""
		}
		switch e := active.(type) {
		case *format.FormatEditor:
			options.Kind = toolbar.FormatEditor
		case *diff.DiffEditor:
			options.Kind = toolbar.DiffEditor
			options.Comparing = e.Comparing()
			if options.Comparing {
				options.DifferenceCount = e.DifferenceCount()
			}
		}
	}
	if active == nil {
		app.loadingEditorID = ""
	}

	action := toolbar.View(c, options)
	if active == nil || action == toolbar.NoAction {
		return
	}
	app.applyToolbarAction(c, active, action)
}

func (app *Application) sidebar(c *ui.Context) {
	tabs := make([]sidebar.Tab, 0, len(app.editors))
	for _, e := range app.editors {
		tabs = append(tabs, sidebar.Tab{ID: e.ID(), Title: e.Title()})
	}
	var activeID string
	if app.active != nil {
		activeID = app.active.ID()
	}
	event := sidebar.View(c, sidebar.Options{
		Tabs: tabs, ActiveID: activeID, Disabled: app.settingState.Open, TopInset: headerHeight(c),
	})
	switch event.Action {
	case sidebar.NewFormat:
		app.addFormatEditor(c)
	case sidebar.NewComparison:
		app.addDiffEditor(c)
	case sidebar.ActivateTab:
		app.selectEditor(event.TabID)
	case sidebar.CloseTab:
		app.closeEditor(event.TabID)
	case sidebar.ToggleSidebar:
		app.sidebarOpen = !app.sidebarOpen
	default:
		return
	}
	c.Invalidate()
}

func (app *Application) sidebarFooter(c *ui.Context) {
	if sidebar.FooterView(c, app.settingState.Open) {
		app.settingState.Open = true
		c.Invalidate()
	}
}
