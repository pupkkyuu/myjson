package component

import (
	"github.com/egoist/mygo/ui"
)

// Choice pairs a select value with its display label.
type Choice[T comparable] struct {
	Value T
	Label string
}

// Select renders a styled select and returns its trigger for the caller to
// configure and handle changes. fallback is displayed when value is unknown.
func Select[T comparable](c *ui.Context, value *T, choices []Choice[T], fallback string) ui.Element {
	t := c.Theme()
	selectBox := ui.SelectBase(c, value)
	styleSelectTrigger(c, selectBox.Trigger)
	selectBox.Trigger.Children(func() {
		for _, option := range choices {
			if option.Value == *value {
				ui.Text(c, option.Label).SingleLine()
				return
			}
		}
		if fallback != "" {
			ui.Text(c, fallback).SingleLine()
		}
	})
	selectBox.Popup(func(panel ui.Element) {
		panel.Padding(t.Space(1)).Radius(t.Radius*1.5).
			Background(t.Background).Border(1, t.Border).
			Shadow(0, t.Space(1), t.Space(4), 0, ui.RGBA(0, 0, 0, .14))
		for _, option := range choices {
			item := selectBox.Item(option.Value).Label(option.Label).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).
				Background(ui.Transparent).TextColor(t.Text)
			if option.Value == *value {
				item.Background(t.Text.Alpha(.1))
			} else if item.Hovered() || item.Highlighted() {
				item.Background(t.Text.Alpha(.04))
			}
			item.Children(func() { ui.Text(c, option.Label) })
		}
	})
	return selectBox.Trigger
}

func styleSelectTrigger(c *ui.Context, trigger ui.Element) {
	t := c.Theme()
	trigger.Padding(t.Space(1.5), t.Space(6), t.Space(1.5), t.Space(2.5)).Radius(t.Radius).Justify(ui.Start)
	bg := t.Surface
	if !trigger.IsDisabled() {
		switch {
		case trigger.Pressed():
			bg = t.SurfacePressed
		case trigger.Hovered():
			bg = t.SurfaceHover
		}
	} else {
		trigger.Opacity(.45)
	}
	trigger.Background(bg).TextColor(t.Text).Border(1, t.Border).DrawOver(func(p *ui.Painter, r ui.Rect) {
		s := t.Space(2)
		cx, cy := r.X+r.W-t.Space(3), r.Y+r.H/2
		var path ui.Path
		path.MoveTo(cx-s*.35, cy-s*.15).LineTo(cx, cy+s*.2).LineTo(cx+s*.35, cy-s*.15)
		p.StrokePath(&path, 1.5, t.TextMuted)
	})
}
