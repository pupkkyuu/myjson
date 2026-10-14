package app

import "github.com/egoist/mygo/ui"

// resolveTheme 根据外观、所选主题、对比度和文字缩放生成新主题，避免调整跨帧累积。
// id 是主题的名字，未知时用默认主题。
func resolveTheme(c *ui.Context, id string) *ui.Theme {
	t := lookFor(id, c.Theme().Dark).theme
	p := c.Preferences()
	if p.HighContrast {
		t.Border = t.Border.Mix(t.Text, 0.45)
		t.TextMuted = t.TextMuted.Mix(t.Text, 0.4)
		t.Focus.A = 255
		t.Scrollbar = t.Scrollbar.Alpha(1.6)
	}
	if s := p.TextScale; s > 0 && s != 1 {
		t.FontSize *= s
	}
	return t
}
