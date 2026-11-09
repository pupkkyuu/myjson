package settings

import (
	"strconv"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/locales"
)

func editorSettings(c *ui.Context, width float32, p *editor.Preferences) {
	settingRow(c, width, locales.EditorFontSize(), locales.EditorFontSizeDescription(), func(c *ui.Context, title ui.Element) {
		value := float64(p.FontSize)
		ui.Row(c).AlignItems(ui.Center).Gap(c.Theme().Space(2)).Children(func() {
			if ui.StepSlider(c, &value, 10, 28, 1).Width(c.Theme().Space(28)).Label(locales.SettingControl(locales.EditorFontSize())).LabelledBy(title).Changed() {
				p.FontSize = float32(value)
				c.Invalidate()
			}
			ui.Text(c, strconv.Itoa(int(p.FontSize))).FontFeatures("tnum").MinWidth(c.Theme().Space(5))
		})
	})
	settingRow(c, width, locales.EditorLineHeight(), locales.EditorLineHeightDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.EditorLineHeight(), &p.LineHeight, []choice[float32]{{1.3, "1.3"}, {1.5, "1.5"}, {1.8, "1.8"}})
	})
	settingRow(c, width, locales.EditorIndentWidth(), locales.EditorIndentWidthDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.EditorIndentWidth(), &p.TabSize, []choice[int]{{2, "2"}, {4, "4"}, {8, "8"}})
	})
	settingToggle(c, width, locales.EditorDefaultWrap(), locales.EditorDefaultWrapDescription(), &p.DefaultWrap)
	settingRow(c, width, locales.EditorWhitespace(), locales.EditorWhitespaceDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.EditorWhitespace(), &p.Whitespace, []choice[mygoeditor.Whitespace]{
			{mygoeditor.WhitespaceSelection, locales.WhitespaceSelection()},
			{mygoeditor.WhitespaceNone, locales.WhitespaceNone()},
			{mygoeditor.WhitespaceAll, locales.WhitespaceAll()},
			{mygoeditor.WhitespaceTrailing, locales.WhitespaceTrailing()},
			{mygoeditor.WhitespaceBoundary, locales.WhitespaceBoundary()},
		})
	})
	settingToggle(c, width, locales.EditorIndentGuides(), locales.EditorIndentGuidesDescription(), &p.IndentGuides)
	settingToggle(c, width, locales.EditorActiveLine(), locales.EditorActiveLineDescription(), &p.ActiveLine)
	settingToggle(c, width, locales.EditorAutoClose(), locales.EditorAutoCloseDescription(), &p.AutoClose)
}

func diffSettings(c *ui.Context, width float32, p *editor.Preferences) {
	settingRow(c, width, locales.DiffLayout(), locales.DiffLayoutDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.DiffLayout(), &p.DiffLayout, []choice[mygoeditor.DiffLayout]{
			{mygoeditor.DiffSplit, locales.DiffLayoutSplit()}, {mygoeditor.DiffUnified, locales.DiffLayoutUnified()},
		})
	})
	settingRow(c, width, locales.DiffInlineHighlight(), locales.DiffInlineHighlightDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.DiffInlineHighlight(), &p.DiffInline, []choice[mygoeditor.DiffInline]{
			{mygoeditor.DiffInlineChar, locales.DiffInlineChar()}, {mygoeditor.DiffInlineWord, locales.DiffInlineWord()},
			{mygoeditor.DiffInlineNone, locales.DiffInlineNone()},
		})
	})
}

func settingRow(c *ui.Context, width float32, label, description string, control func(*ui.Context, ui.Element)) {
	t := c.Theme()
	var row ui.Element
	if width < t.Space(116) {
		row = ui.Column(c)
	} else {
		row = ui.Row(c).AlignItems(ui.Center)
	}
	row.Key(label).FillWidth().MinWidth(0).Gap(t.Space(3)).Children(func() {
		var title ui.Element
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(1)).Children(func() {
			title = ui.Text(c, label).Wrap()
			ui.Text(c, description).TextColor(t.TextMuted).FontSize(t.Rem(.875)).Wrap()
		})
		control(c, title)
	})
}

func settingToggle(c *ui.Context, width float32, label, description string, value *bool) {
	settingRow(c, width, label, description, func(c *ui.Context, title ui.Element) {
		if ui.Switch(c, value).Label(locales.SettingControl(label)).LabelledBy(title).Shrink(0).Changed() {
			c.Invalidate()
		}
	})
}

type choice[T comparable] struct {
	value T
	label string
}

func preferenceSelect[T comparable](c *ui.Context, title ui.Element, label string, value *T, choices []choice[T]) {
	t := c.Theme()
	selectBox := ui.SelectBase(c, value)
	styleSelectTrigger(c, selectBox.Trigger)
	selectBox.Trigger.Label(locales.SettingControl(label)).LabelledBy(title).MinWidth(t.Space(32)).MaxWidthPercent(100).Children(func() {
		for _, option := range choices {
			if option.value == *value {
				ui.Text(c, option.label).SingleLine()
				break
			}
		}
	})
	selectBox.Popup(func(panel ui.Element) {
		panel.Padding(t.Space(1)).Radius(t.Radius*1.5).
			Material(glass.Glass{Tint: t.Background.Alpha(.92)}).Border(1, t.Border).
			Shadow(0, t.Space(1), t.Space(4), 0, ui.RGBA(0, 0, 0, .14))
		for _, option := range choices {
			item := selectBox.Item(option.value).Label(option.label).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).
				Background(ui.Transparent).TextColor(t.Text)
			if option.value == *value {
				item.Background(t.Text.Alpha(.1))
			} else if item.Hovered() || item.Highlighted() {
				item.Background(t.Text.Alpha(.04))
			}
			item.Children(func() { ui.Text(c, option.label) })
		}
	})
	if selectBox.Trigger.Changed() {
		c.Invalidate()
	}
}
