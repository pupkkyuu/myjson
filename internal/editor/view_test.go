package editor_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

func TestSyntaxThemeColors(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		dark                                 bool
		text, property, str, number, comment string
	}{
		{"浅色", false, "#000000", "#A31515", "#0451A5", "#098658", "#008000"},
		{"深色", true, "#DCDCDC", "#9CDCFE", "#CE9178", "#B5CEA8", "#608B4E"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			palette := ui.LightTheme()
			if tc.dark {
				palette = ui.DarkTheme()
			}
			theme := editor.Theme(palette)
			if theme.Text != ui.Hex(tc.text) {
				t.Fatal("默认文本颜色不符合主题")
			}
			for kind, want := range map[mygoeditor.TokenKind]string{
				mygoeditor.TokenText: tc.text, mygoeditor.TokenProperty: tc.property,
				mygoeditor.TokenString: tc.str, mygoeditor.TokenEscape: tc.str,
				mygoeditor.TokenConstant: tc.str, mygoeditor.TokenKeyword: tc.str,
				mygoeditor.TokenNumber: tc.number, mygoeditor.TokenComment: tc.comment,
				mygoeditor.TokenOperator: tc.text, mygoeditor.TokenPunctuation: tc.text,
			} {
				if theme.Tokens[kind].Color != ui.Hex(want) {
					t.Errorf("词法类型 %v 的颜色 = %v，期望 %s", kind, theme.Tokens[kind].Color, want)
				}
			}
		})
	}
}

func TestDiagnosticConversion(t *testing.T) {
	if editor.Diagnostics(nil) != nil || editor.Diagnostics([]json.Diagnostic{}) != nil {
		t.Fatal("空诊断应返回 nil")
	}
	source := []json.Diagnostic{
		{Start: 3, End: 6, Line: 2, Column: 1, Detail: "非法字符"},
		{Start: 9, End: 10, Line: 3, Column: 4, Detail: "缺少值"},
	}
	got := editor.Diagnostics(source)
	if len(got) != len(source) {
		t.Fatalf("诊断数量 = %d，期望 %d", len(got), len(source))
	}
	for i, d := range source {
		if got[i].Start != d.Start || got[i].End != d.End || got[i].Severity != mygoeditor.SeverityError || got[i].Source != "JSON" || got[i].Message != locales.JsonErrorLocation(d.Line, d.Column, d.Detail) {
			t.Errorf("诊断转换丢失信息：%+v", got[i])
		}
	}
	got[0].Start = 100
	if source[0].Start != 3 {
		t.Fatal("转换结果不应与输入共享可变数据")
	}
}
