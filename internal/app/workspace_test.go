package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor"
	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func mountWorkspace(app *Application) *ui.Tester {
	return ui.NewTester(app.view, 1400, 700)
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func closeTab(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	r, ok := tt.Find(title)
	if !ok {
		t.Fatalf("未找到标签 %q", title)
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	click(t, tt, locales.CloseTab(title))
}

func TestWorkspaceCreationAndLastTabLifecycle(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	click(t, tt, locales.NewFormat())
	first := app.active
	click(t, tt, locales.NewComparison())
	second := app.active
	tt.Key(ui.Cmd, ui.KeyN)
	if len(app.editors) != 2 {
		t.Fatal("⌘N 不应再创建格式化标签")
	}
	tt.Key(ui.Cmd, ui.KeyT)
	if activeID(app) != "editor-3" || len(app.editors) != 3 {
		t.Fatal("⌘T 应创建第三个格式化标签")
	}
	closeTab(t, tt, app.active.Title())
	if app.active != second || len(app.editors) != 2 {
		t.Fatal("关闭非最后一个格式化标签后应选中左邻居，不应新建标签")
	}
	closeTab(t, tt, first.Title())
	if app.active != second || len(app.editors) != 1 {
		t.Fatal("关闭非活动的最后一个格式化标签后应只保留当前对比标签")
	}
	closeTab(t, tt, second.Title())
	if app.active != nil || len(app.editors) != 0 {
		t.Fatal("关闭最后一个标签后工作区应为空")
	}
	image := tt.Image()
	redPixels := 0
	for y := 200; y < 520; y++ {
		for x := 620; x < 960; x++ {
			pixel := image.RGBAAt(x, y)
			if pixel.R > 180 && int(pixel.R) > int(pixel.G)+35 && int(pixel.R) > int(pixel.B)+35 {
				redPixels++
			}
		}
	}
	if redPixels < 100 {
		t.Fatal("空工作区未显示背景插画")
	}
	tt.Key(ui.Cmd, ui.KeyT)
	if activeID(app) != "editor-4" || app.active.Title() != locales.FormatTabTitle("4") {
		t.Fatal("空工作区应能新建标签，且编号仍应递增")
	}
}

func TestSwitchingPreservesIndependentEditorState(t *testing.T) {
	app := workspaceWithTabs(1)
	first := app.editors[0].(*format.FormatEditor)
	second := diff.NewDiffEditor("editor-2", "{}", "[]")
	app.editors = append(app.editors, second)
	app.active = first
	tt := mountWorkspace(app)
	first.Buffer.Set(`{"a":1}`)
	first.Select(2, 4)
	first.Find("a")
	selection := first.Selections()[0]
	second.ToggleComparison()
	click(t, tt, second.Title())
	if app.active != second || !second.Comparing() {
		t.Fatal("切换标签应保留差异预览模式")
	}
	click(t, tt, first.Title())
	if first.Focused() || first.Buffer.String() != `{"a":1}` || first.Selections()[0] != selection {
		t.Fatalf("侧栏切换应保留侧栏焦点、文本及选区：焦点 %v，文本 %q，选区 %+v，期望 %+v", first.Focused(), first.Buffer.String(), first.Selections(), selection)
	}
	if count, _ := first.FindMatches(); count != 1 {
		t.Fatal("切换标签后应保留查找状态")
	}
	if !first.Buffer.Undo() || first.Buffer.String() != "{}" || !second.Comparing() {
		t.Fatal("撤销历史与比较模式应相互独立")
	}
}

func TestWorkspaceShortcutsRespectSettingsModal(t *testing.T) {
	app := workspaceWithTabs(3)
	app.active = app.editors[0]
	tt := mountWorkspace(app)
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	if activeID(app) != "editor-3" {
		t.Fatal("向左切换应从首项循环到末项")
	}
	tt.Key(ui.Ctrl, ui.KeyTab)
	if activeID(app) != "editor-1" {
		t.Fatal("向右切换应从末项循环到首项")
	}
	for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight} {
		tt.Key(ui.Cmd|ui.Alt, key)
		if activeID(app) != "editor-1" {
			t.Fatal("旧的切换快捷键不应再切换标签")
		}
	}
	click(t, tt, locales.Settings())
	if !app.settingState.Open {
		t.Fatal("底部设置入口未打开面板")
	}
	for _, shortcut := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{
		{ui.Cmd, ui.KeyT}, {ui.Ctrl | ui.Shift, ui.KeyTab}, {ui.Ctrl, ui.KeyTab},
	} {
		tt.Key(shortcut.mods, shortcut.key)
	}
	click(t, tt, locales.NewFormat())
	if len(app.editors) != 3 || activeID(app) != "editor-1" {
		t.Fatal("设置面板打开时应阻止侧栏及工作区快捷键")
	}
	tt.Key(0, ui.KeyEscape)
	if app.settingState.Open {
		t.Fatal("Esc 应关闭设置")
	}
	tt.Key(ui.Ctrl, ui.KeyTab)
	if activeID(app) != "editor-2" {
		t.Fatal("关闭设置后应恢复切换快捷键")
	}
	tt.Key(ui.Cmd, ui.KeyComma)
	if !app.settingState.Open {
		t.Fatal("⌘, 应打开设置")
	}
}

func TestToolbarRoutesDocumentTransforms(t *testing.T) {
	for _, tc := range []struct{ name, label, source, want string }{
		{"格式化", locales.FormatJSON(), `{"a":1}`, "{\n  \"a\": 1\n}"},
		{"压缩", locales.Compact(), "{ \"a\": 1 }", `{"a":1}`},
		{"转义", locales.Escape(), "中文", `"中文"`},
		{"去转义", locales.Unescape(), `"{}"`, "{}"},
		{"排序", locales.SortFields(), `{"b":1,"a":2}`, "{\n  \"a\": 2,\n  \"b\": 1\n}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := workspaceWithTabs(1)
			e := app.editors[0].(*format.FormatEditor)
			e.Buffer.Set(tc.source)
			app.active = e
			wake := make(chan struct{}, 8)
			e.Invalidate = func() { wake <- struct{}{} }
			tt := mountWorkspace(app)
			click(t, tt, tc.label)
			select {
			case <-wake:
			case <-time.After(5 * time.Second):
				t.Fatal("工具栏操作未完成后台变换")
			}
			tt.Frame()
			if e.Buffer.String() != tc.want {
				t.Fatalf("工具栏提交结果 = %q，期望 %q", e.Buffer.String(), tc.want)
			}
		})
	}
}

func TestToolbarLoadingStateFollowsActiveEditorAndCompletion(t *testing.T) {
	for _, kind := range []string{"format", "diff"} {
		t.Run(kind, func(t *testing.T) {
			app := newApplication()
			wake := make(chan struct{}, 8)
			var document editor.Editor
			if kind == "format" {
				e := format.NewFormatEditor("editor-1", `{"a":1}`)
				e.Invalidate = func() { wake <- struct{}{} }
				document = e
			} else {
				e := diff.NewDiffEditor("editor-1", `{"a":1}`, `{"a":2}`)
				e.Invalidate = func() { wake <- struct{}{} }
				document = e
			}
			app.active = document
			consumeResult := false
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					ui.Row(c).FillWidth().Height(48).Children(func() { app.toolbar(c) })
					if consumeResult {
						ui.Column(c).FillWidth().Grow(1).Children(func() { document.View(c) })
					}
				})
			}, 1400, 700)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
			document.Format()
			tt.Frame()
			if !document.Loading() {
				t.Fatal("后台变换期间应保持忙碌状态")
			}
			idle := format.NewFormatEditor("editor-2", `{"b":2}`)
			idleWake := make(chan struct{}, 1)
			idle.Invalidate = func() { idleWake <- struct{}{} }
			app.active = idle
			tt.Frame()
			click(t, tt, locales.FormatJSON())
			if !idle.Loading() {
				t.Fatal("切换到空闲文档后变换按钮应可用")
			}
			select {
			case <-idleWake:
			case <-time.After(5 * time.Second):
				t.Fatal("切换后的文档未完成后台变换")
			}
			app.active = document
			tt.Frame()
			if !document.Loading() {
				t.Fatal("切换文档不应清除原文档的忙碌状态")
			}
			select {
			case <-wake:
			case <-time.After(5 * time.Second):
				t.Fatal("后台变换未完成")
			}
			consumeResult = true
			tt.Frame()
			if document.Loading() {
				t.Fatal("消费结果后未清除忙碌状态")
			}
			consumeResult = false
			click(t, tt, locales.Compact())
			if !document.Loading() {
				t.Fatal("消费结果后工具栏变换按钮未恢复")
			}
			select {
			case <-wake:
			case <-time.After(5 * time.Second):
				t.Fatal("恢复后的工具栏操作未完成后台变换")
			}
			consumeResult = true
			tt.Frame()
			if document.Loading() {
				t.Fatal("再次消费结果后未清除忙碌状态")
			}
		})
	}
}

func TestToolbarShowsDelayedProgressForActiveDocument(t *testing.T) {
	for _, kind := range []string{"format", "diff"} {
		t.Run(kind, func(t *testing.T) {
			app := newApplication()
			var busy editor.Editor
			if kind == "format" {
				e := format.NewFormatEditor("editor-1", `{"a":1}`)
				e.Invalidate = func() {}
				busy = e
			} else {
				e := diff.NewDiffEditor("editor-1", `{"a":1}`, `{"a":2}`)
				e.Invalidate = func() {}
				busy = e
			}
			app.active = busy
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).FillWidth().Height(48).Children(func() { app.toolbar(c) })
			}, 1400, 100)
			busy.Format()
			tt.Frame()
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("短任务不应立即显示进度提示")
			}
			tt.Advance(149 * time.Millisecond)
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("延迟结束前不应显示进度提示")
			}
			tt.Advance(time.Millisecond)
			if _, ok := tt.Find(locales.Transforming()); !ok {
				t.Fatal("长任务应显示进度提示")
			}
			busy.Clear()
			tt.Frame()
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("取消任务后应立即隐藏提示")
			}
			busy.Format()
			tt.Frame()
			app.active = format.NewFormatEditor("editor-2", "{}")
			tt.Frame()
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("切换到空闲文档后不应显示进度提示")
			}
			app.active = busy
			tt.Frame()
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("切回忙碌文档时应重新等待延迟")
			}
			busy.Clear()
			tt.Advance(150 * time.Millisecond)
			if _, ok := tt.Find(locales.Transforming()); ok {
				t.Fatal("任务取消后不应显示进度提示")
			}
		})
	}
}

func TestToolbarRoutesCopyClearWrapAndSearch(t *testing.T) {
	app := workspaceWithTabs(1)
	e := app.editors[0].(*format.FormatEditor)
	app.active = e
	tt := mountWorkspace(app)
	e.Select(0, 1)
	selection := e.Selections()[0]
	click(t, tt, locales.Copy())
	if tt.Clipboard() != "{}" || e.Selections()[0] != selection {
		t.Fatal("复制工具应复制全文并保留选区")
	}
	click(t, tt, locales.SoftWrap())
	if !e.WrapEnabled() {
		t.Fatal("换行工具未切换状态")
	}
	e.Buffer.Set("{\n  \"a\": 1\n}")
	tt.Frame()
	click(t, tt, locales.FoldAll())
	if len(e.Folded()) == 0 {
		t.Fatal("折叠工具未作用于当前文档")
	}
	click(t, tt, locales.UnfoldAll())
	if len(e.Folded()) != 0 {
		t.Fatal("展开工具未恢复折叠内容")
	}
	click(t, tt, locales.Search())
	if _, ok := tt.Find(locales.FindQuery()); !ok {
		t.Fatal("搜索工具未打开查找栏")
	}
	click(t, tt, locales.Clear())
	if e.Buffer.Len() != 0 || !e.Buffer.Undo() || e.Buffer.String() != "{\n  \"a\": 1\n}" {
		t.Fatal("清空工具应允许撤销")
	}
}

func TestDiffToolbarSwitchingAndCopyDisabled(t *testing.T) {
	e := diff.NewDiffEditor("editor-1", "{}", "[]")
	app := newApplication()
	app.active = e
	tt := mountWorkspace(app)
	tt.SetClipboard("保留剪贴板")
	click(t, tt, locales.Copy())
	if tt.Clipboard() != "保留剪贴板" {
		t.Fatal("对比编辑器不应启用全文复制工具")
	}
	click(t, tt, locales.CompareJSON())
	if !e.Comparing() {
		t.Fatal("对比工具未进入预览")
	}
	for _, label := range []string{locales.PreviousDifference(), locales.NextDifference()} {
		click(t, tt, label)
	}
	click(t, tt, locales.FormatJSON())
	if e.Loading() {
		t.Fatal("预览模式不应执行格式化")
	}
	click(t, tt, locales.EditJSON())
	if e.Comparing() {
		t.Fatal("编辑工具未返回输入模式")
	}
	click(t, tt, locales.Clear())
	if e.Comparing() {
		t.Fatal("清空应保留输入模式")
	}
}

type namedDocument struct {
	editor.Editor
	title string
}

func (e *namedDocument) Title() string { return e.title }

func sidebarDocuments(names ...string) *Application {
	app := newApplication()
	for i, title := range names {
		app.next++
		app.editors = append(app.editors, &namedDocument{
			Editor: format.NewFormatEditor(fmt.Sprintf("editor-%d", i+1), "{}"), title: title,
		})
	}
	return app
}

func TestSidebarDispatchesUserActions(t *testing.T) {
	app := sidebarDocuments("文档一", "文档二")
	tt := ui.NewTester(app.sidebar, 250, 600)
	click(t, tt, "文档二")
	if activeID(app) != "editor-2" {
		t.Fatal("点击标签未激活对应编辑器")
	}
	r, _ := tt.Find("文档一")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	click(t, tt, locales.CloseTab("文档一"))
	if len(app.editors) != 1 || activeID(app) != "editor-2" {
		t.Fatal("关闭非活动标签改变了活动编辑器")
	}
	click(t, tt, locales.NewFormat())
	if _, ok := app.active.(*format.FormatEditor); !ok || len(app.editors) != 2 {
		t.Fatal("新建格式化未进入工作区")
	}
	click(t, tt, locales.NewComparison())
	if len(app.editors) != 3 || activeID(app) != "editor-4" {
		t.Fatal("新建对比未进入工作区")
	}
}
