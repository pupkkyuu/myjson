package settings

import (
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func dialog(c *ui.Context, state *State, inset float32) {
	t := c.Theme()
	w, h := c.Size()
	dragHeight := min(max(0, inset), max(0, h))
	if inset <= 0 {
		inset = t.Space(5)
	}
	inset = max(inset, dragHeight)
	width := min(t.Space(180), w*.82, max(0, w-2*inset))
	height := min(t.Space(110), h*.75, max(0, h-2*inset))
	ui.Overlay(c, func() {
		ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(ui.RGBA(0, 0, 0, .18)).Transition(dialogFade)
		back := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			AlignItems(ui.Center).Padding(inset).Modal().Transition(dialogPop)
		if back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			state.Open = false
		}
		back.Children(func() {
			state.ZoomShortcuts(c)
			ui.Box(c).Grow(1).MinHeight(0)
			panel := ui.Column(c).Role(ui.RoleDialog).Label(locales.Settings()).
				Width(width).Height(height).Shrink(0).Padding(0).Gap(0).Clip().
				Radius(t.Radius*2).Material(glass.Glass{Tint: t.Background.Alpha(.96)}).TextColor(t.Text).
				Border(1, t.Background.Mix(t.Text, .25)).
				Shadow(0, t.Space(2), t.Space(6), 0, ui.RGBA(0, 0, 0, .12))
			panel.Clicked()
			panel.Children(func() { layout(c, state, width) })
			ui.Box(c).Grow(2).MinHeight(0)
			if dragHeight > 0 {
				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Height(dragHeight).DragWindow()
			}
		})
	})
}

var (
	dialogFade = ui.ElementTransition{Duration: 180 * time.Millisecond, Colors: true,
		Enter: &ui.Motion{Opacity: 0}, Exit: &ui.Motion{Opacity: 0}}
	dialogPop = ui.ElementTransition{Duration: 180 * time.Millisecond, Colors: true,
		Enter: &ui.Motion{Opacity: 0, Scale: .96, Y: 8}, Exit: &ui.Motion{Opacity: 0, Scale: .98}}
)
