package toolbar

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestActionAvailabilityByEditorMode(t *testing.T) {
	base := []Action{Format, Compact, Escape, Unescape, SortFields, FoldAll, UnfoldAll, ToggleWrap, Search, CopyAll, Clear}
	for _, tc := range []struct {
		name             string
		options          Options
		prefix, disabled []Action
	}{
		{"格式化", Options{Kind: FormatEditor, CopyEnabled: true}, nil, nil},
		{"不可复制", Options{Kind: FormatEditor}, nil, []Action{CopyAll}},
		{"双侧输入", Options{Kind: DiffEditor, CopyEnabled: true}, []Action{ToggleComparison}, []Action{CopyAll}},
		{"无差异预览", Options{Kind: DiffEditor, Comparing: true}, []Action{PreviousDifference, NextDifference, ToggleComparison}, []Action{PreviousDifference, NextDifference, Format, Compact, Escape, Unescape, SortFields, CopyAll}},
		{"有差异预览", Options{Kind: DiffEditor, Comparing: true, DifferenceCount: 2}, []Action{PreviousDifference, NextDifference, ToggleComparison}, []Action{Format, Compact, Escape, Unescape, SortFields, CopyAll}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var actions []Action
			for _, group := range toolGroups(tc.options) {
				for _, tool := range group {
					actions = append(actions, tool.action)
					if tool.label == "" || tool.icon == nil || tool.disabled != slices.Contains(tc.disabled, tool.action) {
						t.Errorf("工具状态不正确：%+v", tool)
					}
					if tool.action == ToggleComparison {
						want := locales.CompareJSON()
						if tc.options.Comparing {
							want = locales.EditJSON()
						}
						if tool.label != want {
							t.Errorf("模式切换标签 = %q，期望 %q", tool.label, want)
						}
					}
				}
			}
			want := append(slices.Clone(tc.prefix), base...)
			if !slices.Equal(actions, want) {
				t.Fatalf("工具顺序 = %v，期望 %v", actions, want)
			}
		})
	}
}

func TestToolbarClickDispatchAndDisabledActions(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		options := Options{Kind: FormatEditor, Title: "文档", CopyEnabled: true}
		if disabled {
			options.DisabledActions = []Action{Format, Clear}
		}
		var clicked []Action
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Row(c).FillWidth().Height(48).Children(func() {
				if action := View(c, options); action != NoAction {
					clicked = append(clicked, action)
				}
			})
		}, 1600, 100)
		for _, tc := range []struct {
			label  string
			action Action
		}{
			{locales.FormatJSON(), Format}, {locales.Clear(), Clear}, {locales.Copy(), CopyAll},
		} {
			clicked = nil
			if err := tt.Click(tc.label); err != nil {
				t.Fatal(err)
			}
			if disabled && tc.action != CopyAll {
				if len(clicked) != 0 {
					t.Fatalf("禁用的工具仍产生操作：%v", clicked)
				}
			} else if !slices.Equal(clicked, []Action{tc.action}) {
				t.Fatalf("点击 %q 返回 %v，期望 %v", tc.label, clicked, tc.action)
			}
		}
	}
}

func TestNoEditorOnlyShowsTitle(t *testing.T) {
	for _, title := range []string{"", "自定义标题"} {
		tt := ui.NewTester(func(c *ui.Context) {
			if action := View(c, Options{Title: title}); action != NoAction {
				t.Fatal("无编辑器时不应返回操作")
			}
		}, 800, 80)
		if title != "" && !tt.HasText(title) {
			t.Fatalf("未显示标题 %q", title)
		}
		if title == "" && tt.HasText("MyJson") {
			t.Fatal("无标题时不应显示默认标题")
		}
		if _, ok := tt.Find(locales.FormatJSON()); ok {
			t.Fatal("无编辑器时不应显示文档工具")
		}
	}
}

func TestNarrowToolbarCanReachTrailingActions(t *testing.T) {
	var clicked Action
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Height(48).Children(func() {
			if action := View(c, Options{Kind: FormatEditor, Title: "文档", CopyEnabled: true}); action != NoAction {
				clicked = action
			}
		})
	}, 320, 100)
	r, ok := tt.Find(locales.EditorToolbar())
	if !ok {
		t.Fatal("未找到可滚动工具栏")
	}
	tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 2000, 0)
	if err := tt.Click(locales.Clear()); err != nil {
		t.Fatal(err)
	}
	if clicked != Clear {
		t.Fatalf("横向滚动后无法点击末尾工具：%v", clicked)
	}
}

func TestLoadingToolbarBlocksTransformsAndRestoresActions(t *testing.T) {
	for _, kind := range []EditorKind{FormatEditor, DiffEditor} {
		options := Options{Kind: kind, Title: "文档", CopyEnabled: true, Loading: true}
		var clicked Action
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Row(c).FillWidth().Height(48).Children(func() {
				if action := View(c, options); action != NoAction {
					clicked = action
				}
			})
		}, 1600, 100)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
		for _, label := range []string{locales.FormatJSON(), locales.Compact(), locales.Escape(), locales.Unescape(), locales.SortFields()} {
			clicked = NoAction
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if clicked != NoAction {
				t.Fatalf("忙碌时仍派发 %q", label)
			}
		}
		for _, tc := range []struct {
			label string
			want  Action
		}{{locales.Search(), Search}, {locales.SoftWrap(), ToggleWrap}, {locales.Clear(), Clear}} {
			clicked = NoAction
			if err := tt.Click(tc.label); err != nil {
				t.Fatal(err)
			}
			if clicked != tc.want {
				t.Fatalf("忙碌时 %q 应保持可用，收到 %q", tc.label, clicked)
			}
		}
		options.Loading = false
		tt.Frame()
		clicked = NoAction
		if err := tt.Click(locales.FormatJSON()); err != nil {
			t.Fatal(err)
		}
		if clicked != Format {
			t.Fatal("恢复空闲后变换按钮仍被禁用")
		}
	}
}

func TestLoadingPreservesButtonAppearanceAndPressFeedback(t *testing.T) {
	options := Options{Kind: FormatEditor, Title: "文档", CopyEnabled: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Height(48).Children(func() { View(c, options) })
	}, 1600, 100)
	button, ok := tt.Find(locales.FormatJSON())
	if !ok {
		t.Fatal("未找到格式化按钮")
	}
	x, y := button.X+button.W/2, button.Y+button.H/2
	tt.Move(x, y)
	before := tt.Image()
	options.Loading = true
	tt.Frame()
	after := tt.Image()
	for py := int(button.Y); py < int(button.Y+button.H); py++ {
		for px := int(button.X); px < int(button.X+button.W); px++ {
			if before.RGBAAt(px, py) != after.RGBAAt(px, py) {
				t.Fatal("处理中不应改变按钮外观")
			}
		}
	}
	insideX, insideY := int(x), int(button.Y+3)
	hover := after.RGBAAt(insideX, insideY)
	tt.Press(x, y)
	if tt.Image().RGBAAt(insideX, insideY) == hover {
		t.Fatal("处理中应保留按钮按下反馈")
	}
	tt.Release(x, y)
}

func TestLoadingIndicatorKeepsToolbarInPlace(t *testing.T) {
	options := Options{Kind: FormatEditor, Title: "文档", CopyEnabled: true, Loading: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Height(48).Children(func() { View(c, options) })
	}, 1600, 100)
	before, _ := tt.Find(locales.FormatJSON())
	if _, ok := tt.Find(locales.Transforming()); ok {
		t.Fatal("延迟前不应显示处理提示")
	}
	options.ShowLoading = true
	tt.Frame()
	if _, ok := tt.Find(locales.Transforming()); !ok {
		t.Fatal("延迟后应显示处理提示")
	}
	after, _ := tt.Find(locales.FormatJSON())
	if before != after {
		t.Fatalf("处理提示不应移动工具按钮：%+v → %+v", before, after)
	}
	options.Loading = false
	tt.Frame()
	if _, ok := tt.Find(locales.Transforming()); ok {
		t.Fatal("处理完成后应隐藏提示")
	}
}

func TestNarrowToolbarWheelScrolling(t *testing.T) {
	for _, tc := range []struct {
		name             string
		reduced, loading bool
	}{
		{"smooth", false, false},
		{"reduced-motion", true, false},
		{"smooth-loading", false, true},
		{"reduced-motion-loading", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var clicked Action
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).FillWidth().Height(48).Children(func() {
					if action := View(c, Options{Kind: FormatEditor, Title: "文档", CopyEnabled: true, Loading: tc.loading}); action != NoAction {
						clicked = action
					}
				})
			}, 320, 100)
			tt.SetPreferences(ui.Preferences{ReduceMotion: tc.reduced, TextScale: 1})
			scroll, ok := tt.Find(locales.EditorToolbar())
			if !ok {
				t.Fatal("未找到工具栏滚动区")
			}
			before, _ := tt.Find(locales.Clear())
			x, y := scroll.X+scroll.W/2, scroll.Y+scroll.H/2
			if !tc.reduced {
				tt.Wheel(x, y, 80, 0)
				start, _ := tt.Find(locales.Clear())
				if start.X != before.X {
					t.Fatal("平滑滚动不应立即跳到终点")
				}
				tt.Advance(30 * time.Millisecond)
				middle, _ := tt.Find(locales.Clear())
				if middle.X >= before.X || middle.X <= before.X-80 {
					t.Fatalf("未观察到滚轮动画的中间位置：%g -> %g", before.X, middle.X)
				}
				tt.Advance(200 * time.Millisecond)
			}
			tt.Wheel(x, y, 2000, 0)
			if !tc.reduced {
				tt.Advance(200 * time.Millisecond)
			}
			end, ok := tt.Find(locales.Clear())
			if !ok || end.X < scroll.X || end.X+end.W > scroll.X+scroll.W {
				t.Fatalf("滚轮滚动后末尾按钮不可见：%+v，滚动区 %+v", end, scroll)
			}
			tt.Advance(time.Second)
			settled, _ := tt.Find(locales.Clear())
			if settled.X != end.X {
				t.Fatal("滚动结束后位置仍在变化")
			}
			if err := tt.Click(locales.Clear()); err != nil {
				t.Fatal(err)
			}
			if clicked != Clear {
				t.Fatal("滚轮滚动后无法操作末尾按钮")
			}
		})
	}
}
