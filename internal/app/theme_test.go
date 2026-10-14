package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestApplicationThemeFollowsFrameWithoutAccumulatingScale(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	for _, dark := range []bool{false, true} {
		tt.SetDark(dark)
		for _, contrast := range []bool{false, true} {
			tt.SetPreferences(ui.Preferences{TextScale: 1.5, HighContrast: contrast})
			base := lookFor(defaultWash, dark).theme
			for range 3 {
				tt.Frame()
				if app.theme.Dark != dark || app.theme.FontSize != base.FontSize*1.5 || app.theme.Accent != base.Accent || app.theme.Background != base.Background {
					t.Fatalf("主题状态错误或字号发生累积：%+v", app.theme)
				}
				if contrast && (app.theme.Border == base.Border || app.theme.Focus.A != 255) {
					t.Fatal("高对比度偏好未应用")
				}
			}
		}
	}
	light := lookFor(defaultWash, false).theme
	darkTheme := lookFor(defaultWash, true).theme
	if light.Background != ui.Hex("#ffffff") || darkTheme.Background != ui.Hex("#232323") {
		t.Fatalf("默认主题的卡片色变了：浅色 %v，深色 %v", light.Background, darkTheme.Background)
	}
}

func TestWashPaintsTheWindow(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	paper := tt.Image().RGBAAt(40, 20)
	app.settingState.SetWash(false, "rex")
	tt.Frame()
	top := tt.Image().RGBAAt(40, 20)
	bottom := tt.Image().RGBAAt(40, 640)
	if top == paper {
		t.Fatal("切换主题没有改变窗口底色")
	}
	if top == bottom {
		t.Fatal("渐变主题的窗口上下应是不同的颜色")
	}
	app.settingState.SetWash(false, "plain")
	tt.Frame()
	if tt.Image().RGBAAt(40, 20) != tt.Image().RGBAAt(40, 640) {
		t.Fatal("纯色主题不应有渐变")
	}
	if lookFor("missing", false).theme.Background != lookFor(defaultWash, false).theme.Background {
		t.Fatal("未知主题应回退到默认主题")
	}
}
