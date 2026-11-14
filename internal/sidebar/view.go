package sidebar

import (
	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/locales"
	"myjson/pkg/icons/lucide"
)

// View 渲染导航并返回本帧触发的操作。
func View(c *ui.Context, options Options) Event {
	event := Event{}
	selected := options.ActiveID
	root := ui.Column(c.Key("sidebar")).Fill().Disabled(options.Disabled)
	last := ui.Local(root, "selection", func() string { return "" })
	reveal := *last != options.ActiveID
	*last = options.ActiveID
	root.Children(func() {
		t := c.Theme()
		if options.TopInset > 0 {
			// 这一行只用来拖动窗口。开关由应用叠在侧栏和卡片之间，收起时跟着滑过去。
			ui.Row(c.Key("sidebar-bar")).FillWidth().Shrink(0).Height(options.TopInset).DragWindow()
		}
		ui.Column(c.Key("sidebar-actions")).FillWidth().Shrink(0).Padding(t.Space(2.5), t.Space(2.5), 0, t.Space(2.5)).Gap(t.Space(1)).Children(func() {
			for _, action := range []struct {
				key, label string
				icon       *ui.SVG
				action     Action
			}{
				{"new-format", locales.NewFormat(), icons.Plus(), NewFormat},
				{"new-comparison", locales.NewComparison(), icons.FileDiff(), NewComparison},
			} {
				button := ui.Button(c.Key(action.key), "").Label(action.label).FillWidth().MinHeight(t.Space(9)).Justify(ui.Start).KeepFocus().Border(0, ui.Transparent)
				background := ui.Transparent
				if !button.IsDisabled() {
					if button.Pressed() {
						background = t.SurfacePressed
					} else if button.Hovered() {
						background = t.SurfaceHover
					}
				}
				if button.Background(background).Children(func() {
					ui.Icon(c, action.icon).Size(t.Space(4.5), t.Space(4.5)).TextColor(t.Accent)
					ui.Text(c, action.label).SingleLine()
				}).Clicked() {
					event = Event{Action: action.action}
				}
			}
		})
		ui.Row(c.Key("tabs-heading")).FillWidth().Shrink(0).
			Padding(t.Space(3.5), t.Space(4.5), t.Space(1), t.Space(4.5)).Children(func() {
			ui.Text(c, locales.CountTabs(len(options.Tabs))).FontSize(t.FontSize - 1).
				FontWeight(600).TextColor(t.TextMuted).SingleLine()
		})
		if len(options.Tabs) == 0 {
			ui.Column(c.Key("empty-tabs")).Label(locales.TabList()).FillWidth().Grow(1).Shrink(1).MinHeight(0).
				Justify(ui.Center).AlignItems(ui.Center).Padding(t.Space(4.5)).Gap(t.Space(2)).Children(func() {
				ui.Icon(c, icons.Inbox()).Size(t.Space(9), t.Space(9)).TextColor(t.TextMuted)
				ui.Text(c, locales.EmptyTabs()).FontSize(t.FontSize - 1).FontWeight(500).
					TextColor(t.TextMuted).TextAlign(ui.Center).Wrap()
			})
			return
		}
		if ui.Sidebar(c.Key("tabs"), &selected, func() {
			for _, tab := range options.Tabs {
				item := ui.SidebarItem(c, tab.ID, nil, tab.Title).Padding(t.Space(1.5), t.Space(3.5))
				if reveal && tab.ID == options.ActiveID {
					item.ScrollIntoView()
				}
				over := item.Hovered()
				item.Children(func() {
					closeButton := component.IconButton(c.Key("close-"+tab.ID), icons.X(), locales.CloseTab(tab.Title)).
						Size(t.Space(6), t.Space(6)).Shrink(0).KeepFocus().Draw(nil)
					if !over {
						closeButton.Invisible()
					}
					if closeButton.Clicked() {
						event = Event{Action: CloseTab, TabID: tab.ID}
					}
				})
			}
		}).Padding(t.Space(.5), t.Space(2.5), t.Space(2.5), t.Space(2.5)).Background(ui.Transparent).Label(locales.TabList()).FocusGroup(ui.Vertical).FillWidth().Grow(1).Shrink(1).MinHeight(0).Changed() && event.Action == NoAction {
			event = Event{Action: ActivateTab, TabID: selected}
		}
	})
	return event
}

// FooterView 渲染底部入口并返回是否请求打开设置。
func FooterView(c *ui.Context, disabled bool) bool {
	t := c.Theme()
	button := component.CircleIconButton(c.Key("settings"), icons.Settings(), locales.WithShortcut(locales.Settings(), "⌘,")).
		Label(locales.Settings()).Disabled(disabled).KeepFocus().Draw(nil)
	if !button.IsDisabled() {
		if button.Pressed() {
			button.Background(t.SurfacePressed)
		} else if button.Hovered() {
			button.Background(t.SurfaceHover)
		}
	}
	clicked := button.Clicked()
	ui.Spacer(c)
	return clicked
}
