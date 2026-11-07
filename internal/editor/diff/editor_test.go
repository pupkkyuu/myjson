package diff

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/format"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

func TestSearchBarStaysInActivePane(t *testing.T) {
	for _, width := range []int{440, 900} {
		for _, comparing := range []bool{false, true} {
			t.Run(fmt.Sprintf("width=%d/preview=%v", width, comparing), func(t *testing.T) {
				e := NewDiffEditor("editor-1", `{"pet":"cat"}`, `{"pet":"dog"}`)
				e.comparing, e.activeSide = comparing, 1
				tt := ui.NewTester(e.View, width, 400)
				e.Search()
				tt.Frame()
				for _, label := range []string{locales.FindQuery(), locales.FindMatchCase(), locales.FindNext(), locales.FindClose()} {
					r, ok := tt.Find(label)
					if !ok || r.X < float32(width)/2 || r.X+r.W > float32(width) {
						t.Fatalf("搜索控件 %q 超出活动侧：%+v", label, r)
					}
				}
				if comparing {
					if _, ok := tt.Find("⇄"); ok {
						t.Fatal("只读差异预览不应出现替换入口")
					}
				} else if err := tt.Click("⇄"); err != nil {
					t.Fatal(err)
				} else if r, ok := tt.Find(locales.FindReplaceAll()); !ok || r.X+r.W > float32(width) {
					t.Fatalf("替换操作未显示在活动侧内：%+v", r)
				}
			})
		}
	}
}

func mountDiff(e *DiffEditor) (*ui.Tester, <-chan struct{}) {
	wake := make(chan struct{}, 8)
	e.Invalidate = func() { wake <- struct{}{} }
	return ui.NewTester(e.View, 1000, 600), wake
}

func awaitDiff(t *testing.T, wake <-chan struct{}) {
	t.Helper()
	select {
	case <-wake:
	case <-time.After(5 * time.Second):
		t.Fatal("对比编辑器后台任务未完成")
	}
}

func TestDiffIdentityAndSharedBuffers(t *testing.T) {
	e := NewDiffEditor("editor-18446744073709551615", "{}", "[]")
	if e.ID() != "editor-18446744073709551615" || e.Title() != locales.ComparisonTabTitle("18446744073709551615") {
		t.Fatalf("对比编辑器标识丢失精度：%s，%q", e.ID(), e.Title())
	}
	if e.preview.Old != e.left.Buffer || e.preview.New != e.right.Buffer {
		t.Fatal("差异预览应复用输入区的缓冲区")
	}
	if e.left.Label != locales.LeftEditorLabel() || e.right.Label != locales.RightEditorLabel() || e.left.Placeholder != locales.LeftEditorPlaceholder() || e.right.Placeholder != locales.RightEditorPlaceholder() {
		t.Fatal("左右编辑器应使用各自的中文标签和占位提示")
	}
	if !e.TransformEnabled() || e.Loading() || e.Comparing() || e.CopyEnabled() {
		t.Fatal("新对比编辑器的操作状态不正确")
	}
}

func TestPairedTransformsCommitAndUndo(t *testing.T) {
	for _, tc := range []struct {
		name, left, right, wantLeft, wantRight string
		action                                 func(*DiffEditor)
	}{
		{"格式化", `{"b":2}`, `[1]`, "{\n  \"b\": 2\n}", "[\n  1\n]", (*DiffEditor).Format},
		{"压缩", "{ \"a\": 1 }", "[ 2, 1 ]", `{"a":1}`, `[2,1]`, (*DiffEditor).Compact},
		{"转义", "左侧", "右侧", `"左侧"`, `"右侧"`, (*DiffEditor).Escape},
		{"去转义", `"{}"`, `"[]"`, "{}", "[]", (*DiffEditor).Unescape},
		{"字段排序", `{"z":1,"a":2}`, `{"d":3,"b":4}`, "{\n  \"a\": 2,\n  \"z\": 1\n}", "{\n  \"b\": 4,\n  \"d\": 3\n}", (*DiffEditor).SortFields},
		{"跳过空侧", " \n", `{"a":1}`, " \n", "{\n  \"a\": 1\n}", (*DiffEditor).Format},
		{"跳过空白转义", "", " \t", "", " \t", (*DiffEditor).Escape},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewDiffEditor("editor-1", tc.left, tc.right)
			tt, wake := mountDiff(e)
			tc.action(e)
			if !e.Loading() || e.TransformEnabled() || e.left.Buffer.String() != tc.left || e.right.Buffer.String() != tc.right {
				t.Fatal("后台计算期间应保留两侧原文并禁用重复变换")
			}
			awaitDiff(t, wake)
			tt.Frame()
			if e.Loading() || !e.TransformEnabled() || e.left.Buffer.String() != tc.wantLeft || e.right.Buffer.String() != tc.wantRight {
				t.Fatalf("双侧提交结果不正确：%q / %q", e.left.Buffer.String(), e.right.Buffer.String())
			}
			for _, pane := range []struct {
				source, want string
				undo         func() bool
				text         func() string
			}{
				{tc.left, tc.wantLeft, e.left.Buffer.Undo, e.left.Buffer.String},
				{tc.right, tc.wantRight, e.right.Buffer.Undo, e.right.Buffer.String},
			} {
				if pane.source == pane.want {
					if pane.undo() {
						t.Fatal("未变化的文档不应产生撤销记录")
					}
				} else if !pane.undo() || pane.text() != pane.source || pane.undo() {
					t.Fatal("每侧变换应各自作为单次可撤销编辑")
				}
			}
		})
	}
}

func TestPairedFailurePreservesBothDocuments(t *testing.T) {
	for _, badSide := range []int{0, 1} {
		t.Run([]string{"左侧失败", "右侧失败"}[badSide], func(t *testing.T) {
			sources := [2]string{`{"a":1}`, `{"b":2}`}
			sources[badSide] = `{"bad":}`
			e := NewDiffEditor("editor-1", sources[0], sources[1])
			tt, wake := mountDiff(e)
			e.Format()
			awaitDiff(t, wake)
			tt.Frame()
			if e.left.Buffer.String() != sources[0] || e.right.Buffer.String() != sources[1] || e.left.Buffer.CanUndo() || e.right.Buffer.CanUndo() {
				t.Fatal("任一侧失败时，两侧都必须保留原文及撤销历史")
			}
			panes := [2]int{len(e.left.Diagnostics()), len(e.right.Diagnostics())}
			if panes[badSide] != 1 || panes[1-badSide] != 0 || e.activeSide != badSide || e.Loading() {
				t.Fatalf("应定位并聚焦出错侧：诊断 %v，活动侧 %d", panes, e.activeSide)
			}
			d := json.Validate(sources[badSide])[0]
			message := locales.JsonErrorLocation(d.Line, d.Column, d.Detail)
			message = []func(string) string{locales.DiffLeftError, locales.DiffRightError}[badSide](message)
			if !tt.HasText(message) {
				t.Fatalf("错误提示缺少出错侧：%q", message)
			}
		})
	}
}

func TestPairedTransformInvalidation(t *testing.T) {
	for _, mode := range []string{"编辑左侧", "编辑右侧", "清空", "进入预览"} {
		t.Run(mode, func(t *testing.T) {
			e := NewDiffEditor("editor-1", `{"a":1}`, `{"b":2}`)
			tt, wake := mountDiff(e)
			started, release := make(chan struct{}), make(chan struct{})
			firstCall := true
			e.transform(func(source string) (string, error) {
				if firstCall {
					firstCall = false
					close(started)
					<-release
				}
				return json.Format(source)
			})
			awaitDiff(t, started)
			switch mode {
			case "编辑左侧":
				e.left.Buffer.Set("null")
			case "编辑右侧":
				e.right.Buffer.Set("null")
			case "清空":
				e.Clear()
			case "进入预览":
				e.ToggleComparison()
			}
			wantLeft, wantRight := e.left.Buffer.String(), e.right.Buffer.String()
			close(release)
			awaitDiff(t, wake)
			tt.Frame()
			if e.Loading() || e.left.Buffer.String() != wantLeft || e.right.Buffer.String() != wantRight {
				t.Fatal("被取消或版本过期的双侧结果不应提交任何一侧")
			}
			if e.Comparing() != (mode == "进入预览") {
				t.Fatal("后台完成不应改变用户选择的模式")
			}
		})
	}
}

func TestClearAllowsNewPairedTransform(t *testing.T) {
	for _, oldFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("旧任务先完成=%v", oldFirst), func(t *testing.T) {
			e := NewDiffEditor("editor-1", "{}", "[]")
			tt, wake := mountDiff(e)
			started := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
			release := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
			for _, signal := range release {
				t.Cleanup(func() {
					select {
					case <-signal:
					default:
						close(signal)
					}
				})
			}
			for i := range started {
				called := false
				e.transform(func(string) (string, error) {
					if !called {
						called = true
						close(started[i])
						<-release[i]
					}
					return []string{`{"old":true}`, `{"new":true}`}[i], nil
				})
				awaitDiff(t, started[i])
				if i == 0 {
					e.Clear()
					if e.Loading() || !e.TransformEnabled() {
						t.Fatal("清空后应立即恢复空闲状态")
					}
					e.left.Buffer.Set("{}")
					e.right.Buffer.Set("[]")
				}
			}
			order := [2]int{0, 1}
			if !oldFirst {
				order = [2]int{1, 0}
			}
			for _, i := range order {
				close(release[i])
				awaitDiff(t, wake)
				tt.Frame()
			}
			if e.Loading() || e.left.Buffer.String() != `{"new":true}` || e.right.Buffer.String() != `{"new":true}` {
				t.Fatalf("旧任务覆盖了新结果：%q / %q", e.left.Buffer.String(), e.right.Buffer.String())
			}
		})
	}
}

func TestNonJSONWhitespaceIsNotSkipped(t *testing.T) {
	e := NewDiffEditor("editor-1", "\u00a0", "{}")
	tt, wake := mountDiff(e)
	e.Format()
	awaitDiff(t, wake)
	tt.Frame()
	if e.left.Buffer.String() != "\u00a0" || e.right.Buffer.String() != "{}" || len(e.left.Diagnostics()) != 1 {
		t.Fatal("非 JSON 空白应参与格式化并报告错误")
	}
}

func TestPaneDiagnosticsWithoutInvalidate(t *testing.T) {
	e := NewDiffEditor("editor-1", "", "")
	tt := ui.NewTester(e.View, 900, 600)
	e.left.Buffer.Set(`{"a":}`)
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	deadline := time.After(5 * time.Second)
	for len(e.left.Diagnostics()) == 0 {
		select {
		case <-deadline:
			t.Fatal("没有刷新回调时，左侧诊断未提交")
		default:
			tt.Advance(16 * time.Millisecond)
		}
	}
}

func TestLoadingAndPreviewRejectTransforms(t *testing.T) {
	e := NewDiffEditor("editor-1", "{}", "[]")
	tt, wake := mountDiff(e)
	started, release := make(chan struct{}), make(chan struct{})
	e.transform(func(source string) (string, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
		return source, nil
	})
	awaitDiff(t, started)
	duplicate := make(chan struct{}, 1)
	e.transform(func(string) (string, error) { duplicate <- struct{}{}; return "null", nil })
	close(release)
	awaitDiff(t, wake)
	tt.Frame()
	e.ToggleComparison()
	for _, action := range []func(){e.Format, e.Compact, e.Escape, e.Unescape, e.SortFields} {
		action()
	}
	if e.Loading() || e.TransformEnabled() || e.left.Buffer.String() != "{}" || e.right.Buffer.String() != "[]" {
		t.Fatal("差异预览中不应启动文档变换")
	}
	select {
	case <-duplicate:
		t.Fatal("忙碌时不应启动重复变换")
	default:
	}
}

func TestDifferenceNavigationWrapsAndTracksEdits(t *testing.T) {
	left := "a\n" + strings.Repeat("same\n", 20) + "b\n"
	right := "A\n" + strings.Repeat("same\n", 20) + "B\n"
	e := NewDiffEditor("editor-1", left, right)
	tt, _ := mountDiff(e)
	e.NextDifference()
	if e.currentHunk != -1 {
		t.Fatal("输入模式不应导航差异")
	}
	e.ToggleComparison()
	tt.Frame()
	count := e.DifferenceCount()
	if count < 2 {
		t.Fatalf("测试文档应包含多个独立差异，得到 %d", count)
	}
	e.PreviousDifference()
	if e.currentHunk != count-1 {
		t.Fatal("首次向前导航应选中最后一个差异")
	}
	e.NextDifference()
	if e.currentHunk != 0 {
		t.Fatal("末尾差异应循环到首个差异")
	}
	e.NextDifference()
	if e.currentHunk != 1 {
		t.Fatal("未选中下一个差异")
	}
	e.right.Buffer.Set(left)
	tt.Frame()
	e.NextDifference()
	if e.DifferenceCount() != 0 || e.currentHunk != -1 {
		t.Fatal("编辑后应重新计算差异并重置导航位置")
	}
}

func TestDiffViewActionsAndClear(t *testing.T) {
	e := NewDiffEditor("editor-1", "{\n  \"a\": 1\n}", "{\n  \"b\": 2\n}")
	tt, _ := mountDiff(e)
	e.activeSide = 1
	e.Focus()
	tt.Frame()
	if !e.right.Focused() {
		t.Fatal("聚焦请求应发送到当前活动侧")
	}
	e.ToggleWrap()
	if !e.WrapEnabled() || !e.left.SoftWrap || !e.right.SoftWrap || !e.preview.Wrap {
		t.Fatal("换行设置必须同步到双侧输入与预览")
	}
	e.FoldAll()
	tt.Frame()
	if len(e.left.Folded()) == 0 || len(e.right.Folded()) == 0 {
		t.Fatal("输入模式折叠应作用于两侧")
	}
	e.UnfoldAll()
	tt.Frame()
	if len(e.left.Folded()) != 0 || len(e.right.Folded()) != 0 {
		t.Fatal("输入模式展开应作用于两侧")
	}
	e.ToggleComparison()
	tt.Frame()
	if _, ok := tt.Find(locales.DiffPreviewLabel()); !ok {
		t.Fatal("预览模式未渲染差异视图")
	}
	e.FoldAll()
	e.UnfoldAll()
	e.ToggleWrap()
	if e.WrapEnabled() || e.right.SoftWrap || e.preview.Wrap || e.left.Buffer.CanUndo() || e.right.Buffer.CanUndo() {
		t.Fatal("视图操作不应修改内容或新增撤销步骤")
	}
	left, right := e.left.Buffer.String(), e.right.Buffer.String()
	e.Clear()
	tt.Frame()
	if e.Comparing() || e.activeSide != 0 || e.left.Buffer.Len() != 0 || e.right.Buffer.Len() != 0 {
		t.Fatal("清空应清除两侧并返回左侧输入")
	}
	if !e.left.Buffer.Undo() || !e.right.Buffer.Undo() || e.left.Buffer.String() != left || e.right.Buffer.String() != right {
		t.Fatal("双侧清空应各自可撤销")
	}
}

func TestComparisonHidesInputsAndKeepsTheirFindState(t *testing.T) {
	e := NewDiffEditor("editor-1", `{"left":"cat"}`, `{"right":"cat cat"}`)
	tt, _ := mountDiff(e)
	e.activeSide = 1
	e.Search()
	tt.Frame()
	tt.Type("cat")
	tt.Frame()
	if count, _ := e.right.FindMatches(); count != 2 {
		t.Fatalf("右侧搜索应找到两个结果，得到 %d", count)
	}

	e.ToggleComparison()
	tt.Frame()
	if _, ok := tt.Find(locales.RightEditorLabel()); ok {
		t.Fatal("预览期间输入区不应参与布局或交互")
	}
	if _, ok := tt.Find(locales.FindQuery()); ok {
		t.Fatal("隐藏输入区的查找栏不应显示")
	}
	tt.Type("x")
	if e.right.Buffer.String() != `{"right":"cat cat"}` {
		t.Fatal("预览期间键盘输入不应修改隐藏的编辑器")
	}

	e.ToggleComparison()
	tt.Frame()
	if _, ok := tt.Find(locales.FindQuery()); !ok {
		t.Fatal("返回输入模式应保留右侧查找栏")
	}
	if count, _ := e.right.FindMatches(); count != 2 {
		t.Fatalf("返回输入模式应保留查询和匹配结果，得到 %d", count)
	}
	if !e.right.Focused() || e.activeSide != 1 {
		t.Fatal("返回输入模式应聚焦原活动侧")
	}
}

func TestPairedTransformCommitsDecodedTextDiagnostics(t *testing.T) {
	e := NewDiffEditor("editor-1", `"普通文本"`, `"{}"`)
	tt, wake := mountDiff(e)
	e.Unescape()
	awaitDiff(t, wake)
	tt.Frame()
	if e.left.Buffer.String() != "普通文本" || len(e.left.Diagnostics()) != 1 || e.right.Buffer.String() != "{}" || len(e.right.Diagnostics()) != 0 {
		t.Fatal("去转义应允许普通文本，并在同一次提交中同步 JSON 诊断")
	}
}

func TestUnescapeNormalizesLineEndingsBeforeDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, decoded, want string
	}{
		{"混合换行和 Unicode", "{\r\n\n\r\n\"中文🙂\":?}", "{\n\n\n\"中文🙂\":?}"},
		{"保留字面转义", `{"a":"\r\n"}`, `{"a":"\r\n"}`},
		{"保留独立回车", "{\r\"a\":?}", "{\r\"a\":?}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := json.Escape(tc.decoded)
			if err != nil {
				t.Fatal(err)
			}
			e := NewDiffEditor("editor-1", source, source)
			tt, wake := mountDiff(e)
			e.Unescape()
			awaitDiff(t, wake)
			tt.Frame()
			for side, pane := range []*format.FormatEditor{e.left, e.right} {
				if got := pane.Buffer.String(); got != tc.want {
					t.Fatalf("第 %d 侧去转义结果 = %q，期望 %q", side, got, tc.want)
				}
				diagnostics := pane.Diagnostics()
				if start := strings.IndexByte(tc.want, '?'); start >= 0 {
					if len(diagnostics) != 1 || diagnostics[0].Start != start || diagnostics[0].End != start+1 {
						t.Fatalf("第 %d 侧诊断 = %+v，应覆盖 ? 的字节区间 [%d,%d)", side, diagnostics, start, start+1)
					}
				} else if len(diagnostics) != 0 {
					t.Fatalf("第 %d 侧有效 JSON 不应产生诊断：%+v", side, diagnostics)
				}
				if tc.decoded != tc.want {
					if !pane.Buffer.Undo() || pane.Buffer.String() != source || pane.Buffer.CanUndo() ||
						!pane.Buffer.Redo() || pane.Buffer.String() != tc.want {
						t.Fatal("规范化后的文本应可作为一次编辑撤销并重做")
					}
				}
			}
		})
	}
}

func TestDiagnosticsContinueInPreview(t *testing.T) {
	e := NewDiffEditor("editor-1", "", "")
	tt, wake := mountDiff(e)
	e.ToggleComparison()
	tt.Frame()
	e.left.Buffer.Set("{")
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	awaitDiff(t, wake)
	tt.Frame()
	if !e.Comparing() || len(e.left.Diagnostics()) != 1 || len(e.right.Diagnostics()) != 0 {
		t.Fatal("隐藏输入区时仍应处理所属文档的诊断结果")
	}
	e.left.Buffer.Set("{}")
	tt.Frame()
	if len(e.left.Diagnostics()) != 0 {
		t.Fatal("预览模式中内容变化也应立即清除旧诊断")
	}
}
