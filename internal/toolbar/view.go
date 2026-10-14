package toolbar

import (
	"math"
	"slices"

	"github.com/egoist/mygo/plugins/loading"
	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/locales"
)

// View 渲染工具栏并返回本帧触发的操作。
func View(c *ui.Context, options Options) Action {
	if options.Kind == NoEditor {
		if options.Title != "" {
			ui.Text(c, options.Title).SingleLine()
		}
		return NoAction
	}

	action := NoAction
	t := c.Theme()
	ui.Row(c.Key("document-header")).Fill().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
		ui.Text(c, options.Title).SingleLine().Shrink(0).TextColor(ui.Transparent).
			DrawOver(func(p *ui.Painter, r ui.Rect) {
				// 对齐物理像素，避免高分屏动画时按整 DIP 跳动。
				scale := p.Scale()
				x := float32(math.Round(float64(r.X*scale))) / scale
				p.Text(x, r.Y, options.Title, t.FontSize, t.Text)
			})
		ui.Spacer(c)
		// 固定预留提示空间，避免任务开始或结束时工具按钮移动。
		ui.Box(c.Key("transform-progress-slot")).Size(t.FontSize*1.25, t.FontSize*1.25).Shrink(0).Children(func() {
			if options.ShowLoading && options.Loading {
				loading.Snake(c.Key("transform-progress"), loading.Options{Size: t.FontSize * 1.25, Color: t.TextMuted}).Label(locales.Transforming())
			}
		})
		// 焦点环绘制在按钮外，滚动视口四周需留出空间。
		ui.ScrollHorizontal(c.Key("editor-tools")).
			Label(locales.EditorToolbar()).MinWidth(0).Shrink(1).Padding(4).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(t.Space(.5)).Children(func() {
				for i, group := range toolGroups(options) {
					if i > 0 {
						ui.Box(c.Key(i)).Size(1, t.Rem(1.25)).Margin(0, t.Space(.5)).Background(t.Border)
					}
					for _, tool := range group {
						button := component.CircleIconButton(c.Key(string(tool.action)), tool.icon, tool.label).
							Label(tool.label).Disabled(tool.disabled || slices.Contains(options.DisabledActions, tool.action)).KeepFocus()
						if tool.action == ToggleWrap {
							button.Selected(options.WrapEnabled).TextColor(t.TextMuted)
							if options.WrapEnabled {
								button.Background(t.Text.Alpha(.1))
							}
						}
						if button.Clicked() && !(options.Loading && isTransform(tool.action)) {
							action = tool.action
						}
					}
				}
			})
		})
	})
	return action
}
