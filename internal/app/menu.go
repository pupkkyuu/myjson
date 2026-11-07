package app

import (
	"runtime"

	"github.com/egoist/mygo"

	"myjson/internal/locales"
)

func (app *Application) applicationMenu() *mygo.Menu {
	items := make([]*mygo.MenuItem, 0, 5)
	if runtime.GOOS == "darwin" {
		items = append(items, &mygo.MenuItem{Role: mygo.RoleAppMenu})
	}
	fileItems := []*mygo.MenuItem{
		{ID: "new-format", Label: locales.NewFormat(), Accelerator: "CmdOrCtrl+T", Click: func(*mygo.MenuItem, *mygo.Window) { app.menuNewEditor(false) }},
		{ID: "new-comparison", Label: locales.NewComparison(), Click: func(*mygo.MenuItem, *mygo.Window) { app.menuNewEditor(true) }},
		mygo.Separator(),
		{ID: "close", Label: locales.CloseToast(), Accelerator: "CmdOrCtrl+W", Click: func(_ *mygo.MenuItem, win *mygo.Window) { app.menuClose(win) }},
	}
	if runtime.GOOS != "darwin" {
		fileItems = append(fileItems, &mygo.MenuItem{Role: mygo.RoleQuit})
	}
	windowMenu := &mygo.MenuItem{Role: mygo.RoleWindowMenu}
	if runtime.GOOS != "darwin" {
		windowMenu.Submenu = []*mygo.MenuItem{
			{Role: mygo.RoleMinimize},
			{Label: locales.CloseWindow(), Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				if win != nil {
					win.Close()
				}
			}},
		}
	}
	items = append(items, []*mygo.MenuItem{
		{Label: locales.MenuFile(), Submenu: fileItems},
		{Role: mygo.RoleEditMenu},
		{Label: locales.MenuView(), Submenu: []*mygo.MenuItem{
			{ID: "toggle-sidebar", Label: locales.SidebarToggle(), Accelerator: "CmdOrCtrl+B", Click: func(*mygo.MenuItem, *mygo.Window) {
				if app.window != nil && !app.settingState.Open {
					app.sidebarOpen = !app.sidebarOpen
					app.window.Invalidate()
				}
			}},
			mygo.Separator(),
			{ID: "zoom-in", Label: locales.ZoomIn(), Accelerator: "CmdOrCtrl+=", Click: func(*mygo.MenuItem, *mygo.Window) { app.menuZoom(1) }},
			{ID: "zoom-out", Label: locales.ZoomOut(), Accelerator: "CmdOrCtrl+-", Click: func(*mygo.MenuItem, *mygo.Window) { app.menuZoom(-1) }},
			{ID: "zoom-reset", Label: locales.ZoomReset(), Accelerator: "CmdOrCtrl+0", Click: func(*mygo.MenuItem, *mygo.Window) { app.menuZoom(0) }},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		windowMenu,
	}...)
	return mygo.NewMenu(items)
}

func (app *Application) menuClose(win *mygo.Window) {
	if len(app.editors) > 1 && app.active != nil {
		app.closeEditor(app.active.ID())
		if app.window != nil {
			app.window.Invalidate()
		}
		return
	}
	if win == nil {
		win = app.window
	}
	if win != nil {
		win.Close()
	}
}

func (app *Application) menuNewEditor(comparison bool) {
	if app.settingState.Open {
		return
	}
	newWindow := app.window == nil || app.window.IsDestroyed()
	app.openWindow()
	if app.window == nil || app.window.IsDestroyed() {
		return
	}
	if comparison {
		app.createDiffEditor(app.window.Invalidate)
	} else if !newWindow {
		app.createFormatEditor(app.window.Invalidate)
	}
	app.window.Invalidate()
}

func (app *Application) menuZoom(delta int) {
	if delta == 0 {
		app.settingState.ResetZoom()
	} else {
		app.settingState.StepZoom(delta)
	}
	if app.window != nil && !app.window.IsDestroyed() {
		app.window.Invalidate()
	} else {
		app.flushSettings()
	}
}

// syncMenu 在设置面板打开时禁用相关菜单命令，仅向原生菜单提交变化。
func (app *Application) syncMenu() {
	if app.menu == nil {
		return
	}
	for _, id := range []string{"new-format", "new-comparison", "toggle-sidebar"} {
		item := app.menu.ItemByID(id)
		if item == nil {
			continue
		}
		enabled := !app.settingState.Open && (id != "toggle-sidebar" || app.window != nil)
		if item.IsEnabled() != enabled {
			item.SetEnabled(enabled)
		}
	}
}
