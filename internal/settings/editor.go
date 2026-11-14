package settings

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/component"
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
		preferenceSelect(c, title, locales.EditorLineHeight(), &p.LineHeight, []component.Choice[float32]{{Value: 1.3, Label: "1.3"}, {Value: 1.5, Label: "1.5"}, {Value: 1.8, Label: "1.8"}})
	})
	settingRow(c, width, locales.EditorIndentWidth(), locales.EditorIndentWidthDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.EditorIndentWidth(), &p.TabSize, []component.Choice[int]{{Value: 2, Label: "2"}, {Value: 4, Label: "4"}, {Value: 8, Label: "8"}})
	})
	settingToggle(c, width, locales.EditorDefaultWrap(), locales.EditorDefaultWrapDescription(), &p.DefaultWrap)
	settingRow(c, width, locales.EditorWhitespace(), locales.EditorWhitespaceDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.EditorWhitespace(), &p.Whitespace, []component.Choice[mygoeditor.Whitespace]{
			{Value: mygoeditor.WhitespaceSelection, Label: locales.WhitespaceSelection()},
			{Value: mygoeditor.WhitespaceNone, Label: locales.WhitespaceNone()},
			{Value: mygoeditor.WhitespaceAll, Label: locales.WhitespaceAll()},
			{Value: mygoeditor.WhitespaceTrailing, Label: locales.WhitespaceTrailing()},
			{Value: mygoeditor.WhitespaceBoundary, Label: locales.WhitespaceBoundary()},
		})
	})
	settingToggle(c, width, locales.EditorIndentGuides(), locales.EditorIndentGuidesDescription(), &p.IndentGuides)
	settingToggle(c, width, locales.EditorActiveLine(), locales.EditorActiveLineDescription(), &p.ActiveLine)
	settingToggle(c, width, locales.EditorAutoClose(), locales.EditorAutoCloseDescription(), &p.AutoClose)
}

func diffSettings(c *ui.Context, width float32, p *editor.Preferences) {
	settingRow(c, width, locales.DiffLayout(), locales.DiffLayoutDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.DiffLayout(), &p.DiffLayout, []component.Choice[mygoeditor.DiffLayout]{
			{Value: mygoeditor.DiffSplit, Label: locales.DiffLayoutSplit()}, {Value: mygoeditor.DiffUnified, Label: locales.DiffLayoutUnified()},
		})
	})
	settingRow(c, width, locales.DiffInlineHighlight(), locales.DiffInlineHighlightDescription(), func(c *ui.Context, title ui.Element) {
		preferenceSelect(c, title, locales.DiffInlineHighlight(), &p.DiffInline, []component.Choice[mygoeditor.DiffInline]{
			{Value: mygoeditor.DiffInlineChar, Label: locales.DiffInlineChar()}, {Value: mygoeditor.DiffInlineWord, Label: locales.DiffInlineWord()},
			{Value: mygoeditor.DiffInlineNone, Label: locales.DiffInlineNone()},
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

func preferenceSelect[T comparable](c *ui.Context, title ui.Element, label string, value *T, choices []component.Choice[T]) {
	t := c.Theme()
	trigger := component.Select(c, value, choices, "")
	trigger.Label(locales.SettingControl(label)).LabelledBy(title).MinWidth(t.Space(32)).MaxWidthPercent(100)
	if trigger.Changed() {
		c.Invalidate()
	}
}
