package format

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
)

func TestPreferencesPreserveDocumentAndWrap(t *testing.T) {
	p := editor.DefaultPreferences()
	p.DefaultWrap = true
	e := NewFormatEditor("editor-1", "{}", p)
	tt, _ := mountEditor(e)
	e.Select(1, 1)
	e.ToggleWrap()
	p.FontSize, p.LineHeight, p.TabSize = 18, 1.8, 4
	p.Whitespace, p.ActiveLine, p.IndentGuides, p.AutoClose = mygoeditor.WhitespaceAll, true, false, false
	e.ApplyPreferences(p)
	tt.Frame()
	if e.Buffer.String() != "{}" || e.Buffer.CanUndo() || e.Cursor() != 1 || e.SoftWrap {
		t.Fatal("设置变更改写了文档、光标或当前换行选择")
	}
	if e.Font.Size != 18 || e.LineHeight != 1.8 || e.NoActiveLine || !e.NoIndentGuides || !e.NoAutoClose || e.Whitespace != mygoeditor.WhitespaceAll {
		t.Fatal("已打开编辑器未应用外观和输入设置")
	}
	input := NewFormatEditor("editor-2", "", p)
	input.Focus()
	inputTester, _ := mountEditor(input)
	inputTester.Key(0, ui.KeyTab)
	if input.Buffer.String() != "    " || !input.SoftWrap {
		t.Fatalf("新文档未采用缩进或默认换行：%q / %v", input.Buffer.String(), input.SoftWrap)
	}
}

func TestTransformsCaptureIndentPreference(t *testing.T) {
	for _, action := range []struct {
		name string
		run  func(*FormatEditor)
	}{{"format", (*FormatEditor).Format}, {"sort", (*FormatEditor).SortFields}} {
		t.Run(action.name, func(t *testing.T) {
			p := editor.DefaultPreferences()
			p.TabSize = 4
			source := `{"b":9007199254740993,"a":{"x":1}}`
			e := NewFormatEditor("editor-1", source, p)
			tt, wake := mountEditor(e)
			action.run(e)
			p.TabSize = 8
			e.ApplyPreferences(p)
			waitForEditor(t, wake)
			tt.Frame()
			got := e.Buffer.String()
			if !strings.Contains(got, "\n    \"") || !strings.Contains(got, "9007199254740993") {
				t.Fatalf("后台操作未保留调用时缩进或数字原文：%s", got)
			}
		})
	}
}
