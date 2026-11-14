package settings

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestAboutMetadataAndNavigation(t *testing.T) {
	previous := mygo.App.Version()
	t.Cleanup(func() { mygo.App.SetVersion(previous) })
	mygo.App.SetVersion("1.2.3")
	state := State{Open: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, &state, 48) }, 1000, 650)
	if err := tt.Click(locales.About()); err != nil {
		t.Fatal(err)
	}
	if tt.HasText(locales.ApplicationName()) {
		t.Fatal("关于页面不应显示应用名称")
	}
	for _, value := range []string{"1.2.3", applicationRepository} {
		if !tt.HasText(value) {
			t.Fatalf("关于页面缺少 %q", value)
		}
	}
	if err := tt.Click(applicationRepository); err != nil {
		t.Fatal(err)
	}
	if urls := tt.OpenedURLs(); len(urls) != 1 || urls[0] != applicationRepository {
		t.Fatalf("仓库链接未打开正确地址：%v", urls)
	}
	tt.Key(0, ui.KeyEscape)
	state.Open = true
	tt.Frame()
	if state.Category != aboutCategory || !tt.HasText("1.2.3") {
		t.Fatal("重新打开设置未保留关于分类")
	}
	mygo.App.SetVersion("9.8.7")
	tt.Frame()
	if !tt.HasText("9.8.7") {
		t.Fatal("关于页面应显示运行时构建版本")
	}
}

func TestAboutLayout(t *testing.T) {
	for _, width := range []int{400, 760, 1000} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("width=%d/dark=%v", width, dark), func(t *testing.T) {
				state := State{Open: true, Category: aboutCategory}
				tt := ui.NewTester(func(c *ui.Context) {
					theme := *c.Theme()
					theme.FontSize *= 1.5
					c.SetTheme(&theme)
					View(c, &state, 48)
				}, width, 400)
				tt.SetDark(dark)
				panel, _ := tt.Find(locales.Settings())
				tt.Scroll(panel.X+panel.W-20, panel.Y+panel.H/2, 0, 10000)
				link, ok := tt.Find(applicationRepository)
				if !ok || link.W <= 0 || link.H <= 0 || link.X < panel.X || link.Y < panel.Y ||
					link.X+link.W > panel.X+panel.W+1 || link.Y+link.H > panel.Y+panel.H+1 {
					t.Fatalf("仓库链接超出设置面板或无法访问：%+v / %+v", link, panel)
				}
				if err := tt.Click(applicationRepository); err != nil {
					t.Fatal(err)
				}
				if len(tt.OpenedURLs()) != 1 {
					t.Fatal("窄窗口中无法点击仓库链接")
				}
				if dir := os.Getenv("MYJSON_TEST_SCREENSHOTS"); dir != "" {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("about-%d-dark-%v.png", width, dark)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					closeErr := f.Close()
					if err != nil || closeErr != nil {
						t.Fatalf("保存关于页截图：%v / %v", err, closeErr)
					}
				}
			})
		}
	}
}
