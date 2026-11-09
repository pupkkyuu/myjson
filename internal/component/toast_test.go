package component

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestToastsActionsAndDismissal(t *testing.T) {
	var actions, closed int
	pending := true
	tt := ui.NewTester(func(c *ui.Context) {
		if pending {
			pending = false
			c.AddToast(ui.Toast{ID: "error", Title: "格式错误", Description: "原文保持不变", Type: ToastError,
				Timeout: -1, Action: "重试", OnAction: func() { actions++ }, OnClose: func() { closed++ }})
		}
		Toasts(c, ToastOptions{})
	}, 480, 360)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if !tt.HasText("原文保持不变") || slices.Index(tt.Texts(), "格式错误") == -1 {
		t.Fatal("提示标题或说明丢失")
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatal(err)
	}
	if actions != 1 || closed != 1 || tt.HasText("格式错误") {
		t.Fatalf("动作未关闭提示或重复回调：%d / %d", actions, closed)
	}
	pending = true
	tt.Frame()
	if err := tt.Click(locales.CloseToast()); err != nil {
		t.Fatal(err)
	}
	if actions != 1 || closed != 2 || tt.HasText("格式错误") {
		t.Fatalf("关闭按钮未仅关闭提示：%d / %d", actions, closed)
	}
	pending = true
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("重试") {
		t.Fatal("键盘未进入提示动作按钮")
	}
	tt.Key(0, ui.KeyEscape)
	if actions != 1 || closed != 3 || tt.HasText("格式错误") {
		t.Fatalf("Esc 未关闭获得焦点的提示：%d / %d", actions, closed)
	}
}

func TestToastsStackAndPlacement(t *testing.T) {
	for _, bottom := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("bottom=%v/dark=%v", bottom, dark), func(t *testing.T) {
				added := false
				tt := ui.NewTester(func(c *ui.Context) {
					if !added {
						added = true
						for _, title := range []string{"第一条提示", "第二条提示", "最新提示"} {
							c.AddToast(ui.Toast{Title: title, Timeout: -1})
						}
					}
					Toasts(c, ToastOptions{Bottom: bottom, Inset: 48})
				}, 800, 600)
				tt.SetDark(dark)
				tt.SetPreferences(ui.Preferences{ReduceMotion: true})
				find := func() [3]ui.Rect {
					t.Helper()
					var rs [3]ui.Rect
					for i, title := range []string{"第一条提示", "第二条提示", "最新提示"} {
						var ok bool
						rs[i], ok = tt.Find(title)
						if !ok {
							t.Fatalf("缺少提示 %q", title)
						}
					}
					return rs
				}
				distance := func(older, newer ui.Rect) float32 {
					if bottom {
						return newer.Y - older.Y
					}
					return older.Y - newer.Y
				}
				rs := find()
				if !bottom && (rs[2].Y < 48 || rs[2].Y > 120) || bottom && (rs[2].Y < 400 || rs[2].Y > 552) {
					t.Fatalf("提示未避让对应窗口边缘：%+v", rs[2])
				}
				if d := distance(rs[0], rs[1]); d <= 0 || d > 20 || !(rs[0].W < rs[1].W && rs[1].W < rs[2].W) {
					t.Fatalf("提示未缩小堆叠：%+v", rs)
				}
				tt.Move(rs[2].X, rs[2].Y)
				tt.Frame()
				rs = find()
				if distance(rs[0], rs[1]) < 40 || distance(rs[1], rs[2]) < 40 || rs[0].W != rs[2].W {
					t.Fatalf("悬停未展开提示：%+v", rs)
				}
				tt.Move(rs[1].X, (rs[0].Y+rs[1].Y)/2)
				tt.Frame()
				if current := find(); distance(current[0], current[1]) < 40 {
					t.Fatal("跨越卡片间隙导致堆叠收起")
				}
				tt.Move(1, 1)
				tt.Frame()
				if rs = find(); distance(rs[0], rs[1]) > 20 {
					t.Fatal("移出提示后未收起堆叠")
				}
				tt.Key(0, ui.KeyTab)
				tt.Frame()
				if !tt.Focused(locales.CloseToast()) || distance(find()[0], find()[1]) < 40 {
					t.Fatal("键盘焦点进入后未展开堆叠")
				}
			})
		}
	}
}

func TestToastsPauseAndExpire(t *testing.T) {
	for _, pause := range []string{"悬停", "焦点"} {
		t.Run(pause, func(t *testing.T) {
			added, closed := false, 0
			tt := ui.NewTester(func(c *ui.Context) {
				if !added {
					added = true
					c.AddToast(ui.Toast{Title: "暂时提示", Timeout: time.Second, OnClose: func() { closed++ }})
				}
				Toasts(c, ToastOptions{})
			}, 480, 360)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			switch pause {
			case "悬停":
				r, _ := tt.Find("暂时提示")
				tt.Move(r.X+1, r.Y+1)
			case "焦点":
				tt.Key(0, ui.KeyTab)
			}
			tt.Advance(10 * time.Second)
			if !tt.HasText("暂时提示") || closed != 0 {
				t.Fatal("暂停计时期间提示被关闭")
			}
			switch pause {
			case "悬停":
				tt.Move(1, 1)
			case "焦点":
				tt.ClickAt(1, 1)
			}
			tt.Advance(2 * time.Second)
			if tt.HasText("暂时提示") || closed != 1 {
				t.Fatal("恢复计时后提示未正常超时关闭")
			}
		})
	}
}

func TestToastsExpireInBackground(t *testing.T) {
	added, closed := false, 0
	tt := ui.NewTester(func(c *ui.Context) {
		if !added {
			added = true
			c.AddToast(ui.Toast{Title: "暂时提示", Timeout: time.Second, OnClose: func() { closed++ }})
		}
		Toasts(c, ToastOptions{})
	}, 480, 360)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(locales.CloseToast()) {
		t.Fatal("键盘未进入提示")
	}
	tt.SetFocused(false)
	tt.Advance(2 * time.Second)
	if tt.HasText("暂时提示") || closed != 1 {
		t.Fatal("窗口进入后台后提示未正常超时关闭")
	}
}

func TestToastsYieldToEarlierViewport(t *testing.T) {
	added := false
	tt := ui.NewTester(func(c *ui.Context) {
		if !added {
			added = true
			c.AddToast(ui.Toast{Title: "已有容器", Timeout: -1})
		}
		ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
			for _, toast := range toasts {
				ui.ToastBase(c, toast).Root.Children(func() { ui.Text(c, toast.Title) })
			}
		})
		Toasts(c, ToastOptions{})
	}, 480, 360)
	if !tt.HasText("已有容器") || tt.HasText(locales.CloseToast()) {
		t.Fatal("后备容器覆盖了已注册的容器")
	}
}

func TestToastsLayoutAcrossThemesAndScales(t *testing.T) {
	for _, tc := range []struct {
		width int
		dark  bool
		scale float32
		kind  string
	}{
		{320, false, 1, ToastInfo},
		{320, true, 1.5, ToastSuccess},
		{800, false, 1.5, ToastWarning},
		{800, true, 1, ToastError},
		{320, false, 1.5, ""},
	} {
		width, dark, scale, kind := tc.width, tc.dark, tc.scale, tc.kind
		t.Run(fmt.Sprintf("%d/dark=%v/scale=%g/%s", width, dark, scale, kind), func(t *testing.T) {
			added := false
			tt := ui.NewTester(func(c *ui.Context) {
				theme := *c.Theme()
				theme.FontSize *= scale
				c.SetTheme(&theme)
				if !added {
					added = true
					c.AddToast(ui.Toast{Title: "JSON 格式错误：第 123 行，第 45 列", Description: "请检查引号、逗号与括号是否匹配。", Type: kind, Timeout: -1})
				}
				Toasts(c, ToastOptions{})
			}, width, 400)
			tt.SetDark(dark)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			tt.Advance(time.Second)
			for _, label := range []string{"JSON 格式错误：第 123 行，第 45 列", "请检查引号、逗号与括号是否匹配。", locales.CloseToast()} {
				r, ok := tt.Find(label)
				if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(width)+1 || r.Y+r.H > 400 {
					t.Fatalf("提示内容超出窗口：%q / %+v", label, r)
				}
			}
			if tt.FramePending() {
				t.Fatal("动画结束后提示仍持续重绘")
			}
			if directory := os.Getenv("MYJSON_TEST_SCREENSHOTS"); directory != "" {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("toast-%d-dark-%v-scale-%g-%s.png", width, dark, scale, kind)
				f, err := os.Create(filepath.Join(directory, name))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatalf("保存提示截图失败：%v / %v", err, closeErr)
				}
			}
		})
	}
}
