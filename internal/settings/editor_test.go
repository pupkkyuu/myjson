package settings

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/locales"
)

func revealSetting(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	panel, _ := tt.Find(locales.Settings())
	tt.Scroll(panel.X+panel.W-25, panel.Y+panel.H/2, 0, -10000)
	for range 30 {
		r, ok := tt.Find(locales.SettingControl(label))
		if ok && r.W > 0 && r.H >= 20 && r.Y >= panel.Y+64 && r.Y+r.H < panel.Y+panel.H-4 {
			return
		}
		tt.Scroll(panel.X+panel.W-25, panel.Y+panel.H/2, 0, 60)
	}
	t.Fatalf("设置项 %q 无法滚动访问", label)
}

func chooseSetting(t *testing.T, tt *ui.Tester, label, choice string) {
	t.Helper()
	revealSetting(t, tt, label)
	if err := tt.Click(locales.SettingControl(label)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(choice); err != nil {
		t.Fatal(err)
	}
}

func TestEditorAndDiffSettingsControls(t *testing.T) {
	state := State{Open: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 1000, 650)
	if err := tt.Click(locales.EditorSettings()); err != nil {
		t.Fatal(err)
	}
	p := state.Preferences()
	if err := tt.Click(locales.SettingControl(locales.EditorFontSize())); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnd)
	if p.FontSize != 28 {
		t.Fatalf("字号未响应键盘：%g", p.FontSize)
	}
	chooseSetting(t, tt, locales.EditorLineHeight(), "1.8")
	chooseSetting(t, tt, locales.EditorIndentWidth(), "4")
	chooseSetting(t, tt, locales.EditorWhitespace(), locales.WhitespaceTrailing())
	for _, label := range []string{locales.EditorDefaultWrap(), locales.EditorIndentGuides(), locales.EditorActiveLine(), locales.EditorAutoClose()} {
		revealSetting(t, tt, label)
		if err := tt.Click(locales.SettingControl(label)); err != nil {
			t.Fatal(err)
		}
	}
	if p.LineHeight != 1.8 || p.TabSize != 4 || p.Whitespace != mygoeditor.WhitespaceTrailing ||
		!p.DefaultWrap || p.IndentGuides || !p.ActiveLine || p.AutoClose {
		t.Fatalf("编辑器控件未应用设置：%+v", p)
	}
	if err := tt.Click(locales.DiffSettings()); err != nil {
		t.Fatal(err)
	}
	chooseSetting(t, tt, locales.DiffLayout(), locales.DiffLayoutUnified())
	chooseSetting(t, tt, locales.DiffInlineHighlight(), locales.DiffInlineWord())
	if p.DiffLayout != mygoeditor.DiffUnified || p.DiffInline != mygoeditor.DiffInlineWord {
		t.Fatalf("对比控件未应用设置：%+v", p)
	}
	if err := tt.Click(locales.SettingControl(locales.DiffInlineHighlight())); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if !state.Open || !tt.Focused(locales.SettingControl(locales.DiffInlineHighlight())) {
		t.Fatal("Esc 应先关闭下拉并保留控件焦点")
	}
	tt.Key(0, ui.KeyEscape)
	state.Open = true
	tt.Frame()
	if state.Category != diffCategory || state.Preferences() != p || p.TabSize != 4 {
		t.Fatal("重新打开设置未保留分类和会话偏好")
	}
}

func TestNewSettingsCategoriesLayout(t *testing.T) {
	for _, width := range []int{400, 760, 1000} {
		for _, scale := range []float32{1, 1.5} {
			if width == 400 && scale != 1 {
				continue
			}
			for _, dark := range []bool{false, true} {
				for _, category := range []string{editorCategory, diffCategory} {
					t.Run(fmt.Sprintf("%s/%d/scale=%g/dark=%v", category, width, scale, dark), func(t *testing.T) {
						state := State{Open: true, Category: category}
						tt := ui.NewTester(func(c *ui.Context) {
							theme := *c.Theme()
							theme.FontSize *= scale
							c.SetTheme(&theme)
							View(c, &state, max(48, theme.Rem(3)))
						}, width, 400)
						tt.SetDark(dark)
						labels := []string{locales.DiffLayout(), locales.DiffInlineHighlight()}
						if category == editorCategory {
							labels = []string{locales.EditorFontSize(), locales.EditorLineHeight(), locales.EditorIndentWidth(), locales.EditorDefaultWrap(),
								locales.EditorWhitespace(), locales.EditorIndentGuides(), locales.EditorActiveLine(), locales.EditorAutoClose()}
						}
						panel, _ := tt.Find(locales.Settings())
						for _, label := range labels {
							revealSetting(t, tt, label)
							r, ok := tt.Find(locales.SettingControl(label))
							if !ok || r.W <= 0 || r.H <= 0 || r.X < panel.X || r.X+r.W > panel.X+panel.W+1 ||
								r.Y < panel.Y || r.Y+r.H > panel.Y+panel.H+1 {
								t.Fatalf("设置控件不可滚动访问：%q %+v，面板 %+v", label, r, panel)
							}
						}
						selectLabel, optionLabel := locales.DiffLayout(), locales.DiffLayoutUnified()
						if category == editorCategory {
							selectLabel, optionLabel = locales.EditorWhitespace(), locales.WhitespaceTrailing()
						}
						revealSetting(t, tt, selectLabel)
						if err := tt.Click(locales.SettingControl(selectLabel)); err != nil {
							t.Fatal(err)
						}
						option, ok := tt.Find(optionLabel)
						if !ok || option.X < 0 || option.Y < 0 || option.X+option.W > float32(width)+1 || option.Y+option.H > 401 {
							t.Fatalf("下拉选项超出窗口：%q %+v", optionLabel, option)
						}
						tt.Key(0, ui.KeyEscape)
						revealSetting(t, tt, labels[len(labels)-1])
						if dir := os.Getenv("MYJSON_TEST_SCREENSHOTS"); dir != "" {
							if err := os.MkdirAll(dir, 0755); err != nil {
								t.Fatal(err)
							}
							for _, position := range []string{"bottom", "top"} {
								if position == "top" {
									tt.Scroll(panel.X+panel.W-25, panel.Y+panel.H/2, 0, -10000)
								}
								name := fmt.Sprintf("%s-%d-scale-%g-dark-%v-%s.png", category, width, scale, dark, position)
								f, err := os.Create(filepath.Join(dir, name))
								if err != nil {
									t.Fatal(err)
								}
								err = png.Encode(f, tt.Image())
								closeErr := f.Close()
								if err != nil || closeErr != nil {
									t.Fatalf("保存截图：%v / %v", err, closeErr)
								}
							}
						}
					})
				}
			}
		}
	}
}
