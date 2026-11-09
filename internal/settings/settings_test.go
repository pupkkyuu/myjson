package settings

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestThemeChoiceLabels(t *testing.T) {
	for source, want := range map[mygo.ThemeSource]string{
		mygo.ThemeSystem: locales.SystemTheme(), mygo.ThemeLight: locales.LightTheme(), mygo.ThemeDark: locales.DarkTheme(), "未知": locales.SystemTheme(),
	} {
		if got := themeChoiceLabel(source); got != want {
			t.Errorf("主题 %q 的标签 = %q，期望 %q", source, got, want)
		}
	}
}

func TestSettingsDialogCanCloseAtDifferentSizes(t *testing.T) {
	for _, tc := range []struct {
		width   int
		closeBy string
	}{{1000, "按钮"}, {760, "Esc"}, {400, "遮罩"}} {
		t.Run(fmt.Sprintf("宽度%d/%s", tc.width, tc.closeBy), func(t *testing.T) {
			state := State{Open: true}
			tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, tc.width, 650)
			switch tc.closeBy {
			case "按钮":
				if err := tt.Click(locales.CloseSettings()); err != nil {
					t.Fatal(err)
				}
			case "Esc":
				tt.Key(0, ui.KeyEscape)
			case "遮罩":
				tt.ClickAt(2, 58)
			}
			if state.Open {
				t.Fatal("关闭操作未收起设置面板")
			}
		})
	}
}

func TestThemeSelectorAppliesAllChoices(t *testing.T) {
	choice := mygo.ThemeSystem
	var applied []mygo.ThemeSource
	tt := ui.NewTester(func(c *ui.Context) {
		renderThemeSelector(c, choice, func(c *ui.Context, source mygo.ThemeSource) {
			choice = source
			applied = append(applied, source)
			c.Invalidate()
		})
	}, 500, 400)
	for _, source := range []mygo.ThemeSource{mygo.ThemeDark, mygo.ThemeLight, mygo.ThemeSystem} {
		if err := tt.Click(locales.ThemeSelector()); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(themeChoiceLabel(source)); err != nil {
			t.Fatal(err)
		}
		if got := choice; got != source {
			t.Fatalf("选择主题后来源 = %q，期望 %q", got, source)
		}
	}
	if len(applied) != 3 {
		t.Fatalf("主题选择回调触发 %d 次，期望 3 次", len(applied))
	}
}

func TestClosedSettingsRenderNoControls(t *testing.T) {
	state := State{}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 800, 600)
	if _, ok := tt.Find(locales.ThemeSelector()); ok {
		t.Fatal("关闭的设置面板不应显示控件")
	}
}

func TestSettingsCategoryState(t *testing.T) {
	state := State{Open: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 1000, 650)
	if state.Category != appearanceCategory || !tt.HasText(locales.ThemeDescription()) {
		t.Fatal("首次打开未默认选择外观")
	}
	state.Category = "future-category"
	tt.Frame()
	if state.Category != "future-category" || tt.HasText(locales.ThemeDescription()) {
		t.Fatal("未知分类应保留 ID 并显示空白内容")
	}
	if _, ok := tt.Find(locales.ThemeSelector()); ok {
		t.Fatal("未知分类不应渲染主题选择器")
	}
	if _, ok := tt.Find(locales.CloseSettings()); !ok {
		t.Fatal("空白页丢失关闭按钮")
	}
	tt.Key(0, ui.KeyEscape)
	state.Open = true
	tt.Frame()
	if state.Category != "future-category" || tt.HasText(locales.ThemeDescription()) {
		t.Fatal("重新打开设置重置了未知分类")
	}
	if err := tt.Click(locales.Appearance()); err != nil {
		t.Fatal(err)
	}
	if state.Category != appearanceCategory || !tt.HasText(locales.ThemeDescription()) {
		t.Fatal("点击分类未返回外观页")
	}
	for _, key := range []ui.Key{ui.KeyDown, ui.KeyUp, ui.KeyHome, ui.KeyEnd} {
		state.Category = "future-category"
		tt.Frame()
		for i := 0; i < 4 && !tt.Focused(locales.Appearance()); i++ {
			tt.Key(0, ui.KeyTab)
		}
		if !tt.Focused(locales.Appearance()) {
			t.Fatal("键盘无法聚焦外观分类")
		}
		tt.Key(0, key)
		wantCategory, wantLabel := appearanceCategory, locales.Appearance()
		if key == ui.KeyEnd {
			wantCategory, wantLabel = aboutCategory, locales.About()
		}
		if state.Category != wantCategory {
			t.Fatalf("%v 激活分类 %q，期望 %q", key, state.Category, wantCategory)
		}
		if !tt.Focused(wantLabel) {
			t.Fatal("选择分类后焦点应留在侧栏")
		}
	}
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(locales.CloseSettings()) {
		t.Fatal("Tab 应离开分类侧栏并进入内容区域")
	}
	tt.Key(0, ui.KeyTab)
	if control, _ := settingControl(tt, locales.Theme()); !control.Focused {
		t.Fatal("Tab 应继续进入主题选择器")
	}
}

func TestSettingsSidebarLayoutWithoutSearch(t *testing.T) {
	for _, size := range []struct{ width, height int }{{1000, 650}, {760, 650}, {760, 400}, {400, 650}} {
		width, height := size.width, size.height
		for _, dark := range []bool{false, true} {
			for _, scale := range []float32{1, 1.25, 1.5} {
				if width == 400 && scale != 1 {
					continue
				}
				t.Run(fmt.Sprintf("%dx%d/dark=%v/scale=%g", width, height, dark, scale), func(t *testing.T) {
					state := State{Open: true}
					var headingHeight, navigationInset float32
					tt := ui.NewTester(func(c *ui.Context) {
						theme := *c.Theme()
						theme.FontSize *= scale
						c.SetTheme(&theme)
						headingHeight = theme.Space(14)
						navigationInset = theme.Space(6)
						View(c, &state, max(48, theme.Rem(3)))
					}, width, height)
					tt.SetDark(dark)
					panel, panelOK := tt.Find(locales.Settings())
					nav, navOK := tt.Find(locales.Appearance())
					description, descriptionOK := tt.Find(locales.ThemeDescription())
					selectorInfo, selectorOK := settingControl(tt, locales.Theme())
					selector := selectorInfo.Bounds
					closeButton, closeOK := tt.Find(locales.CloseSettings())
					if !panelOK || !navOK || !descriptionOK || !selectorOK || !closeOK {
						t.Fatalf("设置布局丢失控件：%v", tt.Texts())
					}
					if nav.X+nav.W >= description.X || selector.X < description.X-1 {
						t.Fatalf("未保持左侧分类、右侧内容布局：%+v / %+v / %+v", nav, description, selector)
					}
					if gap := nav.Y - panel.Y - headingHeight; gap < 0 || gap > navigationInset {
						t.Fatalf("分类与标题重叠或保留了搜索占位：面板 %+v / 分类 %+v", panel, nav)
					}
					for _, rect := range []ui.Rect{nav, description, selector, closeButton} {
						if rect.W <= 0 || rect.H <= 0 || rect.X < panel.X-1 || rect.Y < panel.Y-1 ||
							rect.X+rect.W > panel.X+panel.W+1 || rect.Y+rect.H > panel.Y+panel.H+1 {
							t.Fatalf("控件超出设置面板：%+v / %+v", rect, panel)
						}
					}
					for _, text := range tt.Texts() {
						if strings.Contains(text, "搜索") || strings.Contains(strings.ToLower(text), "search") {
							t.Fatalf("仍渲染设置搜索：%q", text)
						}
					}
					if directory := os.Getenv("MYJSON_TEST_SCREENSHOTS"); directory != "" {
						if err := os.MkdirAll(directory, 0755); err != nil {
							t.Fatal(err)
						}
						name := fmt.Sprintf("settings-%dx%d-dark-%v-scale-%g.png", width, height, dark, scale)
						f, err := os.Create(filepath.Join(directory, name))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, tt.Image())
						closeErr := f.Close()
						if err != nil || closeErr != nil {
							t.Fatalf("保存设置截图失败：%v / %v", err, closeErr)
						}
					}
					clickSettingControl(t, tt, locales.Theme())
					for _, label := range []string{locales.LightTheme(), locales.DarkTheme()} {
						r, ok := tt.Find(label)
						if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(width)+1 || r.Y+r.H > float32(height)+1 {
							t.Fatalf("主题选项越过窗口边界：%q / %+v", label, r)
						}
					}
				})
			}
		}
	}
}

func TestSettingsPopupDismissal(t *testing.T) {
	state := State{Open: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 760, 650)
	popupVisible := func() bool { return tt.HasText(locales.LightTheme()) && tt.HasText(locales.DarkTheme()) }
	for _, closeBy := range []string{"Esc", "面板空白"} {
		clickSettingControl(t, tt, locales.Theme())
		if !popupVisible() {
			t.Fatal("点击选择器未展开主题选项")
		}
		if closeBy == "Esc" {
			tt.Key(0, ui.KeyEscape)
		} else {
			panel, _ := tt.Find(locales.Settings())
			tt.ClickAt(panel.X+panel.W-20, panel.Y+panel.H-20)
		}
		if !state.Open || popupVisible() {
			t.Fatalf("%s 未仅关闭下拉框", closeBy)
		}
		control, _ := settingControl(tt, locales.Theme())
		if closeBy == "Esc" && !control.Focused {
			t.Fatal("Esc 关闭下拉后未保留选择器焦点")
		}
	}
	tt.Key(0, ui.KeyEscape)
	if state.Open {
		t.Fatal("下拉关闭后 Esc 未关闭设置弹窗")
	}
}

func TestThemeSelectorKeyboardNavigation(t *testing.T) {
	choice := mygo.ThemeSystem
	var applied []mygo.ThemeSource
	tt := ui.NewTester(func(c *ui.Context) {
		renderThemeSelector(c, choice, func(c *ui.Context, source mygo.ThemeSource) {
			choice = source
			applied = append(applied, source)
			c.Invalidate()
		})
	}, 500, 400)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused(locales.ThemeSelector()) {
		t.Fatal("Tab 未聚焦主题选择器")
	}
	for _, source := range []mygo.ThemeSource{mygo.ThemeLight, mygo.ThemeDark} {
		tt.Key(0, ui.KeyDown)
		tt.Key(0, ui.KeyDown)
		tt.Key(0, ui.KeyEnter)
		if choice != source {
			t.Fatalf("键盘选择主题 = %q，期望 %q", choice, source)
		}
	}
	if len(applied) != 2 {
		t.Fatalf("键盘选择回调触发 %d 次，期望 2 次", len(applied))
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEscape)
	if choice != mygo.ThemeDark || len(applied) != 2 {
		t.Fatal("Esc 取消选择仍应用了高亮选项")
	}
}

func TestAppearanceTitlesFocusTheirSelectors(t *testing.T) {
	state := State{Open: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 1000, 650)
	for _, label := range []string{locales.Theme(), locales.InterfaceZoom()} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if a, ok := tt.AccessByRole(ui.RolePopUpButton, label); !ok || !a.Focused {
			t.Fatalf("点击 %q 标题未聚焦选择器：%+v，存在=%v", label, a, ok)
		}
	}
	state.Category = colorCategory
	tt.Frame()
	for _, label := range []string{locales.LightAppearance(), locales.DarkAppearance()} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if a, ok := tt.AccessByRole(ui.RolePopUpButton, label); !ok || !a.Focused {
			t.Fatalf("点击 %q 标题未聚焦选择器：%+v，存在=%v", label, a, ok)
		}
	}
}
