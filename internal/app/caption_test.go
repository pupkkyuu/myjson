package app

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/locales"
	"myjson/internal/palette"
	"myjson/pkg/icons/lucide"
)

type testWindowCommands struct {
	minimized, closed int
	maximized         bool
}

func (w *testWindowCommands) Minimize()         { w.minimized++ }
func (w *testWindowCommands) ToggleMaximize()   { w.maximized = !w.maximized }
func (w *testWindowCommands) IsMaximized() bool { return w.maximized }
func (w *testWindowCommands) Close()            { w.closed++ }

func TestCaptionButtonsDispatchAndRefresh(t *testing.T) {
	w := &testWindowCommands{}
	tt := ui.NewTester(func(c *ui.Context) { captionButtons(c, w) }, 400, 60)
	click(t, tt, locales.MinimizeWindow())
	click(t, tt, locales.MaximizeWindow())
	if !w.maximized || !tt.HasText(locales.RestoreWindow()) {
		t.Fatal("最大化状态及按钮未更新")
	}
	w.maximized = false
	tt.Frame()
	if !tt.HasText(locales.MaximizeWindow()) {
		t.Fatal("外部还原后按钮未跟随窗口状态")
	}
	click(t, tt, locales.MaximizeWindow())
	click(t, tt, locales.RestoreWindow())
	click(t, tt, locales.CloseWindow())
	if w.minimized != 1 || w.closed != 1 || w.maximized {
		t.Fatalf("窗口操作派发错误：%+v", w)
	}
}

func TestCaptionAndToolbarGlyphsMatch(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		app := newApplication()
		tt := ui.NewTester(func(c *ui.Context) {
			app.theme = resolveTheme(c, palette.Default)
			c.SetTheme(app.theme)
			c.Root().Background(ui.Transparent)
			ui.Column(c).Children(func() {
				captionButtons(c, nil)
				component.IconButton(c.Key("reference"), icons.X(), "Reference close")
			})
		}, 500, 200)
		tt.SetPreferences(ui.Preferences{TextScale: scale})
		caption, _ := tt.Find(locales.CloseWindow())
		reference, _ := tt.Find("Reference close")
		if caption.W != reference.W || caption.H != reference.H || caption.W < 32 {
			t.Fatal("窗口按钮与工具栏的命中区域不一致")
		}
		inkWidth := func(r ui.Rect) int {
			left, right := int(r.X+r.W), int(r.X)-1
			img := tt.Image()
			for y := int(r.Y); y < int(r.Y+r.H); y++ {
				for x := int(r.X); x < int(r.X+r.W); x++ {
					if img.RGBAAt(x, y).A > 0 {
						left, right = min(left, x), max(right, x)
					}
				}
			}
			return right - left + 1
		}
		if a, b := inkWidth(caption), inkWidth(reference); a <= 0 || a != b {
			t.Fatalf("缩放 %g 下图标尺寸不一致：%d / %d", scale, a, b)
		}
		tt.Move(caption.X+caption.W/2, caption.Y+caption.H/2)
		color := tt.Image().RGBAAt(int(caption.X+caption.W/2), int(caption.Y+3))
		if color.R != 232 || color.G != 17 || color.B != 35 {
			t.Fatalf("关闭按钮丢失红色悬浮反馈：%+v", color)
		}
	}
}

func TestWorkspaceIconButtonsHaveRoundHover(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, label := range []string{locales.FormatJSON(), locales.SidebarToggle(), locales.CloseWindow(), locales.Settings()} {
			t.Run(fmt.Sprintf("dark=%v/%s", dark, label), func(t *testing.T) {
				app := workspaceWithTabs(1)
				app.active = app.editors[0]
				tt := ui.NewTester(func(c *ui.Context) { app.frame(c, "windows") }, 1000, 650)
				tt.SetDark(dark)
				button, ok := tt.Find(label)
				if !ok {
					t.Fatalf("未找到按钮 %q", label)
				}
				// 左上角应位于圆外，上方中点应位于圆内且避开图标。
				cornerX, cornerY := int(button.X+3), int(button.Y+3)
				insideX, insideY := int(button.X+button.W/2), int(button.Y+3)
				beforeCorner := tt.Image().RGBAAt(cornerX, cornerY)
				beforeInside := tt.Image().RGBAAt(insideX, insideY)
				tt.Move(button.X+button.W/2, button.Y+button.H/2)
				afterCorner := tt.Image().RGBAAt(cornerX, cornerY)
				afterInside := tt.Image().RGBAAt(insideX, insideY)
				if beforeCorner != afterCorner || beforeInside == afterInside {
					t.Fatalf("按钮悬浮背景应为圆形：角落 %v → %v，圆内 %v → %v", beforeCorner, afterCorner, beforeInside, afterInside)
				}
				tt.Press(button.X+button.W/2, button.Y+button.H/2)
				pressedCorner := tt.Image().RGBAAt(cornerX, cornerY)
				pressedInside := tt.Image().RGBAAt(insideX, insideY)
				if beforeCorner != pressedCorner || beforeInside == pressedInside {
					t.Fatalf("按钮按下背景应为圆形：角落 %v → %v，圆内 %v → %v", beforeCorner, pressedCorner, beforeInside, pressedInside)
				}
			})
		}
	}
}
