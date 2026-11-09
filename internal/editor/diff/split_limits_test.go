package diff

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
)

func splitPosition(t *testing.T, tt *ui.Tester) float32 {
	t.Helper()
	divider, ok := tt.AccessByRole(ui.RoleSplitter, "")
	if !ok {
		t.Fatal("未找到对比输入区分隔线")
	}
	return divider.Bounds.X + divider.Bounds.W/2
}

func dragSplit(t *testing.T, tt *ui.Tester, to float32) {
	t.Helper()
	x := splitPosition(t, tt)
	tt.Press(x, 200)
	tt.Move(to, 200)
	tt.Release(to, 200)
}

func TestInputSplitMinimumWidth(t *testing.T) {
	e := NewDiffEditor("editor-1", "{}", "[]")
	tt := ui.NewTester(e.View, 1000, 400)
	dragSplit(t, tt, 0)
	left := splitPosition(t, tt)
	if left < 90 || left > 250 {
		t.Fatalf("左侧最小宽度异常：%g", left)
	}
	for range 20 {
		tt.Key(0, ui.KeyLeft)
	}
	if got := splitPosition(t, tt); math.Abs(float64(got-left)) > 1 {
		t.Fatalf("方向键使左侧越过最小宽度：%g -> %g", left, got)
	}
	dragSplit(t, tt, 999)
	right := 1000 - splitPosition(t, tt)
	if right < 90 || math.Abs(float64(right-left)) > 2 {
		t.Fatalf("右侧最小宽度 %g 与左侧 %g 不一致", right, left)
	}
	for range 20 {
		tt.Key(0, ui.KeyRight)
	}
	if got := 1000 - splitPosition(t, tt); math.Abs(float64(got-right)) > 1 {
		t.Fatalf("方向键使右侧越过最小宽度：%g -> %g", right, got)
	}
}

func TestInputSplitMinimumAdaptsToFontAndGutter(t *testing.T) {
	e := NewDiffEditor("editor-1", "{}", "[]")
	tt := ui.NewTester(e.View, 1000, 400)
	dragSplit(t, tt, 0)
	initial := splitPosition(t, tt)
	p := editor.DefaultPreferences()
	p.FontSize = 28
	e.ApplyPreferences(p)
	tt.Frame()
	dragSplit(t, tt, 0)
	largeFont := splitPosition(t, tt)
	if largeFont <= initial+30 {
		t.Fatalf("字号增大后最小宽度未增大：%g -> %g", initial, largeFont)
	}
	e.left.Buffer.Set(strings.Repeat("{}\n", 10000))
	tt.Frame()
	dragSplit(t, tt, 0)
	longGutter := splitPosition(t, tt)
	if longGutter <= largeFont {
		t.Fatalf("行号位数增多后未为编辑区保留足够宽度：%g -> %g", largeFont, longGutter)
	}
}

func TestInputSplitRestoresAfterNarrowWindow(t *testing.T) {
	e := NewDiffEditor("editor-1", "{}", "[]")
	tt := ui.NewTester(e.View, 1000, 400)
	dragSplit(t, tt, 700)
	share := e.inputSplit
	if share < 69 || share > 71 {
		t.Fatalf("调整分栏失败：%g", share)
	}
	tt.SetSize(100, 400)
	tt.Frame()
	if got := splitPosition(t, tt); math.Abs(float64(got-50)) > 1 {
		t.Fatalf("空间不足时应均分：%g", got)
	}
	dragSplit(t, tt, 0)
	tt.Key(0, ui.KeyLeft)
	if math.Abs(float64(e.inputSplit-share)) > .01 {
		t.Fatalf("窄窗口调整不应覆盖原比例：%g -> %g", share, e.inputSplit)
	}
	tt.SetSize(1000, 400)
	tt.Frame()
	if got := splitPosition(t, tt); math.Abs(float64(got-700)) > 2 {
		t.Fatalf("窗口恢复后未恢复比例：%g", got)
	}
}

// Compare the actual accessible divider limits, so the input editor follows
// the upstream diff viewer as font metrics and gutter sizes change.
func TestInputSplitMinimumMatchesDiffPreview(t *testing.T) {
	for _, tc := range []struct {
		name  string
		font  float32
		lines int
	}{
		{"default", 13, 1},
		{"large font", 28, 1},
		{"wide gutter", 20, 10000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := editor.DefaultPreferences()
			p.FontSize = tc.font
			e := NewDiffEditor("editor-1", strings.Repeat("{}\n", tc.lines), "[]", p)
			input := ui.NewTester(e.View, 1000, 400)
			preview := ui.NewTester(func(c *ui.Context) { mygoeditor.DiffView(c, &e.preview).Fill() }, 1000, 400)
			input.Frame()
			preview.Frame()
			inDivider, ok := input.AccessByRole(ui.RoleSplitter, "")
			if !ok {
				t.Fatal("输入区缺少分隔线")
			}
			outDivider, ok := preview.AccessByRole(ui.RoleSplitter, "")
			if !ok {
				t.Fatal("差异预览缺少分隔线")
			}
			if math.Abs(inDivider.Min-outDivider.Min) > 2 || math.Abs(inDivider.Max-outDivider.Max) > 2 {
				t.Fatalf("输入区范围 [%g, %g] 与预览范围 [%g, %g] 不一致", inDivider.Min, inDivider.Max, outDivider.Min, outDivider.Max)
			}
		})
	}
}

func TestInputSplitFirstLayoutKeepsShare(t *testing.T) {
	e := NewDiffEditor("editor-1", "{}", "[]")
	e.inputSplit = 70
	tt := ui.NewTester(e.View, 1000, 400)
	tt.Frame()
	if got := splitPosition(t, tt); math.Abs(float64(got-700)) > 2 {
		t.Fatalf("初次布局应显示保存的会话比例：%g", got)
	}
}
