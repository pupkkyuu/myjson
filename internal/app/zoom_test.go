package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func TestWindowZoomShortcutsAndSettings(t *testing.T) {
	app := workspaceWithTabs(1)
	app.active = app.editors[0]
	ed := app.active.(*format.FormatEditor)
	ed.Buffer.Set(`{"value":1}`)
	ed.Focus()
	var renderedZoom float32
	tt := ui.NewTester(func(c *ui.Context) {
		app.frame(c, "darwin")
		renderedZoom = c.Zoom()
	}, 1000, 650)
	check := func(want int) {
		t.Helper()
		if got := app.settingState.ZoomPercent(); got != want || renderedZoom != float32(want)/100 {
			t.Fatalf("缩放 = %d%%，渲染 = %v，期望 %d%%", got, renderedZoom, want)
		}
		if ed.Buffer.String() != `{"value":1}` {
			t.Fatal("界面缩放不应修改文档")
		}
	}
	check(100)
	if !ed.Focused() {
		t.Fatal("测试需要编辑器获得焦点")
	}
	tt.Key(ui.Cmd, ui.KeyEqual)
	check(110)
	if !ed.Focused() {
		t.Fatal("缩放不应改变编辑器焦点")
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyEqual)
	check(125)
	tt.Key(ui.Cmd, ui.KeyMinus)
	check(110)
	tt.Key(ui.Cmd, ui.Key0)
	check(100)

	click(t, tt, locales.Settings())
	tt.Key(ui.Cmd, ui.KeyEqual)
	check(110)
	clickPreferenceControl(t, tt, ui.RolePopUpButton, locales.InterfaceZoom())
	click(t, tt, "125%")
	check(125)
	tt.Key(ui.Cmd, ui.Key0)
	check(100)
	if !app.settingState.Open {
		t.Fatal("调整缩放不应关闭设置")
	}
}

func TestWindowZoomBounds(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	for range 20 {
		tt.Key(ui.Cmd, ui.KeyMinus)
	}
	if app.settingState.ZoomPercent() != 75 {
		t.Fatal("缩放应限制在 75%")
	}
	for range 20 {
		tt.Key(ui.Cmd, ui.KeyEqual)
	}
	if app.settingState.ZoomPercent() != 200 {
		t.Fatal("缩放应限制在 200%")
	}
}
