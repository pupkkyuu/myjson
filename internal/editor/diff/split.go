package diff

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

// inputSplitMin mirrors the diff viewer's gutter, twelve text columns and
// scrollbar allowance. The viewer's actual layout metrics are private.
func (e *DiffEditor) inputSplitMin(c *ui.Context) float32 {
	font := e.preview.Font
	if font.Family == "" {
		font.Family = "monospace"
	}
	if font.Size <= 0 {
		font.Size = 13
	}
	charWidth, _ := c.MeasureText(0, ui.Span{Text: "0", Font: font.Family, Size: font.Size, Weight: font.Weight, Italic: font.Italic, Features: font.Features, Lang: font.Lang})
	digits := max(3, len(strconv.Itoa(max(e.left.Buffer.LineCount(), e.right.Buffer.LineCount()))))
	gutter := float32(math.Ceil(float64((float32(digits)+1.5)*charWidth+4))) + 6
	return max(c.Theme().Space(10), gutter+4+12*charWidth+10+4)
}

// inputSplitView keeps the same minimum as the diff preview. A narrow window
// temporarily shows equal halves, without losing the user's chosen share.
func (e *DiffEditor) inputSplitView(c *ui.Context, first, second func()) ui.Element {
	t := c.Theme()
	root := ui.Row(c).Fill().MinWidth(0).AlignItems(ui.Stretch)
	room := max(0, root.LayoutBounds().W-1)
	minSide := e.inputSplitMin(c)
	fits := room >= 2*minSide && room > 0
	shown := room / 2
	if fits {
		shown = max(minSide, min(room-minSide, room*e.inputSplit/100))
	}
	set := func(value float32) {
		if !fits {
			return
		}
		value = max(minSide, min(room-minSide, value))
		if value != shown {
			shown = value
			e.inputSplit = 100 * value / room
			c.Invalidate()
		}
	}
	root.Children(func() {
		share := float32(.5)
		if room > 0 {
			share = shown / room
		}
		ui.Box(c.Key("left-pane")).Grow(share).Basis(0).MinWidth(0).Clip().Children(first)
		divider := ui.Box(c.Key("divider")).Width(1).Shrink(0).FillHeight().Focusable().FocusRing(false).
			Role(ui.RoleSplitter).Label(locales.In(c.Locale()).InputDivider())
		if divider.Shortcut(0, ui.KeyLeft) {
			set(shown - t.Space(2.5))
		}
		if divider.Shortcut(0, ui.KeyRight) {
			set(shown + t.Space(2.5))
		}
		low, high := minSide, room-minSide
		if !fits {
			low, high = shown, shown
		}
		divider.Range(float64(low), float64(high), float64(shown)).Step(float64(t.Space(2.5)))
		ui.Box(c.Key("right-pane")).Grow(1 - share).Basis(0).MinWidth(0).Clip().Children(second)
		grip := t.Space(1.5)
		handle := ui.Box(c.Key("divider-handle")).Absolute().Left(shown - (grip-1)/2).Top(0).Bottom(0).
			Width(grip).Cursor(ui.CursorResizeEW)
		dx, _, held := handle.Dragged()
		if held {
			divider.Focus()
			if dx != 0 {
				set(shown + dx)
			}
		}
		if handle.DoubleClicked() && fits {
			divider.Focus()
			e.inputSplit = 50
			c.Invalidate()
		}
		line := t.Border
		if held || handle.Hovered() {
			line = t.Accent
		}
		divider.Background(line)
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if divider.FocusVisible() {
				p.FocusRing(r, [4]float32{})
			}
		})
	})
	return root
}
