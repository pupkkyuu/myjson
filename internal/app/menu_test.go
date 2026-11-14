package app

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo"

	"myjson/internal/editor/diff"
	"myjson/internal/locales"
)

func runMenuCommand(t *testing.T, app *Application, id string) {
	t.Helper()
	if app.menu == nil {
		app.menu = app.applicationMenu()
	}
	app.syncMenu()
	item := app.menu.ItemByID(id)
	if item == nil || item.Click == nil {
		t.Fatalf("菜单命令 %s 不存在", id)
	}
	item.Click(item, app.window)
}

func TestApplicationMenuCommands(t *testing.T) {
	app := newApplication()
	app.menu = app.applicationMenu()
	previousTabAccelerator := "Ctrl+Shift+Tab"
	if runtime.GOOS == "darwin" {
		previousTabAccelerator = "Ctrl+Shift+\x19"
	}
	for _, tc := range []struct{ id, accelerator string }{
		{"new-format", "CmdOrCtrl+T"},
		{"new-comparison", ""},
		{"close", "CmdOrCtrl+W"},
		{"toggle-sidebar", "CmdOrCtrl+B"},
		{"zoom-in", "CmdOrCtrl+="},
		{"zoom-out", "CmdOrCtrl+-"},
		{"zoom-reset", "CmdOrCtrl+0"},
		{"previous-tab", previousTabAccelerator},
		{"next-tab", "Ctrl+Tab"},
	} {
		item := app.menu.ItemByID(tc.id)
		if item == nil || item.Label == "" || item.Icon != nil || item.Click == nil || item.Accelerator != tc.accelerator {
			t.Fatalf("menu item %s has an unexpected icon or is missing its label, action, or accelerator", tc.id)
		}
	}
	items := app.menu.Items()
	if runtime.GOOS == "darwin" {
		if len(items) != 5 || items[0].Role != mygo.RoleAppMenu {
			t.Fatal("macOS application menu must remain available")
		}
		items = items[1:]
	}
	if len(items) != 4 || items[1].Role != mygo.RoleEditMenu || items[3].Role != mygo.RoleWindowMenu {
		t.Fatal("application, editing, and window roles must remain available")
	}
	for _, id := range []string{"previous-tab", "next-tab"} {
		found := false
		for _, item := range items[3].Submenu {
			found = found || item.ID == id
		}
		if !found {
			t.Fatalf("窗口菜单缺少 %s", id)
		}
	}
}

func TestTabMenuCommandsAndModalState(t *testing.T) {
	app := workspaceWithTabs(3)
	app.activateEditor("editor-1")
	for _, tc := range []struct{ command, want string }{
		{"previous-tab", "editor-3"}, {"next-tab", "editor-1"}, {"next-tab", "editor-2"},
	} {
		runMenuCommand(t, app, tc.command)
		if activeID(app) != tc.want {
			t.Fatalf("%s 后活动标签 = %s，期望 %s", tc.command, activeID(app), tc.want)
		}
	}
	app.settingState.Open = true
	for _, id := range []string{"previous-tab", "next-tab"} {
		runMenuCommand(t, app, id)
		if app.menu.ItemByID(id).IsEnabled() || activeID(app) != "editor-2" {
			t.Fatal("设置打开时应禁用切换菜单及其回调")
		}
	}
	app.settingState.Open = false
	runMenuCommand(t, app, "next-tab")
	if activeID(app) != "editor-3" || !app.menu.ItemByID("next-tab").IsEnabled() {
		t.Fatal("关闭设置后应恢复菜单切换")
	}
	for len(app.editors) > 0 {
		app.closeEditor(app.editors[0].ID())
		app.syncMenu()
		for _, id := range []string{"previous-tab", "next-tab"} {
			if got, want := app.menu.ItemByID(id).IsEnabled(), len(app.editors) > 1; got != want {
				t.Fatalf("剩余 %d 个标签时 %s 可用状态 = %v，期望 %v", len(app.editors), id, got, want)
			}
		}
	}
}

func TestMenuCloseRemovesActiveTabWithoutReplacement(t *testing.T) {
	app := workspaceWithTabs(1)
	comparison := diff.NewDiffEditor("editor-2", "{}", "[]")
	app.editors = append(app.editors, comparison)
	app.activateEditor("editor-1")
	item := app.applicationMenu().ItemByID("close")
	item.Click(item, nil)
	if len(app.editors) != 1 || app.active != comparison {
		t.Fatal("关闭格式化标签后应只保留并激活对比标签")
	}
}

func TestMenuCloseDismissesSettingsBeforeDocuments(t *testing.T) {
	for _, tc := range []struct {
		name string
		tabs int
	}{
		{"empty workspace", 0}, {"last tab", 1}, {"multiple tabs", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := workspaceWithTabs(tc.tabs)
			if tc.tabs > 0 {
				app.activateEditor("editor-1")
			}
			active := app.active
			app.menu = app.applicationMenu()
			app.settingState.Open = true
			app.syncMenu()
			item := app.menu.ItemByID("close")
			if !item.IsEnabled() {
				t.Fatal("关闭命令应允许关闭设置弹窗")
			}
			if item.Label != locales.CloseSettings() {
				t.Fatal("设置打开时关闭菜单应明确显示关闭设置")
			}
			item.Click(item, nil)
			if app.settingState.Open {
				t.Fatal("关闭命令应先关闭设置弹窗")
			}
			if len(app.editors) != tc.tabs || app.active != active {
				t.Fatal("关闭设置弹窗不应关闭或切换后面的标签页")
			}
			if !app.menu.ItemByID("new-format").IsEnabled() {
				t.Fatal("关闭设置弹窗后应恢复文档菜单命令")
			}
			if item.Label != locales.CloseToast() {
				t.Fatal("设置关闭后应恢复关闭菜单文案")
			}
			if tc.tabs > 1 {
				item.Click(item, nil)
				if len(app.editors) != tc.tabs-1 || app.active == active {
					t.Fatal("设置关闭后再次执行关闭命令应正常关闭活动标签页")
				}
			}
		})
	}
}

func TestApplicationMenuZoomAndModalState(t *testing.T) {
	app := newApplication()
	app.menu = app.applicationMenu()
	for _, tc := range []struct {
		id   string
		want int
	}{
		{"zoom-in", 110}, {"zoom-in", 125}, {"zoom-out", 110}, {"zoom-reset", 100},
	} {
		item := app.menu.ItemByID(tc.id)
		if item == nil || item.Click == nil {
			t.Fatalf("menu item %s is missing or has no action", tc.id)
		}
		item.Click(item, nil)
		if got := app.settingState.ZoomPercent(); got != tc.want {
			t.Fatalf("%s: zoom %d, want %d", tc.id, got, tc.want)
		}
	}
	app.settingState.Open = true
	app.syncMenu()
	for _, id := range []string{"new-format", "new-comparison", "toggle-sidebar", "previous-tab", "next-tab"} {
		item := app.menu.ItemByID(id)
		if item == nil {
			t.Fatalf("menu item %s is missing", id)
		}
		if item.IsEnabled() {
			t.Fatalf("%s remains enabled while settings are open", id)
		}
	}
	app.settingState.Open = false
	app.syncMenu()
	for _, id := range []string{"new-format", "new-comparison"} {
		item := app.menu.ItemByID(id)
		if item == nil || !item.IsEnabled() {
			t.Fatalf("menu item %s should be enabled after settings close", id)
		}
	}
}

func TestSyncMenuWithMissingItems(t *testing.T) {
	app := newApplication()
	app.menu = mygo.NewMenu([]*mygo.MenuItem{{ID: "new-format", Label: "New"}})
	app.settingState.Open = true
	app.syncMenu()
	item := app.menu.ItemByID("new-format")
	if item == nil || item.IsEnabled() {
		t.Fatal("现有菜单项应随设置面板禁用")
	}
}
