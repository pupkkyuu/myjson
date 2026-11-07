package app

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo"

	"myjson/internal/editor/diff"
)

func TestApplicationMenuCommands(t *testing.T) {
	app := newApplication()
	app.menu = app.applicationMenu()
	for _, tc := range []struct{ id, accelerator string }{
		{"new-format", "CmdOrCtrl+T"},
		{"new-comparison", ""},
		{"close", "CmdOrCtrl+W"},
		{"toggle-sidebar", "CmdOrCtrl+B"},
		{"zoom-in", "CmdOrCtrl+="},
		{"zoom-out", "CmdOrCtrl+-"},
		{"zoom-reset", "CmdOrCtrl+0"},
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
	for _, id := range []string{"new-format", "new-comparison", "toggle-sidebar"} {
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
