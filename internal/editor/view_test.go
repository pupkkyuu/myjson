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
			var cache editor.ThemeCache
			theme := cache.Resolve(palette)
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

func TestThemeCacheFollowsColors(t *testing.T) {
	var first, second editor.ThemeCache
	palette := *ui.LightTheme()
	light := first.Resolve(&palette)
	if light.Background != palette.Surface {
		t.Fatal("编辑器背景应使用 UI 主题的 Surface")
	}
	frameTheme := palette
	frameTheme.FontSize *= 1.5
	if first.Resolve(&frameTheme) != light {
		t.Fatal("相同深浅色和背景色应复用主题，不依赖 UI 主题指针或字号")
	}
	other := second.Resolve(&palette)
	palette.Surface = ui.Hex("#123456")
	changed := first.Resolve(&palette)
	if changed == light || changed.Background != palette.Surface || changed.Text != light.Text || changed.Tokens != light.Tokens {
		t.Fatal("同一深浅色下切换背景应更新背景并保留语法颜色")
	}
	if light.Background != frameTheme.Surface || other.Background != frameTheme.Surface {
		t.Fatal("更新缓存不应修改先前主题或其他编辑器的主题")
	}
	palette.Dark = true
	dark := first.Resolve(&palette)
	if dark == changed || dark.Text != ui.Hex("#DCDCDC") || dark.Background != palette.Surface {
		t.Fatal("相同背景下切换深浅色也应更新语法颜色")
	}
	palette.Dark = false
	restored := first.Resolve(&palette)
	if restored.Text != light.Text || restored.Tokens != light.Tokens || restored.Background != palette.Surface {
		t.Fatal("切回浅色应恢复语法颜色")
	}
}

func TestThemeCacheHasNoSteadyStateAllocations(t *testing.T) {
	var cache editor.ThemeCache
	palette := *ui.LightTheme()
	benchmarkTheme = cache.Resolve(&palette)
	if allocations := testing.AllocsPerRun(100, func() {
		frameTheme := palette
		benchmarkTheme = cache.Resolve(&frameTheme)
	}); allocations != 0 {
		t.Fatalf("颜色不变时不应分配编辑器主题，得到 %g 次分配", allocations)
	}
}
