package settings

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"myjson/internal/locales"
)

func TestColorSettingsSeparateAndPreview(t *testing.T) {
	state := State{Open: true, Category: colorCategory}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 1000, 650)
	for _, tc := range []struct {
		dark      bool
		id, label string
	}{{false, "rex", locales.WashPetal()}, {true, "tide", locales.WashTide()}} {
		label := locales.LightAppearance()
		if tc.dark {
			label = locales.DarkAppearance()
		}
		clickSettingControl(t, tt, label)
		if err := tt.Click(tc.label); err != nil {
			t.Fatal(err)
		}
		if got := state.Wash(tc.dark); got != tc.id {
			t.Fatalf("选择 %s = %q，期望 %q", label, got, tc.id)
		}
	}
	if state.Wash(false) != "rex" || state.Wash(true) != "tide" {
		t.Fatal("明暗配色应独立保存")
	}
	for _, dark := range []bool{false, true} {
		id := "paper"
		preview := ui.NewTester(func(c *ui.Context) { washPreview(c, 300, dark, id) }, 300, 100)
		before := preview.Image().RGBAAt(10, 10)
		id = "rex"
		preview.Frame()
		if after := preview.Image().RGBAAt(10, 10); before == after {
			t.Fatalf("dark=%v 色卡未随选择变化：%v", dark, before)
		}
	}
}

func TestColorSettingsLayout(t *testing.T) {
	for _, tc := range []struct {
		w, h  int
		scale float32
	}{{1000, 650, 1}, {760, 650, 1}, {760, 400, 1}, {400, 650, 1}, {760, 650, 1.5}} {
		t.Run(fmt.Sprintf("%dx%d-%g", tc.w, tc.h, tc.scale), func(t *testing.T) {
			state := State{Open: true, Category: colorCategory}
			tt := ui.NewTester(func(c *ui.Context) {
				theme := *c.Theme()
				theme.FontSize *= tc.scale
				c.SetTheme(&theme)
				View(c, &state, 48)
			}, tc.w, tc.h)
			for _, label := range []string{locales.LightAppearance(), locales.DarkAppearance()} {
				control, ok := settingControl(tt, label)
				r := control.Bounds
				if !ok || r.W <= 0 || r.X < 0 || r.X+r.W > float32(tc.w)+1 {
					t.Fatalf("配色选择器越界 %s: %+v", label, r)
				}
			}
			if dir := os.Getenv("MYJSON_TEST_SCREENSHOTS"); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("colors-%dx%d-%g.png", tc.w, tc.h, tc.scale)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				cerr := f.Close()
				if err != nil || cerr != nil {
					t.Fatalf("保存截图失败：%v / %v", err, cerr)
				}
			}
		})
	}
}
