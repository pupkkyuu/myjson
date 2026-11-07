package component

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// IconButton 创建带无障碍标签和提示的图标按钮，操作由调用方处理。
func IconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	return iconButton(c, icon, label, false)
}

// CircleIconButton 使用圆形悬浮背景，保持图标按钮的尺寸和交互区域。
func CircleIconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	return iconButton(c, icon, label, true)
}

func iconButton(c *ui.Context, icon *ui.SVG, label string, circle bool) ui.Element {
	t := c.Theme()
	size := max(32, t.Rem(2))
	radius := t.Radius
	if circle {
		radius = size / 2
	}
	b := ui.ButtonBase(c).Size(size, size).Radius(radius).TextColor(t.TextMuted).Label(label)
	var face ui.Color
	resolved := false
	b.Draw(func(p *ui.Painter, r ui.Rect) {
		if !resolved {
			resolved = true
			switch {
			case b.IsDisabled():
			case b.Pressed():
				face = t.Text.Alpha(.14)
			case b.Hovered():
				face = t.Text.Alpha(.08)
			}
		}
		if face.A > 0 {
			if circle {
				p.Fill(r, face, min(r.W, r.H)/2)
			} else {
				p.Fill(r, face, radius)
			}
		}
	}).Children(func() { ui.Icon(c, icon).Size(t.Rem(1.25), t.Rem(1.25)) })
	if label != "" {
		b.Description(label)
		ui.TooltipBase(c, b, func(tip ui.Element) {
			tip.MaxWidth(t.Space(80)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).
				Material(glass.Glass{Style: glass.Clear, Tint: t.Text}).TextColor(t.Background).
				FontSize(t.FontSize-1).Shadow(0, t.Space(.5), t.Space(2), 0, ui.RGBA(0, 0, 0, .14)).
				Children(func() { ui.Text(c, label) })
		})
	}
	return b
}
