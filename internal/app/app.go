package app

import (
	"fmt"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/editor"
	"myjson/internal/settings"
)

type Application struct {
	window           *mygo.Window
	menu             *mygo.Menu
	theme            *ui.Theme
	sidebarOpen      bool
	sidebarWidth     float32
	settingState     settings.State
	editors          []editor.Editor
	active           editor.Editor
	next             uint64
	settingsPath     string
	lastSettings     settings.Saved
	settingsError    error
	settingsPending  bool
	settingsBlocked  bool
	settingsDeadline time.Time
	loadingEditorID  string
	loadingSince     time.Time
}

func newApplication() *Application {
	return &Application{sidebarOpen: true, sidebarWidth: defaultSidebarWidth}
}

func Run() error {
	app := newApplication()
	path, err := settings.ConfigPath()
	if err == nil {
		app.settingsPath = path
		var saved settings.Saved
		saved, err = settings.Load(path)
		if err == nil {
			app.settingState.Restore(saved)
			mygo.Theme.SetSource(saved.Theme)
		}
	}
	if err != nil {
		app.settingsError = fmt.Errorf("加载设置失败: %w", err)
	}
	app.lastSettings = app.settingState.Snapshot(mygo.Theme.Source())
	defer app.flushSettings()
	app.menu = app.applicationMenu()
	mygo.App.SetMenu(app.menu)
	off := app.configureLifecycle(runtime.GOOS)
	defer off()
	mygo.App.WhenReady(app.openWindow)
	return mygo.App.Run()
}

func (app *Application) configureLifecycle(platform string) func() {
	if platform != "darwin" {
		return func() {}
	}
	offClosed := mygo.App.OnWindowAllClosed(func() {})
	offActivate := mygo.App.OnActivate(app.activate)
	return func() {
		offActivate()
		offClosed()
	}
}

func (app *Application) activate(hasVisibleWindows bool) {
	if !hasVisibleWindows {
		app.openWindow()
	}
}
