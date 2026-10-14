package editor_test

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/locales"
)

func TestFindBarFitsPaneAndKeepsRows(t *testing.T) {
	for _, width := range []int{220, 900} {
		for _, replacing := range []bool{false, true} {
			t.Run(fmt.Sprintf("width=%d/replace=%v", width, replacing), func(t *testing.T) {
				width := width
				st := &mygoeditor.State{Buffer: mygoeditor.NewBuffer("cat dog cat")}
				st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *mygoeditor.FindBar) {
					editor.RenderFindBar(c, bar, f)
					bar.Label("find-toolbar")
				}
				st.Find("cat")
				if replacing {
					st.Run(mygoeditor.FindReplace)
				}
				tt := ui.NewTester(func(c *ui.Context) { mygoeditor.View(c, st).Fill() }, width, 400)
				bounds := func(label string) ui.Rect {
					t.Helper()
					r, ok := tt.Find(label)
					if !ok || r.W <= 0 || r.H <= 0 {
						t.Fatalf("控件 %q 不可见：%+v", label, r)
					}
					if r.X < 0 || r.X+r.W > float32(width)+0.5 {
						t.Fatalf("控件 %q 超出分栏：%+v，宽度 %d", label, r, width)
					}
					return r
				}
				bar := bounds("find-toolbar")
				query := bounds(locales.FindQuery())
				options := bounds(locales.FindMatchCase())
				if options.Y < query.Y+query.H {
					t.Fatal("搜索选项应独占输入框下方的一行")
				}
				for _, label := range []string{locales.FindWholeWord(), locales.FindRegexp(), locales.FindInSelection(), locales.FindPrevious(), locales.FindNext(), "⇄", locales.FindClose()} {
					r := bounds(label)
					if delta := (r.Y + r.H/2) - (options.Y + options.H/2); delta < -0.5 || delta > 0.5 {
						t.Fatalf("控件 %q 未与选项同排：%+v", label, r)
					}
				}
				if options.Y+options.H > bar.Y+bar.H {
					t.Fatal("选项超出搜索栏")
				}
				if replacing {
					replacement := bounds(locales.FindReplace())
					if replacement.Y < query.Y+query.H || replacement.Y+replacement.H > options.Y {
						t.Fatal("替换输入框应位于查询与选项之间")
					}
					bounds(locales.FindReplaceAll())
				}
				tt.SetSize(width+50, 400)
				width += 50
				bounds("find-toolbar")
				bounds(locales.FindClose())
			})
		}
	}
}

func TestFindBarKeepsNativeInputAndReplacement(t *testing.T) {
	st := &mygoeditor.State{Buffer: mygoeditor.NewBuffer("cat dog cat"), RenderFindBar: editor.RenderFindBar}
	tt := ui.NewTester(func(c *ui.Context) { mygoeditor.View(c, st).Fill() }, 280, 400)
	st.Find("cat")
	st.Run(mygoeditor.Find)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if count, current := st.FindMatches(); count != 2 || current != 1 {
		t.Fatalf("Enter 未跳到下一匹配：%d/%d", current, count)
	}
	tt.Key(ui.Shift, ui.KeyEnter)
	if _, current := st.FindMatches(); current != 0 {
		t.Fatal("Shift+Enter 未跳到上一匹配")
	}
	if err := tt.Click("⇄"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(locales.FindReplace()); err != nil {
		t.Fatal(err)
	}
	tt.Type("fox")
	tt.Key(0, ui.KeyEnter)
	if st.Buffer.String() != "fox dog cat" {
		t.Fatalf("替换框 Enter 未替换当前匹配：%q", st.Buffer.String())
	}
	if !st.Buffer.Undo() {
		t.Fatal("单次替换应可撤销")
	}
	tt.Frame()
	if err := tt.Click(locales.FindReplaceAll()); err != nil {
		t.Fatal(err)
	}
	if st.Buffer.String() != "fox dog fox" {
		t.Fatalf("原生全部替换未生效：%q", st.Buffer.String())
	}
	if !st.Buffer.Undo() || st.Buffer.String() != "cat dog cat" {
		t.Fatal("全部替换应保留原生单次撤销")
	}
	if err := tt.Click(locales.FindReplace()); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.Find(locales.FindClose()); ok || !st.Focused() {
		t.Fatal("Escape 应关闭搜索栏并恢复编辑器焦点")
	}
}
