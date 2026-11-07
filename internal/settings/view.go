package settings

import (
	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/locales"
	"myjson/pkg/icons"
)

func View(c *ui.Context, state *State, inset float32) {
	if !state.Open {
		return
	}

	if state.Category == "" {
		state.Category = appearanceCategory
	}
	dialog(c.Key("settings-dialog"), state, inset)
}

func layout(c *ui.Context, state *State, width float32) {
	t := c.Theme()
	sidebarWidth := min(t.Space(40), max(t.Space(28), width/4))
	contentWidth := max(0, width-sidebarWidth-1)
	ui.Row(c).Fill().MinWidth(0).Gap(0).Children(func() {
		ui.Column(c.Key("categories")).Width(sidebarWidth).FillHeight().Shrink(0).
			Background(t.Background).Children(func() {
			heading(c, locales.Settings(), nil)
			if ui.Sidebar(c.Key("category-sidebar"), &state.Category, func() {
				ui.SidebarItem(c, appearanceCategory, icons.Palette(), locales.Appearance())
				ui.SidebarItem(c, colorCategory, icons.Layers(), locales.ColorTheme())
				ui.SidebarItem(c, editorCategory, icons.Code(), locales.EditorSettings())
				ui.SidebarItem(c, diffCategory, icons.FileDiff(), locales.DiffSettings())
				ui.SidebarItem(c, aboutCategory, icons.Info(), locales.About())
			}).Background(t.Background).Label(locales.Settings()).FillWidth().Grow(1).Shrink(1).MinHeight(0).Changed() {
				c.Invalidate()
			}
		})
		ui.Box(c).Width(1).FillHeight().Shrink(0).Background(t.Border)
		ui.Column(c.Key("content")).Fill().MinWidth(0).Background(t.Surface).Children(func() {
			title := ""
			switch state.Category {
			case appearanceCategory:
				title = locales.Appearance()
			case colorCategory:
				title = locales.ColorTheme()
			case editorCategory:
				title = locales.EditorSettings()
			case diffCategory:
				title = locales.DiffSettings()
			case aboutCategory:
				title = locales.About()
			}
			heading(c, title, state)
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c.Key(state.Category)).FillWidth().MinWidth(0).Padding(t.Space(5)).Gap(t.Space(5)).Children(func() {
					switch state.Category {
					case appearanceCategory:
						appearance(c, contentWidth, state)
					case colorCategory:
						colors(c, contentWidth, state)
					case editorCategory:
						editorSettings(c, contentWidth, state.Preferences())
					case diffCategory:
						diffSettings(c, contentWidth, state.Preferences())
					case aboutCategory:
						about(c)
					}
				})
			})
		})
	})
}

func heading(c *ui.Context, title string, state *State) {
	t := c.Theme()
	ui.Row(c).FillWidth().MinWidth(0).MinHeight(t.Space(14)).Shrink(0).
		Padding(t.Space(3), t.Space(4)).Gap(t.Space(2)).Children(func() {
		ui.Text(c, title).Bold().MinWidth(0).SingleLine()
		ui.Spacer(c)
		if state != nil && closeButton(c).Clicked() {
			state.Open = false
		}
	})
}

func closeButton(c *ui.Context) ui.Element {
	button := component.IconButton(
		c.Key("close-settings"),
		icons.X(),
		locales.WithShortcut(locales.CloseSettings(), "Esc"),
	).Label(locales.CloseSettings())
	return button
}
