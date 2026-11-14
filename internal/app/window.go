package app

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (app *Application) openWindow() {
	if app.window != nil && !app.window.IsDestroyed() {
		if app.window.IsMinimized() {
			app.window.Restore()
		}
		app.window.Show()
		return
	}
	app.window = mygo.NewWindow(app.windowOptions(runtime.GOOS))
	app.createFormatEditor(app.window.Invalidate)
	app.window.Invalidate()
	app.window.OnMaximize(app.window.Invalidate)
	app.window.OnUnmaximize(app.window.Invalidate)
	app.window.OnClosed(func() {
		app.flushSettings()
		app.window = nil
		app.editors = nil
		app.active = nil
		app.settingState.Open = false
		app.syncMenu()
	})
	app.syncMenu()
}

func (app *Application) windowOptions(platform string) mygo.WindowOptions {
	options := mygo.WindowOptions{
		Title: "MyJson", Width: 1000, Height: 650, MinWidth: 760, MinHeight: 400,
		TitleBarStyle: mygo.TitleBarHidden,
		Content:       ui.View(app.view),
	}
	switch platform {
	case "darwin":
		options.Transparent = true
		// 关闭按钮 16 DIP 高，顶边在 20：中心落在 trafficMid，和侧栏开关同一条线。
		options.TrafficLightPosition = &mygo.Point{X: 16, Y: int(trafficMid) - 8}
		// sidebar 材质只模糊壁纸，没有 Liquid Glass 那圈边缘透镜。
		options.Vibrancy = mygo.VibrancySidebar
	case "windows":
		options.HideWindowButtons = true
		options.Vibrancy = mygo.VibrancyMica
	}
	return options
}
