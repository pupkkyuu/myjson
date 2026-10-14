package sidebar

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestSidebarKeyboardActivation(t *testing.T) {
	options := Options{Tabs: []Tab{{ID: "editor-7", Title: "Apple"}, {ID: "editor-8", Title: "Banana"}}, TopInset: 48}
	var got Event
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			if event := View(c, options); event.Action != NoAction {
				got = event
				if event.Action == ActivateTab {
					options.ActiveID = event.TabID
				}
			}
			ui.Button(c, "After sidebar")
		})
	}, 250, 600)
	// 侧栏自己不再放开关，那一行只用来拖动窗口，第一个停靠点是新建按钮。
	for _, tc := range []struct {
		label string
		key   ui.Key
		want  Event
	}{
		{locales.NewFormat(), ui.KeyEnter, Event{Action: NewFormat}},
		{locales.NewComparison(), ui.KeySpace, Event{Action: NewComparison}},
	} {
		tt.Key(0, ui.KeyTab)
		if !tt.Focused(tc.label) {
			t.Fatalf("Tab 未聚焦 %q", tc.label)
		}
		got = Event{}
		tt.Key(0, tc.key)
		if got != tc.want {
			t.Fatalf("键盘激活返回 %+v，期望 %+v", got, tc.want)
		}
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(locales.TabList()) {
		t.Fatal("标签列表应只有一个 Tab 停靠点")
	}
	for _, tc := range []struct {
		key ui.Key
		id  string
	}{
		{ui.KeyDown, "editor-7"}, {ui.KeyDown, "editor-8"}, {ui.KeyHome, "editor-7"},
		{ui.KeyEnd, "editor-8"}, {ui.KeyUp, "editor-7"}, {ui.KeyB, "editor-8"},
	} {
		got = Event{}
		tt.Key(0, tc.key)
		if got != (Event{Action: ActivateTab, TabID: tc.id}) || !tt.Focused(locales.TabList()) {
			t.Fatalf("原生导航返回 %+v，期望 %s 并保留列表焦点", got, tc.id)
		}
	}
	row, _ := tt.Find("Banana")
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	if _, ok := tt.Find(locales.CloseTab("Banana")); !ok {
		t.Fatal("应显示悬浮关闭按钮")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("After sidebar") {
		t.Fatal("Tab 应直接离开标签列表")
	}
	options.Disabled = true
	tt.Frame()
	got = Event{}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if got.Action != NoAction {
		t.Fatal("禁用侧栏仍响应键盘激活")
	}
}

func TestSidebarLayoutAcrossThemesAndScales(t *testing.T) {
	for _, width := range []int{180, 320} {
		for _, dark := range []bool{false, true} {
			for _, scale := range []float32{1, 1.25, 1.5} {
				t.Run(fmt.Sprintf("%d/dark=%v/scale=%g", width, dark, scale), func(t *testing.T) {
					options := Options{Tabs: []Tab{{ID: "editor-7", Title: "文档一"}, {ID: "editor-8", Title: "文档二"}}, ActiveID: "editor-7", TopInset: 48}
					tt := ui.NewTester(func(c *ui.Context) {
						theme := *c.Theme()
						theme.FontSize *= scale
						c.SetTheme(&theme)
						View(c, options)
					}, width, 600)
					tt.SetDark(dark)
					comparison, ok := tt.Find(locales.NewComparison())
					if !ok {
						t.Fatal("未找到新建对比入口")
					}
					heading, ok := tt.Find(locales.CountTabs(2))
					if !ok {
						t.Fatal("未找到标签页小标题")
					}
					first, ok := tt.Find("文档一")
					if !ok {
						t.Fatal("未找到第一个标签")
					}
					above := heading.Y - comparison.Y - comparison.H
					below := first.Y - heading.Y - heading.H
					if above < 0 || below < 0 {
						t.Fatalf("原生侧栏条目重叠：标题上下间距 %g/%g", above, below)
					}
					if first.X < 0 || first.X+first.W > float32(width) {
						t.Fatalf("标签越过侧栏边界：%+v", first)
					}
					if directory := os.Getenv("MYJSON_TEST_SCREENSHOTS"); directory != "" {
						for _, disabled := range []bool{false, true} {
							options.Disabled = disabled
							tt.Frame()
							if err := os.MkdirAll(directory, 0755); err != nil {
								t.Fatal(err)
							}
							name := fmt.Sprintf("sidebar-%d-dark-%v-scale-%g-disabled-%v.png", width, dark, scale, disabled)
							f, err := os.Create(filepath.Join(directory, name))
							if err != nil {
								t.Fatal(err)
							}
							err = png.Encode(f, tt.Image())
							closeErr := f.Close()
							if err != nil || closeErr != nil {
								t.Fatalf("保存侧栏截图失败：%v / %v", err, closeErr)
							}
						}
					}
				})
			}
		}
	}
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestSidebarReportsUserActions(t *testing.T) {
	options := Options{Tabs: []Tab{{ID: "editor-7", Title: "文档"}}, TopInset: 48}
	var got Event
	tt := ui.NewTester(func(c *ui.Context) {
		if event := View(c, options); event.Action != NoAction {
			got = event
		}
	}, 250, 600)
	for _, tc := range []struct {
		label string
		want  Event
	}{
		{locales.NewFormat(), Event{Action: NewFormat}},
		{locales.NewComparison(), Event{Action: NewComparison}},
		{"文档", Event{Action: ActivateTab, TabID: "editor-7"}},
		{locales.CloseTab("文档"), Event{Action: CloseTab, TabID: "editor-7"}},
	} {
		if tc.want.Action == CloseTab {
			row, _ := tt.Find("文档")
			tt.Move(row.X+row.W/2, row.Y+row.H/2)
		}
		got = Event{}
		click(t, tt, tc.label)
		if got != tc.want {
			t.Fatalf("点击 %q 返回 %+v，期望 %+v", tc.label, got, tc.want)
		}
	}
}

func TestDisabledSidebarBlocksActionsAndHover(t *testing.T) {
	options := Options{Tabs: []Tab{{ID: "editor-1", Title: "文档"}}, Disabled: true, TopInset: 48}
	var events []Event
	tt := ui.NewTester(func(c *ui.Context) {
		if event := View(c, options); event.Action != NoAction {
			events = append(events, event)
		}
	}, 250, 600)
	for _, label := range []string{locales.NewFormat(), locales.NewComparison(), "文档"} {
		click(t, tt, label)
	}
	r, _ := tt.Find("文档")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if len(events) != 0 {
		t.Fatal("禁用侧栏仍派发业务操作")
	}
	if _, ok := tt.Find(locales.CloseTab("文档")); ok {
		t.Fatal("禁用侧栏不应显示悬浮关闭按钮")
	}
}

func TestStationaryPointerTracksReorderedTabs(t *testing.T) {
	options := Options{Tabs: []Tab{{ID: "editor-1", Title: "一"}, {ID: "editor-2", Title: "二"}, {ID: "editor-3", Title: "三"}}, TopInset: 48}
	tt := ui.NewTester(func(c *ui.Context) {
		if event := View(c, options); event.Action == CloseTab {
			options.Tabs = slices.DeleteFunc(options.Tabs, func(tab Tab) bool { return tab.ID == event.TabID })
			c.Invalidate()
		}
	}, 250, 600)
	r, _ := tt.Find("一")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	closeRect, ok := tt.Find(locales.CloseTab("一"))
	if !ok {
		t.Fatal("悬浮时未显示关闭按钮")
	}
	x, y := closeRect.X+closeRect.W/2, closeRect.Y+closeRect.H/2
	for _, title := range []string{"一", "二", "三"} {
		button, ok := tt.Find(locales.CloseTab(title))
		if !ok || !button.Contains(x, y) {
			t.Fatalf("标签 %q 重排后未跟踪静止鼠标", title)
		}
		row, _ := tt.Find(title)
		rowX, rowY := int(row.X+4), int(row.Y+row.H/2)
		buttonX, buttonY := int(button.X+3), int(button.Y+button.H/2)
		rowColor := tt.Image().RGBAAt(rowX, rowY)
		buttonColor := tt.Image().RGBAAt(buttonX, buttonY)
		tt.Move(x, y)
		if tt.Image().RGBAAt(rowX, rowY) != rowColor || tt.Image().RGBAAt(buttonX, buttonY) != buttonColor {
			t.Fatalf("%s：悬浮底色仍依赖额外移动，条目 %v -> %v，按钮 %v -> %v", title, rowColor, tt.Image().RGBAAt(rowX, rowY), buttonColor, tt.Image().RGBAAt(buttonX, buttonY))
		}
		tt.ClickAt(x, y)
		if tt.FramePending() {
			t.Fatal("悬浮处理不应持续请求重绘")
		}
	}
	if len(options.Tabs) != 0 {
		t.Fatal("固定位置连续点击未关闭全部标签")
	}
}

func TestFooterReportsSettingsUnlessDisabled(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		clicked := false
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Row(c).Fill().Children(func() {
				if FooterView(c, disabled) {
					clicked = true
				}
			})
		}, 250, 50)
		click(t, tt, locales.Settings())
		if clicked == disabled {
			t.Fatalf("disabled=%v 时设置操作 = %v", disabled, clicked)
		}
	}
}

func TestSelectionScrollsIntoViewWithoutLockingManualScroll(t *testing.T) {
	options := Options{TopInset: 48}
	for i := 1; i <= 40; i++ {
		options.Tabs = append(options.Tabs, Tab{ID: fmt.Sprintf("editor-%d", i), Title: fmt.Sprintf("文档 %d", i)})
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, options) }, 250, 400)
	heading, ok := tt.Find(locales.CountTabs(len(options.Tabs)))
	if !ok {
		t.Fatal("未找到标签页小标题")
	}
	checkHeading := func() {
		t.Helper()
		if current, ok := tt.Find(locales.CountTabs(len(options.Tabs))); !ok || current != heading {
			t.Fatalf("标签页小标题随列表滚动：%+v -> %+v", heading, current)
		}
	}
	options.ActiveID = "editor-40"
	tt.Frame()
	checkHeading()
	last, _ := tt.Find("文档 40")
	if last.H <= 0 || last.Y+last.H > 400 {
		t.Fatalf("选中项未滚入视野：%+v", last)
	}
	tt.Scroll(100, 250, 0, -2000)
	checkHeading()
	first, _ := tt.Find("文档 1")
	if first.H <= 0 {
		t.Fatal("活动项阻止了手动滚动")
	}
	options.ActiveID = "editor-20"
	tt.Frame()
	checkHeading()
	middle, _ := tt.Find("文档 20")
	if middle.H <= 0 || middle.Y+middle.H > 400 {
		t.Fatalf("再次选择未滚入视野：%+v", middle)
	}
}
