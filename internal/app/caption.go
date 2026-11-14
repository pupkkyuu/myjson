package app

import (
	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/locales"
	"myjson/pkg/icons/lucide"
)

type windowCommands interface {
	Minimize()
	ToggleMaximize()
	IsMaximized() bool
	Close()
}

func captionButtons(c *ui.Context, window windowCommands) {
	ui.Row(c.Key("window-controls")).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		if component.CircleIconButton(c.Key("minimize"), icons.Minus(), locales.MinimizeWindow()).Clicked() && window != nil {
			window.Minimize()
		}
		icon, label := icons.Square(), locales.MaximizeWindow()
		if window != nil && window.IsMaximized() {
			icon, label = icons.Copy(), locales.RestoreWindow()
		}
		if component.CircleIconButton(c.Key("maximize"), icon, label).MaximizeButton().Clicked() && window != nil {
			window.ToggleMaximize()
			c.Invalidate()
		}
		closeButton := component.CircleIconButton(c.Key("close-window"), icons.X(), locales.CloseWindow())
		if closeButton.Hovered() || closeButton.Pressed() {
			closeButton.TextColor(ui.RGB(255, 255, 255))
		}
		var face ui.Color
		if closeButton.Pressed() {
			face = ui.RGB(196, 43, 28)
		} else if closeButton.Hovered() {
			face = ui.RGB(232, 17, 35)
		}
		closeButton.Draw(func(p *ui.Painter, r ui.Rect) {
			if face.A > 0 {
				p.Fill(r, face, min(r.W, r.H)/2)
			}
		})
		if closeButton.Clicked() && window != nil {
			window.Close()
		}
	})
}
