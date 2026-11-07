package component

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
	"myjson/pkg/icons"
)

const (
	ToastInfo    = "info"
	ToastSuccess = "success"
	ToastWarning = "warning"
	ToastError   = "error"
)

type ToastOptions struct {
	Bottom bool
	Inset  float32
}

// Toasts 渲染窗口的提示队列；同一帧以最先注册的提示容器为准。
func Toasts(c *ui.Context, options ToastOptions) ui.Element {
	t := c.Theme()
	width, _ := c.Size()
	margin, inset := t.Space(4), max(0, options.Inset)
	dir, origin := float32(1), float32(1)
	if options.Bottom {
		dir, origin = -1, 0
	}
	gap, peek := t.Space(2), t.Space(3.5)
	padY, border := t.Space(3), float32(1)
	spring := ui.Snappy
	motion := ui.ElementTransition{Animation: &spring, Size: true,
		Enter: &ui.Motion{Y: -dir * t.Space(16)}, Exit: &ui.Motion{Scale: .96}}
	return ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.AlignItems(ui.Center)
		if options.Bottom {
			viewport.Padding(margin, margin, margin+inset, margin)
		} else {
			viewport.Justify(ui.Start).Padding(margin+inset, margin, margin, margin)
		}
		stack := ui.Box(c).Key("myjson.toasts").Width(min(max(0, width-2*margin), t.Space(90)))
		expanded := stack.Hovered() || stack.FocusWithin()
		n := len(toasts)
		roots := make([]ui.Element, n)
		heights := make([]float32, n)
		stack.Children(func() {
			for i, toast := range toasts {
				parts := ui.ToastBase(c, toast)
				roots[i] = parts.Root.Absolute().Left(0).Right(0).Clip().
					Padding(padY, t.Space(2), padY, t.Space(4)).Radius(t.Radius*1.5).
					Material(glass.Glass{Tint: t.Background.Alpha(.92)}).TextColor(t.Text).Border(border, t.Border).
					Shadow(0, t.Space(1), t.Space(4), 0, ui.RGBA(0, 0, 0, .14)).
					Transition(motion).TransformOrigin(.5, origin)
				if options.Bottom {
					roots[i].Bottom(0)
				} else {
					roots[i].Top(0)
				}
				fade := float32(0)
				if i < n-1 && !expanded {
					fade = 1
				}
				roots[i].Children(func() {
					content := toastContent(c, parts, toast, fade, spring)
					if h := content.LayoutBounds().H; h > 0 {
						heights[i] = h + 2*padY + 2*border
					}
				})
			}
		})
		var front float32
		for i := n - 1; i >= 0 && front == 0; i-- {
			front = heights[i]
		}
		stackHeight := front + peek*float32(max(0, n-1))
		var offset float32
		for i := n - 1; i >= 0; i-- {
			depth := float32(n - 1 - i)
			pose := ui.Pose{Y: dir * offset}
			offset += heights[i] + gap
			if depth > 0 && !expanded {
				pose = ui.Pose{Y: dir * peek * depth, Scale: 1 - .05*depth}
				if front > 0 {
					roots[i].Height(front)
				}
			}
			roots[i].AnimatePose(pose, spring)
		}
		if expanded {
			stackHeight = max(0, offset-gap)
		}
		stack.Height(stackHeight)
	})
}

func toastContent(c *ui.Context, parts ui.ToastParts, toast ui.Toast, fade float32, animation ui.Animation) ui.Element {
	t := c.Theme()
	align := ui.Center
	if toast.Description != "" {
		align = ui.Start
	}
	return ui.Row(c).Shrink(0).AlignItems(align).Gap(t.Space(3)).AnimatePose(ui.Pose{Fade: fade}, animation).Children(func() {
		toastLeading(c, toast.Type)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(t.Space(.5)).Children(func() {
			ui.Text(c, toast.Title).FontWeight(500)
			if toast.Description != "" {
				ui.Text(c, toast.Description).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
		})
		if toast.Action != "" {
			toastButton(c, parts.ActionButton(), false).Shrink(0).Label(toast.Action).
				Children(func() { ui.Text(c, toast.Action).SingleLine() })
		}
		toastButton(c, parts.CloseButton(), true).Label(locales.CloseToast()).
			Size(t.Space(6), t.Space(6)).Padding(0).Shrink(0).TextColor(t.TextMuted).Children(func() {
			ui.Icon(c, icons.X()).Size(t.Rem(.875), t.Rem(.875))
		})
	})
}

func toastButton(c *ui.Context, button ui.Element, ghost bool) ui.Element {
	t := c.Theme()
	bg, hover, pressed, border := t.Background.Mix(t.Text, .08), t.Background.Mix(t.Text, .12), t.Background.Mix(t.Text, .18), t.Border
	if ghost {
		bg, hover, pressed, border = ui.Transparent, t.SurfaceHover, t.SurfacePressed, ui.Transparent
	}
	if !button.IsDisabled() {
		switch {
		case button.Pressed():
			bg = pressed
		case button.Hovered():
			bg = hover
		}
	}
	return button.Padding(t.Space(1.5), t.Space(3.5)).Radius(t.Radius).Background(bg).TextColor(t.Text).Border(1, border)
}

func toastLeading(c *ui.Context, kind string) {
	t := c.Theme()
	var icon *ui.SVG
	var color ui.Color
	switch kind {
	case ToastInfo:
		icon, color = icons.Info(), t.Accent
	case ToastSuccess:
		icon, color = icons.CircleCheck(), t.Success
	case ToastWarning:
		icon, color = icons.TriangleAlert(), t.Warning
	case ToastError:
		icon, color = icons.CircleX(), t.Danger
	default:
		return
	}
	ui.Icon(c, icon).Size(t.Rem(1.125), t.Rem(1.125)).Shrink(0).TextColor(color).Role(ui.RoleNone)
}
