package diff

import (
	"strings"
	"testing"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
)

func TestDiffPreferencesUpdateInputsAndPreview(t *testing.T) {
	p := editor.DefaultPreferences()
	p.DefaultWrap = true
	p.DiffLayout = mygoeditor.DiffUnified
	e := NewDiffEditor("editor-1", "{}", "[]", p)
	tt, _ := mountDiff(e)
	if !e.left.SoftWrap || !e.right.SoftWrap || !e.preview.Wrap || e.preview.Layout != mygoeditor.DiffUnified {
		t.Fatal("新对比文档未继承默认换行和布局")
	}
	e.ToggleWrap()
	e.ToggleComparison()
	p.FontSize, p.LineHeight, p.TabSize = 20, 1.8, 4
	p.DiffInline = mygoeditor.DiffInlineNone
	e.ApplyPreferences(p)
	tt.Frame()
	if e.left.Font.Size != 20 || e.right.Font.Size != 20 || e.preview.Font.Size != 20 ||
		e.left.TabSize != 4 || e.right.TabSize != 4 || e.preview.TabSize != 4 ||
		e.preview.LineHeight != 1.8 || e.preview.Inline != mygoeditor.DiffInlineNone {
		t.Fatal("对比输入和预览未同步设置")
	}
	if e.WrapEnabled() || e.right.SoftWrap || e.preview.Wrap || e.left.Buffer.CanUndo() || e.right.Buffer.CanUndo() {
		t.Fatal("更新对比设置重置了当前换行状态或产生了文本编辑")
	}
}

func TestPairedTransformsUseIndentPreference(t *testing.T) {
	for _, action := range []struct {
		name string
		run  func(*DiffEditor)
	}{{"format", (*DiffEditor).Format}, {"sort", (*DiffEditor).SortFields}} {
		t.Run(action.name, func(t *testing.T) {
			p := editor.DefaultPreferences()
			p.TabSize = 4
			source := `{"b":2,"a":1}`
			e := NewDiffEditor("editor-1", source, source, p)
			tt, wake := mountDiff(e)
			action.run(e)
			awaitDiff(t, wake)
			tt.Frame()
			for _, text := range []string{e.left.Buffer.String(), e.right.Buffer.String()} {
				if !strings.Contains(text, "\n    \"") {
					t.Fatal("双侧格式化或排序未采用缩进配置")
				}
			}
		})
	}
}
