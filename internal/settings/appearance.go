package settings

import (
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
	"myjson/internal/palette"
)

func appearance(c *ui.Context, contentWidth float32, state *State) {
	t := c.Theme()
	var row ui.Element
	if contentWidth < t.Space(116) {
		row = ui.Column(c)
	} else {
		row = ui.Row(c)
	}
	row.Key("theme").FillWidth().MinWidth(0).Gap(t.Space(3)).Children(func() {
		ui.Column(c).FillWidth().Shrink(1).MinWidth(0).Gap(t.Space(1)).Children(func() {
			ui.Text(c, locales.Theme()).Wrap()
			ui.Text(c, locales.ThemeDescription()).TextColor(t.TextMuted).FontSize(t.Rem(.875)).Wrap()
		})
		usableWidth := max(0, contentWidth-2*t.Space(5))
		themeSelector(c).MinWidth(min(t.Space(32), usableWidth))
	})
	settingRow(c, contentWidth, locales.InterfaceZoom(), locales.InterfaceZoomDescription(), func(c *ui.Context) {
		value := state.ZoomPercent()
		choices := make([]choice[int], 0, len(zoomLevels))
		for _, level := range zoomLevels {
			choices = append(choices, choice[int]{level, strconv.Itoa(level) + "%"})
		}
		preferenceSelect(c, locales.InterfaceZoom(), &value, choices)
		state.SetZoomPercent(value)
	})
}

// washPreview 使用与窗口相同的渐变和界面颜色，展示对应外观的配色。
func washPreview(c *ui.Context, contentWidth float32, dark bool, id string) {
	t := c.Theme()
	look := palette.LookFor(id, dark)
	width := min(t.Space(52), max(0, contentWidth-2*t.Space(5)))
	key := "light-wash-preview"
	if dark {
		key = "dark-wash-preview"
	}
	ui.Box(c.Key(key)).Width(width).Height(t.Space(20)).Shrink(0).Radius(t.Radius).Border(1, t.Border).Clip().
		LinearGradient(look.Base).Children(func() {
		ui.Row(c).Fill().LinearGradient(look.Glow).Padding(t.Space(2)).Gap(t.Space(2)).Children(func() {
			// 侧边保留渐变，再用一张小卡片预览文字与强调色。
			ui.Box(c).Width(t.Space(7)).FillHeight().Shrink(0)
			ui.Column(c).Fill().MinWidth(0).Radius(t.Radius / 2).Background(look.Theme.Surface).
				Padding(t.Space(2)).Gap(t.Space(2)).Children(func() {
				ui.Row(c).FillWidth().Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(look.Theme.Accent)
					ui.Box(c).Width(t.Space(14)).Height(t.Space(1.25)).Radius(t.Space(1)).Background(look.Theme.Text)
				})
				ui.Box(c).Width(t.Space(25)).MaxWidthPercent(100).Height(t.Space(1)).Radius(t.Space(1)).Background(look.Theme.TextMuted.Alpha(.5))
				ui.Box(c).Width(t.Space(18)).MaxWidthPercent(100).Height(t.Space(1)).Radius(t.Space(1)).Background(look.Theme.Accent.Alpha(.65))
			})
		})
	})
}

// colors 是配色页：浅色和深色外观各自选一套窗口渐变和界面颜色。
func colors(c *ui.Context, contentWidth float32, state *State) {
	t := c.Theme()
	ui.Text(c, locales.ColorThemeDescription()).TextColor(t.TextMuted).FontSize(t.Rem(.875)).Wrap()
	washSetting(c, contentWidth, false, state)
	washSetting(c, contentWidth, true, state)
}

func washSetting(c *ui.Context, contentWidth float32, dark bool, state *State) {
	t := c.Theme()
	label, key := locales.LightAppearance(), "light-wash"
	if dark {
		label, key = locales.DarkAppearance(), "dark-wash"
	}
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(t.Space(3)).Children(func() {
		var row ui.Element
		if contentWidth < t.Space(100) {
			row = ui.Column(c)
		} else {
			row = ui.Row(c).AlignItems(ui.Center)
		}
		row.FillWidth().MinWidth(0).Gap(t.Space(2)).Children(func() {
			ui.Text(c, label).Grow(1).Wrap()
			washSelect(c, dark, state)
		})
		washPreview(c, contentWidth, dark, state.Wash(dark))
	})
}

func themeSelector(c *ui.Context) ui.Element {
	return renderThemeSelector(c, mygo.Theme.Source(), applyTheme)
}

func renderThemeSelector(c *ui.Context, choice mygo.ThemeSource, apply func(*ui.Context, mygo.ThemeSource)) ui.Element {
	t := c.Theme()
	selectBox := ui.SelectBase(c.Key("theme-selector"), &choice)
	styleSelectTrigger(c, selectBox.Trigger)

	selectBox.Trigger.
		Label(locales.ThemeSelector()).
		MinWidth(c.Theme().Space(32)).
		MaxWidthPercent(100).
		Children(func() {
			ui.Text(c, themeChoiceLabel(choice)).SingleLine()
		})

	selectBox.Popup(func(panel ui.Element) {
		panel.Padding(t.Space(1)).Radius(t.Radius*1.5).
			Material(glass.Glass{Tint: t.Background.Alpha(.92)}).Border(1, t.Border).
			Shadow(0, t.Space(1), t.Space(4), 0, ui.RGBA(0, 0, 0, .14))
		for _, value := range []mygo.ThemeSource{mygo.ThemeSystem, mygo.ThemeLight, mygo.ThemeDark} {
			label := themeChoiceLabel(value)
			item := selectBox.Item(value).Label(label).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).
				Background(ui.Transparent).TextColor(t.Text)
			if value == choice {
				item.Background(t.Text.Alpha(.1))
			} else if item.Hovered() || item.Highlighted() {
				item.Background(t.Text.Alpha(.04))
			}
			item.Children(func() { ui.Text(c, label) })
		}
	})

	if selectBox.Trigger.Changed() {
		apply(c, choice)
	}
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

// washSelect 选择一种外观用的配色。
func washSelect(c *ui.Context, dark bool, state *State) {
	label := locales.LightAppearance()
	if dark {
		label = locales.DarkAppearance()
	}
	value := state.Wash(dark)
	if value == "" {
		value = defaultWash
	}
	choices := make([]choice[string], 0, len(washIDs))
	for _, id := range washIDs {
		choices = append(choices, choice[string]{id, washLabel(id)})
	}
	preferenceSelect(c, label, &value, choices)
	if value != state.Wash(dark) {
		state.SetWash(dark, value)
		c.Invalidate()
	}
}

func washLabel(id string) string {
	switch id {
	case "rex":
		return locales.WashPetal()
	case "mist":
		return locales.WashMist()
	case "meadow":
		return locales.WashMeadow()
	case "dusk":
		return locales.WashDusk()
	case "tide":
		return locales.WashTide()
	case "apricot":
		return locales.WashApricot()
	case "aurora":
		return locales.WashAurora()
	case "dune":
		return locales.WashDune()
	case "plain":
		return locales.WashPlain()
	default:
		return locales.WashPaper()
	}
}

func themeChoiceLabel(source mygo.ThemeSource) string {
	switch source {
	case mygo.ThemeLight:
		return locales.LightTheme()
	case mygo.ThemeDark:
		return locales.DarkTheme()
	default:
		return locales.SystemTheme()
	}
}
