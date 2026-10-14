package app

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func TestWindowOptionsRetainNativeChrome(t *testing.T) {
	for _, platform := range []string{"windows", "darwin", "linux"} {
		options := newApplication().windowOptions(platform)
		if options.Width != 1000 || options.Height != 650 || options.MinWidth != 760 || options.MinHeight != 400 ||
			options.Frameless || options.TitleBarStyle != mygo.TitleBarHidden {
			t.Fatalf("窗口参数不符合应用约定：%+v", options)
		}
		if options.HideWindowButtons != (platform == "windows") {
			t.Fatalf("%s 的系统按钮隐藏状态错误", platform)
		}
		if platform == "darwin" && (!options.Transparent || options.TrafficLightPosition == nil || *options.TrafficLightPosition != (mygo.Point{X: 16, Y: 20})) {
			t.Fatal("macOS 交通灯配置丢失")
		}
	}
}

func TestWorkspaceHeaderGeometry(t *testing.T) {
	for _, platform := range []string{"windows", "darwin", "linux"} {
		for _, width := range []int{760, 1000} {
			for _, sidebarWidth := range []float32{180, 320} {
				t.Run(fmt.Sprintf("%s/%d/%g", platform, width, sidebarWidth), func(t *testing.T) {
					app := workspaceWithTabs(1)
					app.active = app.editors[0]
					app.sidebarWidth = sidebarWidth
					tt := ui.NewTester(func(c *ui.Context) { app.frame(c, platform) }, width, 650)
					tt.SetPreferences(ui.Preferences{ReduceMotion: true})
					if platform == "darwin" {
						tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
					}
					for _, open := range []bool{true, false} {
						app.sidebarOpen = open
						tt.Frame()
						tools, ok := tt.Find(locales.EditorToolbar())
						if !ok || tools.W <= 0 || tools.X < 0 || tools.X+tools.W > float32(width) {
							t.Fatalf("工具栏越界：%+v", tools)
						}
						toggle, ok := tt.Find(locales.SidebarToggle())
						if !ok || math.Abs(float64(tools.Y+tools.H/2-toggle.Y-toggle.H/2)) > 1 {
							t.Fatalf("工具栏与侧栏开关的中心线不一致：%+v / %+v", tools, toggle)
						}
						if math.Abs(float64(toggle.Y+toggle.H/2-28)) > 1 {
							t.Fatalf("侧栏开关中心线应位于 28 DIP：%+v", toggle)
						}
						formatButton, ok := tt.Find(locales.FormatJSON())
						if !ok || math.Abs(float64(formatButton.Y+formatButton.H/2-28)) > 1 || math.Abs(float64(formatButton.Y-(cardGap+4))) > 1 {
							t.Fatalf("工具栏按钮应居中并与卡片顶部留出间距：%+v", formatButton)
						}
						if platform == "windows" {
							minimize, ok := tt.Find(locales.MinimizeWindow())
							closeButton, closeOK := tt.Find(locales.CloseWindow())
							if !ok || !closeOK || tools.X+tools.W > minimize.X || closeButton.X+closeButton.W > float32(width) {
								t.Fatalf("工具栏与窗口按钮重叠：%+v / %+v / %+v", tools, minimize, closeButton)
							}
							if math.Abs(float64(tools.Y+tools.H/2-minimize.Y-minimize.H/2)) > 1 {
								t.Fatalf("工具栏与窗口按钮的中心线不一致：%+v / %+v", tools, minimize)
							}
						}
						if platform == "darwin" {
							if toggle.X < 96 {
								t.Fatalf("侧栏开关未避让交通灯：%+v", toggle)
							}
						}
					}
					if directory := os.Getenv("MYJSON_TEST_SCREENSHOTS"); directory != "" && width == 1000 && sidebarWidth == 180 {
						app.sidebarOpen = true
						tt.Frame()
						saveFrame(t, tt, directory, platform+"-editor.png")
					}
				})
			}
		}
	}
}

func TestToolbarKeyboardFocusRingIsNotClipped(t *testing.T) {
	app := workspaceWithTabs(1)
	app.active = app.editors[0]
	tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "darwin") }, 1200, 650)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
	before := tt.Image()
	for i := 0; i < 100 && !tt.Focused(locales.Search()); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused(locales.Search()) {
		t.Fatal("无法通过键盘聚焦搜索按钮")
	}
	r, ok := tt.Find(locales.Search())
	if !ok {
		t.Fatal("未找到搜索按钮")
	}
	after := tt.Image()
	x, y := int(r.X+r.W/2), int(r.Y-2)
	if before.RGBAAt(x, y) == after.RGBAAt(x, y) {
		t.Fatal("搜索按钮上方的键盘焦点环被裁切")
	}
}

func TestDocumentTitleUsesAvailableLeftEdge(t *testing.T) {
	app := workspaceWithTabs(1)
	app.active = app.editors[0]
	progress := float32(1)
	tt := ui.NewTester(func(c *ui.Context) {
		app.documentCard(c, "darwin", progress, 180)
	}, 1000, 650)
	tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
	title, ok := tt.Find(app.active.Title())
	if !ok || title.X > 24 {
		t.Fatalf("侧栏展开时标题应靠近卡片左边缘：%+v", title)
	}
	progress = 0
	tt.Frame()
	title, ok = tt.Find(app.active.Title())
	if !ok || title.X < 128 {
		t.Fatalf("侧栏收起时标题应避让交通灯及侧栏开关：%+v", title)
	}
}

func saveFrame(t *testing.T, tt *ui.Tester, directory, name string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, tt.Image())
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("保存布局截图失败：%v / %v", err, closeErr)
	}
}

func TestSidebarResizeAndCollapseKeepEditorState(t *testing.T) {
	app := workspaceWithTabs(1)
	e := app.editors[0].(*format.FormatEditor)
	app.active = e
	e.Buffer.Set(`{"a":1}`)
	e.Select(2, 4)
	e.Find("a")
	selection := e.Selections()[0]
	tt := mountWorkspace(app)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	click(t, tt, locales.SidebarToggle())
	if app.sidebarOpen {
		t.Fatal("侧栏开关未关闭侧栏")
	}
	tt.Key(ui.Cmd, ui.KeyB)
	if !app.sidebarOpen || app.sidebarWidth != defaultSidebarWidth {
		t.Fatal("快捷键未恢复侧栏")
	}
	tt.SetDark(true)
	if e.Buffer.String() != `{"a":1}` || e.Selections()[0] != selection {
		t.Fatal("布局或主题变化重置了编辑器状态")
	}
	if count, _ := e.FindMatches(); count != 1 || !e.Buffer.Undo() || e.Buffer.String() != "{}" {
		t.Fatal("布局变化重置了搜索或撤销历史")
	}
}

func TestSidebarAnimationAndReducedMotion(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	click(t, tt, locales.SidebarToggle())
	tt.Advance(100 * time.Millisecond)
	if r, ok := tt.Find(locales.TabList()); !ok || r.X >= 180 {
		t.Fatal("折叠动画提前移除了侧栏")
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if _, ok := tt.Find(locales.TabList()); ok {
		t.Fatal("减少动画偏好未立即完成折叠")
	}
	click(t, tt, locales.SidebarToggle())
	tt.Advance(time.Millisecond)
	if _, ok := tt.Find(locales.TabList()); !ok {
		t.Fatal("减少动画偏好下未展开侧栏")
	}
}

func TestSidebarToggleStaysFixedDuringAnimation(t *testing.T) {
	for _, platform := range []string{"darwin", "windows", "linux"} {
		t.Run(platform, func(t *testing.T) {
			app := workspaceWithTabs(1)
			probe := &layoutProbeEditor{Editor: app.editors[0]}
			app.active, app.editors = probe, nil // 标题只出现在工具栏，便于测量。
			tt := ui.NewTester(func(c *ui.Context) { app.frame(c, platform) }, 1000, 650)
			if platform == "darwin" {
				tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
			}
			_, ok := tt.Find(locales.SidebarToggle())
			if !ok {
				t.Fatal("未找到侧栏开关")
			}
			fixedX := float32(8)
			if platform == "darwin" {
				fixedX = 96
			}
			check := func() ui.Rect {
				t.Helper()
				button, ok := tt.Find(locales.SidebarToggle())
				if !ok || math.Abs(float64(button.X-fixedX)) > 0.1 {
					t.Fatalf("侧栏开关应保持固定位置 %g：%+v", fixedX, button)
				}
				title, ok := tt.Find(probe.Title())
				if !ok || title.X < button.X+button.W {
					t.Fatalf("工具栏标题与侧栏开关重叠：%+v / %+v", title, button)
				}
				editor, ok := tt.Find("editor-region")
				if !ok {
					t.Fatal("未找到编辑区域")
				}
				return editor
			}
			opened := check()
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Advance(80 * time.Millisecond)
			middle := check()
			tt.Advance(3 * time.Second)
			closed := check()
			if !(opened.X > middle.X && middle.X > closed.X && opened.W < middle.W && middle.W < closed.W) {
				t.Fatalf("收起时编辑区应逐渐扩大：%+v -> %+v -> %+v", opened, middle, closed)
			}
			click(t, tt, locales.SidebarToggle())
			tt.Advance(80 * time.Millisecond)
			reopening := check()
			tt.Advance(3 * time.Second)
			reopened := check()
			if !(closed.X < reopening.X && reopening.X < reopened.X && closed.W > reopening.W && reopening.W > reopened.W) {
				t.Fatalf("点击固定开关后编辑区应逐渐缩小：%+v -> %+v -> %+v", closed, reopening, reopened)
			}
		})
	}
}

func TestDocumentTitleMovesSmoothlyWithSidebar(t *testing.T) {
	for _, tc := range []struct {
		window       int
		sidebarWidth float32
	}{{1000, 180}, {760, 320}, {520, 180}} {
		t.Run(fmt.Sprintf("%d/%g", tc.window, tc.sidebarWidth), func(t *testing.T) {
			app := workspaceWithTabs(1)
			app.sidebarWidth = tc.sidebarWidth
			probe := &layoutProbeEditor{Editor: app.editors[0]}
			app.active, app.editors = probe, nil // 避免与侧栏中的同名标题混淆。
			tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "darwin") }, tc.window, 650)
			tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
			measure := func() (ui.Rect, ui.Rect, ui.Rect) {
				t.Helper()
				title, titleOK := tt.Find(probe.Title())
				editor, editorOK := tt.Find("editor-region")
				button, buttonOK := tt.Find(locales.SidebarToggle())
				if !titleOK || !editorOK || !buttonOK {
					t.Fatal("未找到标题、编辑区或侧栏开关")
				}
				if title.X < button.X+button.W {
					t.Fatalf("标题与侧栏开关重叠：%+v / %+v", title, button)
				}
				return title, editor, button
			}
			openedTitle, openedEditor, _ := measure()
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Advance(3 * time.Second)
			closedTitle, closedEditor, _ := measure()
			check := func() {
				t.Helper()
				title, editor, _ := measure()
				progress := (editor.X - closedEditor.X) / (openedEditor.X - closedEditor.X)
				want := closedTitle.X + (openedTitle.X-closedTitle.X)*progress
				if math.Abs(float64(title.X-want)) > 0.1 {
					t.Fatalf("标题应随编辑区平滑移动：实际 %g，期望 %g，编辑区 %g", title.X, want, editor.X)
				}
			}
			// 先展开一半再反向收起，覆盖动画进行中改变方向。
			tt.Key(ui.Cmd, ui.KeyB)
			for range 5 {
				tt.Advance(16 * time.Millisecond)
				check()
			}
			tt.Key(ui.Cmd, ui.KeyB)
			for range 25 {
				tt.Advance(16 * time.Millisecond)
				check()
			}
			tt.Key(ui.Cmd, ui.KeyB)
			for range 25 {
				tt.Advance(16 * time.Millisecond)
				check()
			}
		})
	}
}

func TestSettingsDragStripBlocksWorkspaceActions(t *testing.T) {
	app := workspaceWithTabs(1)
	app.active = app.editors[0]
	app.active.(*format.FormatEditor).Focus()
	tt := mountWorkspace(app)
	click(t, tt, locales.Settings())
	for _, x := range []float32{2, 100, 700, 1398} {
		tt.ClickAt(x, 24)
		tt.Press(x, 24)
		tt.Move(x+5, 50)
		tt.Release(x+5, 50)
	}
	if !app.settingState.Open || !app.sidebarOpen || len(app.editors) != 1 {
		t.Fatal("弹窗标题栏拖动泄漏操作或关闭了弹窗")
	}
	tt.Key(0, ui.KeyEscape)
	if app.settingState.Open || !app.active.(*format.FormatEditor).Focused() {
		t.Fatal("关闭弹窗未恢复编辑器焦点")
	}
}

func TestWorkspaceUsesNativeSidebarSurface(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	tt.SetVibrancy(true)
	// 侧栏坐在整窗的底色上，开了材质时底色略透；卡片始终不透明。
	if tt.Image().RGBAAt(100, 500).A == 255 || tt.Image().RGBAAt(500, 500).A != 255 {
		t.Fatal("材质开启时侧栏应透出底色，卡片应保持不透明")
	}
	tt.SetVibrancy(false)
	if tt.Image().RGBAAt(100, 500).A != 255 || tt.Image().RGBAAt(500, 500).A != 255 {
		t.Fatal("材质不可用时底色和卡片都应不透明")
	}
}

func TestEditorToastFollowsContentAndWindowFallbackClearsHeader(t *testing.T) {
	for _, mode := range []string{"empty", "format", "diff", "preview"} {
		app := workspaceWithTabs(1)
		editorOpen := mode != "empty"
		if editorOpen {
			app.active = app.editors[0]
		} else {
			app.editors, app.active = nil, nil
		}
		if mode == "diff" || mode == "preview" {
			e := diff.NewDiffEditor("editor-2", `{"a":1}`, `{"a":2}`)
			if mode == "preview" {
				e.ToggleComparison()
			}
			app.editors[0], app.active = e, e
		}
		show := true
		tt := ui.NewTester(func(c *ui.Context) {
			if show {
				c.AddToast(ui.Toast{Title: "Toast anchor regression", Timeout: -1})
				show = false
			}
			app.frame(c, "windows")
		}, 1000, 650)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		tt.Advance(time.Second)
		before, ok := tt.Find("Toast anchor regression")
		if !ok {
			t.Fatalf("%s 的提示容器丢失", mode)
		}
		if !editorOpen {
			if before.Y < 48 || before.Y > 200 {
				t.Fatalf("窗口级后备提示未在标题栏下方：%+v", before)
			}
			continue
		}
		if before.Y < 500 {
			t.Fatalf("%s 的提示不在编辑器底部：%+v", mode, before)
		}
		app.sidebarWidth = 320
		tt.Frame()
		tt.Advance(time.Second)
		after, _ := tt.Find("Toast anchor regression")
		if difference := after.X - before.X; difference < 69 || difference > 71 || after.Y != before.Y {
			t.Fatalf("%s 的提示没有跟随编辑区的新布局：%+v -> %+v", mode, before, after)
		}
	}
}

type layoutProbeEditor struct {
	editor.Editor
	nodeID uint64
	local  *int
}

func (e *layoutProbeEditor) View(c *ui.Context) {
	// 卡片圆角会把贴边元素的边界往里收半个像素，测量用卡片内部的一块。
	view := ui.Column(c.Key("layout-probe")).Fill().Padding(1).Label("editor-region")
	e.nodeID = view.ID()
	e.local = ui.Local(view, "identity", func() int { return 1 })
	view.Children(func() { e.Editor.View(c) })
}

func TestDocumentCardRetainsTopGap(t *testing.T) {
	app := workspaceWithTabs(1)
	probe := &layoutProbeEditor{Editor: app.editors[0]}
	app.editors[0], app.active = probe, probe
	tt := ui.NewTester(app.view, 1000, 650)
	bounds, ok := tt.Find("editor-region")
	// 编辑区位于卡片顶部间距和 42 DIP 标题行之后；允许半像素边框造成的偏差。
	if !ok || math.Abs(float64(bounds.Y-(cardGap+42))) > 1 {
		t.Fatalf("文档卡片应保留顶部间距：编辑区 %+v", bounds)
	}
}

func TestDocumentCardEdgeIsContinuousAcrossEditor(t *testing.T) {
	for _, mode := range []string{"format", "diff", "preview"} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dark=%v", mode, dark), func(t *testing.T) {
				app := workspaceWithTabs(1)
				app.active = app.editors[0]
				if mode != "format" {
					e := diff.NewDiffEditor("diff", `{"a":1}`, `{"a":2}`)
					if mode == "preview" {
						e.ToggleComparison()
					}
					app.active = e
				}
				tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "darwin") }, 1000, 650)
				tt.SetPreferences(ui.Preferences{ReduceMotion: true})
				tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
				tt.SetScale(2)
				tt.SetDark(dark)
				im := tt.Image()
				// 比较两侧跨越交界处的每行物理像素，避开卡片圆角。
				for _, x := range []int{360, 361, 1982, 1983} {
					want := im.RGBAAt(x, 70)
					for y := 71; y <= 150; y++ {
						if got := im.RGBAAt(x, y); got != want {
							t.Errorf("边缘在 (%d,%d) 不连续：%v，期望 %v", x, y, got, want)
							break
						}
					}
				}
			})
		}
	}
}

func TestDiffPreviewFillsCardHeight(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, layout := range []mygoeditor.DiffLayout{mygoeditor.DiffSplit, mygoeditor.DiffUnified} {
			app := workspaceWithTabs(1)
			prefs := editor.DefaultPreferences()
			prefs.DiffLayout = layout
			e := diff.NewDiffEditor("diff", "{}", "[]", prefs)
			e.ToggleComparison()
			app.active = e
			tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "darwin") }, 1000, 650)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			tt.SetDark(dark)
			preview, ok := tt.Find(locales.DiffPreviewLabel())
			if !ok || math.Abs(float64(preview.Y+preview.H-(650-cardGap))) > 1.1 {
				t.Fatalf("预览应延伸至卡片底边，dark=%v layout=%v：%+v", dark, layout, preview)
			}
		}
	}
}

func TestToolbarEditorVerticalSpacing(t *testing.T) {
	for _, platform := range []string{"darwin", "windows", "linux"} {
		for _, scale := range []float32{1, 1.5, 2} {
			t.Run(fmt.Sprintf("%s/scale=%g", platform, scale), func(t *testing.T) {
				app := workspaceWithTabs(1)
				app.active = app.editors[0]
				tt := ui.NewTester(func(c *ui.Context) { app.frame(c, platform) }, 1000, 650)
				tt.SetPreferences(ui.Preferences{TextScale: scale, ReduceMotion: true})
				if platform == "darwin" {
					tt.SetTitleBar(ui.TitleBar{Height: 48, Left: 80})
				}
				for _, open := range []bool{true, false} {
					app.sidebarOpen = open
					tt.Frame()
					button, buttonOK := tt.Find(locales.FormatJSON())
					editor, editorOK := tt.Find(locales.EditorLabel())
					if !buttonOK || !editorOK {
						t.Fatalf("未找到工具栏按钮或编辑器：%+v / %+v", button, editor)
					}
					if math.Abs(float64(button.Y-(cardGap+4))) > 1 {
						t.Fatalf("工具栏按钮位置发生改变：%+v", button)
					}
					if gap := editor.Y - (button.Y + button.H); gap < 5 || gap > 7 {
						t.Fatalf("工具栏按钮与编辑器应保留 6 DIP 间距：%g，按钮 %+v，编辑器 %+v", gap, button, editor)
					}
					if scale == 1 && math.Abs(float64(editor.Y-(cardGap+42))) > 1 {
						t.Fatalf("默认编辑器顶部应上移至 42 DIP 标题行之后：%+v", editor)
					}
				}
			})
		}
	}
}

func TestEditorStaysBelowTallNativeTitleBar(t *testing.T) {
	app := workspaceWithTabs(1)
	app.active = app.editors[0]
	tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "darwin") }, 1000, 650)
	tt.SetTitleBar(ui.TitleBar{Height: 64, Left: 80})
	editor, ok := tt.Find(locales.EditorLabel())
	if !ok || editor.Y < 64 {
		t.Fatalf("标题栏变高时编辑器不应占用窗口控件区域：%+v", editor)
	}
}

func TestSidebarYieldsToEditorOnNarrowWindows(t *testing.T) {
	app := workspaceWithTabs(1)
	probe := &layoutProbeEditor{Editor: app.editors[0]}
	app.editors[0], app.active = probe, probe
	app.sidebarWidth = 700
	tt := ui.NewTester(app.view, 1400, 650)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	for _, width := range []int{1000, 760, 1400} {
		tt.SetSize(width, 650)
		bounds, ok := tt.Find("editor-region")
		// 侧栏展开时卡片左边贴着侧栏，编辑区还需避开边框和内阴影。
		shown := min(float32(700), max(float32(0), float32(width)-minEditorWidth-cardGap))
		expected := shown + 1
		if !ok || bounds.X != expected || bounds.Y <= cardGap || bounds.W < minEditorWidth-cardGap-2 || app.sidebarWidth != 700 {
			t.Fatalf("窗口 %d：编辑区 %+v，期望起点 %g 且记忆宽度 700，实际 %g", width, bounds, expected, app.sidebarWidth)
		}
	}
	if _, ok := tt.Find("分隔条"); ok {
		t.Fatal("侧栏与卡片之间不应再有分隔条")
	}
}

func TestSplitMotionKeepsWidgetIdentityAndEditorUsable(t *testing.T) {
	app := workspaceWithTabs(1)
	underlying := app.editors[0].(*format.FormatEditor)
	probe := &layoutProbeEditor{Editor: underlying}
	app.editors[0], app.active = probe, probe
	tt := ui.NewTester(app.view, 1000, 650)
	nodeID, local := probe.nodeID, probe.local
	checkIdentity := func() {
		t.Helper()
		if probe.nodeID != nodeID || probe.local != local {
			t.Fatal("切换 Split/过渡布局重建了编辑器节点或本地状态")
		}
	}
	click(t, tt, locales.SidebarToggle())
	tt.Advance(100 * time.Millisecond)
	checkIdentity()
	mid, _ := tt.Find("editor-region")
	if mid.X <= 0 || mid.X >= 181 {
		t.Fatalf("收起中间位置错误：%+v", mid)
	}
	underlying.Focus()
	tt.Frame()
	tt.Type("x")
	if !strings.Contains(underlying.Buffer.String(), "x") {
		t.Fatal("动画期间编辑器不可用")
	}
	tt.Key(ui.Cmd, ui.KeyB)
	reversed, _ := tt.Find("editor-region")
	if delta := reversed.X - mid.X; delta < -.001 || delta > .001 {
		t.Fatalf("动画反向时发生位置跳变：%+v -> %+v", mid, reversed)
	}
	tt.Advance(3 * time.Second)
	checkIdentity()
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Advance(3 * time.Second)
	checkIdentity()
	closed, _ := tt.Find("editor-region")
	if closed.X != cardGap+1 || closed.Y <= cardGap || closed.W != 1000-2*cardGap-2 {
		t.Fatalf("收起后编辑区未占满卡片：%+v", closed)
	}
	for range 10 {
		tt.Key(0, ui.KeyTab)
		if tt.Focused(locales.TabList()) || tt.Focused(locales.NewFormat()) {
			t.Fatal("完全收起后 Tab 仍进入侧栏")
		}
	}
	tt.Key(ui.Cmd, ui.KeyB)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	checkIdentity()
	if app.sidebarWidth != defaultSidebarWidth {
		t.Fatal("减少动态效果改变了侧栏宽度")
	}
}

func TestSidebarSelectionKeepsFocusForConsecutiveNavigation(t *testing.T) {
	app := workspaceWithTabs(3)
	tt := mountWorkspace(app)
	click(t, tt, app.editors[0].Title())
	for _, want := range []string{"editor-2", "editor-3"} {
		tt.Key(0, ui.KeyDown)
		if activeID(app) != want || !tt.Focused(locales.TabList()) {
			t.Fatalf("连续选择 %s 失败或焦点离开列表", want)
		}
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	if activeID(app) != "editor-2" || !app.active.(*format.FormatEditor).Focused() {
		t.Fatal("编辑器切换快捷键未聚焦文档")
	}
}

func TestSidebarResizesFromEditorEdge(t *testing.T) {
	app := workspaceWithTabs(1)
	e := app.editors[0].(*format.FormatEditor)
	app.active = e
	e.Buffer.Set(`{"a":1}`)
	e.Select(2, 4)
	e.Find("a")
	selection := e.Selections()[0]
	tt := ui.NewTester(app.view, 1000, 650)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})

	// 拖动从编辑器卡片的左边缘开始，且鼠标应提示可水平调整。
	tt.Move(defaultSidebarWidth+2, 300)
	if tt.Cursor() != ui.CursorResizeEW {
		t.Fatalf("编辑器左边缘应显示调整宽度指针：%v", tt.Cursor())
	}
	tt.Press(defaultSidebarWidth+2, 300)
	tt.Move(defaultSidebarWidth+102, 300)
	tt.Release(defaultSidebarWidth+102, 300)
	if math.Abs(float64(app.sidebarWidth-(defaultSidebarWidth+100))) > 1 {
		t.Fatalf("拖动后侧栏宽度错误：%g", app.sidebarWidth)
	}
	bounds, ok := tt.Find(locales.EditorLabel())
	if !ok || bounds.X < app.sidebarWidth || bounds.X > app.sidebarWidth+20 {
		t.Fatalf("编辑器未跟随侧栏边缘：%+v，侧栏宽度 %g", bounds, app.sidebarWidth)
	}
	if e.Buffer.String() != `{"a":1}` || e.Selections()[0] != selection {
		t.Fatal("拖动侧栏重置了编辑器内容或选区")
	}
	if count, _ := e.FindMatches(); count != 1 || !e.Buffer.Undo() || e.Buffer.String() != "{}" {
		t.Fatal("拖动侧栏重置了搜索或撤销历史")
	}

	// 拖到窗口两侧时必须保留侧栏和编辑器的最小宽度。
	tt.Press(app.sidebarWidth+2, 300)
	tt.Move(1, 300)
	tt.Release(1, 300)
	if app.sidebarWidth != minSidebarWidth {
		t.Fatalf("侧栏最小宽度错误：%g", app.sidebarWidth)
	}
	tt.Press(app.sidebarWidth+2, 300)
	tt.Move(999, 300)
	tt.Release(999, 300)
	if want := float32(1000) - minEditorWidth - cardGap; app.sidebarWidth != want {
		t.Fatalf("侧栏最大宽度错误：%g，期望 %g", app.sidebarWidth, want)
	}
}

func TestSidebarEdgeResizeDisabledWhenClosedOrSettingsOpen(t *testing.T) {
	for _, state := range []string{"closed", "settings"} {
		t.Run(state, func(t *testing.T) {
			app := workspaceWithTabs(1)
			app.active = app.editors[0]
			app.sidebarWidth = 250
			if state == "closed" {
				app.sidebarOpen = false
			} else {
				app.settingState.Open = true
			}
			tt := ui.NewTester(app.view, 1000, 650)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			x := app.sidebarWidth + 2
			if state == "closed" {
				x = cardGap + 2
			}
			tt.Press(x, 300)
			tt.Move(x+100, 300)
			tt.Release(x+100, 300)
			if app.sidebarWidth != 250 {
				t.Fatalf("%s 时拖动改变了侧栏宽度：%g", state, app.sidebarWidth)
			}
		})
	}
}
