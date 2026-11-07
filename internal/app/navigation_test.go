package app

import (
	"fmt"
	"testing"

	"myjson/internal/editor"
	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
)

func workspaceWithTabs(count int) *Application {
	app := newApplication()
	for id := uint64(1); id <= uint64(count); id++ {
		app.editors = append(app.editors, format.NewFormatEditor(fmt.Sprintf("editor-%d", id), "{}"))
	}
	app.next = uint64(count)
	return app
}

func activeID(app *Application) string {
	if app.active == nil {
		return ""
	}
	return app.active.ID()
}

func TestCloseTabNeighborPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		count                 int
		active, closing, want uint64
		remaining             int
	}{
		{"空工作区", 0, 0, 1, 0, 0},
		{"唯一标签", 1, 1, 1, 0, 0},
		{"首项选择右侧", 3, 1, 1, 2, 2},
		{"中间项选择右侧", 3, 2, 2, 3, 2},
		{"末项选择左侧", 3, 3, 3, 2, 2},
		{"关闭非活动项", 3, 2, 1, 2, 2},
		{"未知标签", 3, 2, 99, 2, 3},
		{"没有活动项", 3, 0, 2, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := workspaceWithTabs(tc.count)
			app.activateEditor(testEditorID(tc.active))
			app.closeEditor(testEditorID(tc.closing))
			if activeID(app) != testEditorID(tc.want) || len(app.editors) != tc.remaining {
				t.Fatalf("关闭后活动项 = %s，剩余 %d；期望 %s，%d", activeID(app), len(app.editors), testEditorID(tc.want), tc.remaining)
			}
			for _, e := range app.editors {
				if e.ID() == testEditorID(tc.closing) {
					t.Fatal("已关闭的标签仍在列表中")
				}
			}
		})
	}
}

func TestClosingLastFormatPreservesComparison(t *testing.T) {
	app := workspaceWithTabs(1)
	comparison := diff.NewDiffEditor("editor-2", "{}", "[]")
	app.next = 2
	app.editors = append(app.editors, comparison)
	app.activateEditor("editor-1")
	app.closeEditor("editor-1")
	if len(app.editors) != 1 || app.editors[0] != comparison || app.active != comparison {
		t.Fatalf("关闭最后一个格式化标签时应只保留并激活对比标签：%v", app.editors)
	}
	app.closeEditor(comparison.ID())
	if len(app.editors) != 0 || app.active != nil {
		t.Fatal("关闭最后一个对比标签后工作区应为空")
	}
}

func TestCycleTabBoundaryPolicy(t *testing.T) {
	for _, tc := range []struct {
		count     int
		active    uint64
		direction int
		want      uint64
	}{
		{0, 0, 1, 0}, {0, 0, -1, 0}, {1, 1, 1, 1}, {1, 1, -1, 1},
		{3, 1, -1, 3}, {3, 3, 1, 1}, {3, 2, 1, 3}, {3, 2, -1, 1},
		{3, 0, 1, 1}, {3, 0, -1, 3}, {3, 99, 1, 1}, {3, 99, -1, 3},
	} {
		t.Run(fmt.Sprintf("数量%d/活动%d/方向%d", tc.count, tc.active, tc.direction), func(t *testing.T) {
			app := workspaceWithTabs(tc.count)
			app.activateEditor(testEditorID(tc.active))
			if tc.active == 99 {
				app.active = format.NewFormatEditor("editor-99", "")
			}
			app.cycleEditor(tc.direction)
			if activeID(app) != testEditorID(tc.want) || len(app.editors) != tc.count {
				t.Fatalf("切换后活动项 = %s，数量 %d；期望活动项 %s", activeID(app), len(app.editors), testEditorID(tc.want))
			}
		})
	}
}

func TestActivateUnknownTabPreservesWorkspace(t *testing.T) {
	app := workspaceWithTabs(2)
	app.activateEditor("editor-1")
	before := app.active
	app.activateEditor("editor-100")
	if app.active != before || len(app.editors) != 2 {
		t.Fatal("激活未知标签不应改变工作区")
	}
}

type customEditor struct {
	editor.Editor
	focused int
}

func (e *customEditor) Focus() { e.focused++ }

func TestNavigationSupportsCustomEditors(t *testing.T) {
	custom := &customEditor{Editor: format.NewFormatEditor("editor-9", "{}")}
	app := workspaceWithTabs(1)
	app.editors = append(app.editors, custom)
	app.activateEditor("editor-1")
	app.cycleEditor(1)
	if app.active != custom || custom.focused != 1 {
		t.Fatal("切换标签应通过可选 Focus 接口聚焦自定义编辑器")
	}
}

func testEditorID(number uint64) string {
	if number == 0 {
		return ""
	}
	return fmt.Sprintf("editor-%d", number)
}
