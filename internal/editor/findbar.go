package editor

import (
	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/locales"
)

// RenderFindBar 调整原生查找栏布局，输入和操作仍由查找栏处理。
func RenderFindBar(c *ui.Context, bar ui.Element, f *mygoeditor.FindBar) {
	t := c.Theme()
	tx := locales.In(c.Locale())
	bar.Left(t.Space(1)).Right(t.Space(4)).MaxWidth(t.Space(90)).Margin(0, 0, 0, ui.Auto).
		Gap(t.Space(1)).Padding(t.Space(1.5)).Background(t.Surface).Border(1, t.Border).Radius(t.Radius).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.18))
	field := func(value *string, placeholder string) ui.Element {
		return ui.TextInputBase(c, value).MinWidth(0).Placeholder(placeholder).Padding(t.Space(1), t.Space(2)).
			Background(t.Background).Border(1, t.Border).Radius(t.Radius - 2).FontSize(t.FontSize - 1).Label(placeholder)
	}
	button := func(label, tip string) ui.Element {
		b := ui.ButtonBase(c).Shrink(0).Padding(t.Space(0.5)).Radius(t.Radius - 2).Label(tip).Tooltip(tip).KeepFocus()
		b.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 1) })
		return b
	}
	bar.Children(func() {
		ui.Row(c.Key("query")).FillWidth().Children(func() {
			f.QueryInput(field(f.Query(), tx.FindQuery()).Grow(1))
		})
		if f.Replacing() {
			ui.Row(c.Key("replacement")).FillWidth().Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				f.ReplacementInput(field(f.Replacement(), tx.FindReplace()).Grow(1))
				if ui.Button(c, tx.FindReplace()).Shrink(0).Padding(t.Space(0.5), t.Space(2)).FontSize(t.FontSize - 1).Clicked() {
					f.Replace()
				}
				if ui.Button(c, tx.FindReplaceAll()).Shrink(0).Padding(t.Space(0.5), t.Space(2)).FontSize(t.FontSize - 1).Clicked() {
					f.ReplaceAll()
				}
			})
		}
		ui.Row(c.Key("options")).FillWidth().Gap(t.Space(0.5)).AlignItems(ui.Center).Children(func() {
			toggle := func(on *bool, label, tip string) {
				b := ui.ToggleBase(c, on).Shrink(0).Padding(t.Space(0.5)).Radius(t.Radius - 2).Label(tip).Tooltip(tip).KeepFocus()
				if *on {
					b.Background(t.Accent.Alpha(0.2)).TextColor(t.Accent)
				}
				b.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 1) })
			}
			toggle(f.CaseSensitive(), "Aa", tx.FindMatchCase())
			toggle(f.WholeWord(), "ab", tx.FindWholeWord())
			toggle(f.Regexp(), ".*", tx.FindRegexp())
			toggle(f.InSelection(), "[ ]", tx.FindInSelection())
			ui.Text(c, f.Status()).Grow(1).MinWidth(0).SingleLine().FontSize(t.FontSize - 2).TextColor(t.TextMuted).FontFeatures("tnum").Role(ui.RoleStatus)
			if button("↑", tx.FindPrevious()).Clicked() {
				f.Previous()
			}
			if button("↓", tx.FindNext()).Clicked() {
				f.Next()
			}
			if f.CanReplace() && button("⇄", tx.FindReplace()).Clicked() {
				f.SetReplacing(!f.Replacing())
				c.Invalidate()
			}
			if button("✕", tx.FindClose()).Clicked() {
				f.Close()
			}
		})
	})
}
