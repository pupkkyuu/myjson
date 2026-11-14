package format

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

func mountEditor(e *FormatEditor) (*ui.Tester, <-chan struct{}) {
	wake := make(chan struct{}, 8)
	e.Invalidate = func() { wake <- struct{}{} }
	return ui.NewTester(e.View, 900, 600), wake
}

func waitForEditor(t *testing.T, wake <-chan struct{}) {
	t.Helper()
	select {
	case <-wake:
	case <-time.After(5 * time.Second):
		t.Fatal("后台任务未在时限内完成")
	}
}

func TestIdentityAndInitialState(t *testing.T) {
	for _, id := range []string{"editor-0", "editor-18446744073709551615"} {
		e := NewFormatEditor(id, `{"a":1}`)
		if e.ID() != id || e.Title() != locales.FormatTabTitle(strings.TrimPrefix(id, "editor-")) || e.Label != locales.EditorLabel() || e.Placeholder != locales.EditorPlaceholder() {
			t.Fatalf("编辑器标识或文案不正确：%s，%q", e.ID(), e.Title())
		}
		if e.Buffer.String() != `{"a":1}` || e.Loading() || !e.CopyEnabled() || e.deadline.IsZero() {
			t.Fatal("初始文档或操作状态不正确")
		}
	}
	if e := NewFormatEditor("editor-1", ""); !e.deadline.IsZero() || e.Buffer.CanUndo() {
		t.Fatal("空文档不应调度诊断或产生撤销记录")
	}
}

func TestDocumentActionsCommitAndUndo(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		action             func(*FormatEditor)
	}{
		{"格式化", `{"b":2,"a":1}`, "{\n  \"b\": 2,\n  \"a\": 1\n}", (*FormatEditor).Format},
		{"压缩", "{\n  \"a\": 1\n}", `{"a":1}`, (*FormatEditor).Compact},
		{"转义", `{"a":1}`, `"{\"a\":1}"`, (*FormatEditor).Escape},
		{"去转义", `"{\"a\":1}"`, `{"a":1}`, (*FormatEditor).Unescape},
		{"字段排序", `{"b":2,"a":1}`, "{\n  \"a\": 1,\n  \"b\": 2\n}", (*FormatEditor).SortFields},
		{"空文本转义", "", `""`, (*FormatEditor).Escape},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewFormatEditor("editor-1", tc.source)
			tt, wake := mountEditor(e)
			e.Select(0, min(2, e.Buffer.Len()))
			tc.action(e)
			if !e.Loading() || e.Buffer.String() != tc.source {
				t.Fatal("后台任务开始时应标记忙碌且保留原文")
			}
			waitForEditor(t, wake)
			tt.Frame()
			if e.Loading() || e.Buffer.String() != tc.want || !e.Focused() {
				t.Fatalf("提交后的文本或状态不正确：%q，忙碌 %v，聚焦 %v", e.Buffer.String(), e.Loading(), e.Focused())
			}
			selection := e.Selections()[0]
			if selection.Anchor != 0 || selection.Head != 0 {
				t.Fatalf("提交后光标未归零：%+v", selection)
			}
			if !e.Buffer.Undo() || e.Buffer.String() != tc.source || e.Buffer.CanUndo() {
				t.Fatal("变换应作为单次可撤销编辑提交")
			}
			if !e.Buffer.Redo() || e.Buffer.String() != tc.want {
				t.Fatal("重做未恢复变换结果")
			}
		})
	}
}

func TestTransformFailureAndNoChangePreserveHistory(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		action       func(*FormatEditor)
		failure      bool
	}{
		{"非法格式化", `{"a":}`, (*FormatEditor).Format, true},
		{"非法去转义", `\q`, (*FormatEditor).Unescape, true},
		{"格式不变", "{}", (*FormatEditor).Format, false},
		{"非字符串去转义", "{}", (*FormatEditor).Unescape, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewFormatEditor("editor-1", tc.source)
			tt, wake := mountEditor(e)
			version := e.Buffer.Version()
			tc.action(e)
			waitForEditor(t, wake)
			tt.Frame()
			if e.Loading() || e.Buffer.String() != tc.source || e.Buffer.CanUndo() || e.Buffer.Version() != version {
				t.Fatal("失败或无变化的变换不应改写文档或新增撤销记录")
			}
			if tc.failure && !strings.Contains(strings.Join(tt.Texts(), "\n"), "JSON 格式错误") {
				t.Fatal("变换失败后应渲染错误提示")
			}
		})
	}
}

func TestTransformsPreservePositionUnlessTextChanges(t *testing.T) {
	compact := "[" + strings.Repeat(`{"a":1},`, 1000) + "null]"
	formatted, err := json.Format(compact)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, source, want string
		run                func(string) (string, error)
	}{
		{"格式化不变", formatted, formatted, json.Format},
		{"压缩不变", compact, compact, json.Compact},
		{"排序不变", formatted, formatted, json.SortFields},
		{"去转义不变", formatted, formatted, json.Unescape},
		{"仅换行不同", formatted, formatted, func(source string) (string, error) {
			return strings.ReplaceAll(source, "\n", "\r\n"), nil
		}},
		{"压缩有变化", formatted, compact, json.Compact},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewFormatEditor("editor-1", tc.source)
			e.SoftWrap = true
			tt, wake := mountEditor(e)
			e.Focus()
			e.Select(e.Buffer.Len()/3, e.Buffer.Len()/3+3)
			e.SetScrollY(200)
			tt.Frame()
			version := e.Buffer.Version()

			started, release := make(chan struct{}), make(chan struct{})
			e.transform(func(source string) (string, error) {
				close(started)
				<-release
				return tc.run(source)
			})
			waitForEditor(t, started)
			// 后台运行期间的位置变化也必须保留，不能恢复任务启动时的位置。
			e.SetSelections(
				mygoeditor.Selection{Anchor: e.Buffer.Len() / 2, Head: e.Buffer.Len()/2 + 3},
				mygoeditor.Selection{Anchor: e.Buffer.Len()*2/3 + 3, Head: e.Buffer.Len() * 2 / 3},
			)
			e.SetScrollY(400)
			tt.Frame()
			selections, scroll := e.Selections(), e.ScrollY()
			close(release)
			waitForEditor(t, wake)
			if scroll <= 0 {
				t.Fatal("测试前应已滚离顶部")
			}
			tt.Frame()
			if e.Loading() || e.Buffer.String() != tc.want || !e.Focused() {
				t.Fatal("变换完成后的文本、忙碌状态或焦点不正确")
			}
			if tc.source == tc.want {
				if !reflect.DeepEqual(e.Selections(), selections) || e.ScrollY() != scroll {
					t.Fatalf("无变化变换重置了位置：选区 %v -> %v，滚动 %g -> %g", selections, e.Selections(), scroll, e.ScrollY())
				}
				if e.Buffer.Version() != version || e.Buffer.CanUndo() {
					t.Fatal("无变化变换修改了文档版本或撤销历史")
				}
			} else {
				if e.Cursor() != 0 || e.ScrollY() != 0 {
					t.Fatal("有变化的变换应按原有行为回到顶部")
				}
				if !e.Buffer.Undo() || e.Buffer.String() != tc.source {
					t.Fatal("有变化的变换应保留撤销能力")
				}
			}
		})
	}
}

func TestTransformErrorToastMatchesDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name, source             string
		start, end, line, column int
	}{
		{"EOF 后换行", "{\n  \"a\": 1\n", 9, 10, 2, 8},
		{"EOF 后多行空白", "{\n  \"a\": 1\n \t\n", 9, 10, 2, 8},
		{"EOF 后无空白", "{\n  \"a\": 1", 9, 10, 2, 8},
		{"Unicode 值后的 EOF", "{\n  \"a\": \"中🙂\"\n", 17, 18, 2, 11},
		{"字符串内非法换行", "{\n  \"a\": \"中\n", 10, 13, 2, 9},
		{"非法值", "{\n  \"a\": ?\n}", 9, 10, 2, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, action := range []struct {
				name string
				run  func(*FormatEditor)
			}{
				{"格式化", (*FormatEditor).Format},
				{"压缩", (*FormatEditor).Compact},
				{"字段排序", (*FormatEditor).SortFields},
			} {
				t.Run(action.name, func(t *testing.T) {
					e := NewFormatEditor("editor-1", tc.source)
					tt, wake := mountEditor(e)
					version := e.Buffer.Version()
					action.run(e)
					waitForEditor(t, wake)
					tt.Frame()
					if e.Loading() || e.Buffer.String() != tc.source || e.Buffer.Version() != version || e.Buffer.CanUndo() {
						t.Fatal("失败的变换不应修改文档或撤销历史")
					}
					diagnostics := e.Diagnostics()
					if len(diagnostics) != 1 {
						t.Fatalf("失败后应立即显示一处红线，得到 %+v", diagnostics)
					}
					d := diagnostics[0]
					want := json.Validate(tc.source)[0]
					message := locales.JsonErrorLocation(tc.line, tc.column, want.Detail)
					if d.Start != tc.start || d.End != tc.end || d.Message != message || !tt.HasText(message) {
						t.Fatalf("Toast 与红线应使用相同诊断位置：诊断 %+v，期望范围 [%d,%d)、提示 %q，界面文本 %q", d, tc.start, tc.end, message, tt.Texts())
					}
				})
			}
		})
	}
}

func TestStaleAndDuplicateTransforms(t *testing.T) {
	for _, shouldClear := range []bool{false, true} {
		t.Run(map[bool]string{false: "编辑后丢弃旧结果", true: "清空后丢弃旧结果"}[shouldClear], func(t *testing.T) {
			e := NewFormatEditor("editor-1", "{}")
			tt, wake := mountEditor(e)
			started, release := make(chan struct{}), make(chan struct{})
			e.transform(func(source string) (string, error) {
				close(started)
				<-release
				return `{"old":true}`, nil
			})
			waitForEditor(t, started)
			duplicate := make(chan struct{}, 1)
			e.transform(func(string) (string, error) { duplicate <- struct{}{}; return "null", nil })
			want := `{"new":true}`
			if shouldClear {
				e.Clear()
				want = ""
			} else {
				e.Buffer.Set(want)
			}
			close(release)
			waitForEditor(t, wake)
			tt.Frame()
			if e.Loading() || e.Buffer.String() != want {
				t.Fatalf("旧结果覆盖了新文档：%q", e.Buffer.String())
			}
			select {
			case <-duplicate:
				t.Fatal("忙碌期间启动了重复变换")
			default:
			}
		})
	}
}

func TestClearEmptyDocumentCancelsTransform(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ready     bool
		transform func(string) (string, error)
	}{
		{"运行中成功", false, json.Escape},
		{"已完成成功", true, json.Escape},
		{"运行中失败", false, json.Format},
		{"已完成失败", true, json.Format},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewFormatEditor("editor-1", "")
			tt, wake := mountEditor(e)
			version := e.Buffer.Version()
			started, release := make(chan struct{}), make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
			})
			e.transform(func(source string) (string, error) {
				close(started)
				<-release
				return tc.transform(source)
			})
			waitForEditor(t, started)
			if tc.ready {
				close(release)
				waitForEditor(t, wake)
			}
			e.Clear()
			if e.Loading() {
				t.Fatal("清空空文档应立即撤销待提交变换并恢复空闲状态")
			}
			if !tc.ready {
				close(release)
				waitForEditor(t, wake)
			}
			tt.Frame()
			if e.Loading() || e.Buffer.String() != "" || e.Buffer.Version() != version || e.Buffer.CanUndo() {
				t.Fatal("取消的变换不应写回空文档、改变版本或新增撤销记录")
			}
			if strings.Contains(strings.Join(tt.Texts(), "\n"), "JSON 格式错误") {
				t.Fatal("取消的变换不应显示错误提示")
			}
		})
	}
}

func TestClearAllowsNewTransformBeforeCanceledTaskFinishes(t *testing.T) {
	for _, oldFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "旧任务先完成", false: "新任务先完成"}[oldFirst], func(t *testing.T) {
			e := NewFormatEditor("editor-1", "")
			tt, wake := mountEditor(e)
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
			texts := [2]string{`{"old":true}`, `{"new":true}`}
			for i := range started {
				e.transform(func(string) (string, error) {
					close(started[i])
					<-release[i]
					return texts[i], nil
				})
				waitForEditor(t, started[i])
				if i == 0 {
					e.Clear()
					if e.Loading() {
						t.Fatal("清空后应允许立即启动新变换")
					}
				}
			}
			order := [2]int{0, 1}
			if !oldFirst {
				order = [2]int{1, 0}
			}
			completed := [2]bool{}
			for _, i := range order {
				close(release[i])
				waitForEditor(t, wake)
				completed[i] = true
				tt.Frame()
				want := ""
				if completed[1] {
					want = texts[1]
				}
				if e.Loading() != !completed[1] || e.Buffer.String() != want {
					t.Fatalf("旧任务不应影响新任务：文本 %q，忙碌 %v；期望 %q，%v", e.Buffer.String(), e.Loading(), want, !completed[1])
				}
			}
			if !e.Buffer.Undo() || e.Buffer.String() != "" || e.Buffer.CanUndo() {
				t.Fatal("只有新变换应产生一次可撤销编辑")
			}
		})
	}
}

func TestCopyClearAndViewActionsPreserveDocumentHistory(t *testing.T) {
	source := "{\n  \"a\": {\n    \"b\": 1\n  }\n}"
	e := NewFormatEditor("editor-1", source)
	tt, _ := mountEditor(e)
	e.Select(2, 5)
	selection := e.Selections()[0]
	var clipboard string
	e.CopyAll(func(text string) { clipboard = text })
	if clipboard != source || e.Selections()[0] != selection {
		t.Fatal("复制全文应保留选区")
	}
	e.ToggleWrap()
	tt.Frame()
	if !e.WrapEnabled() {
		t.Fatal("未开启自动换行")
	}
	e.ToggleWrap()
	e.FoldAll()
	tt.Frame()
	if len(e.Folded()) == 0 {
		t.Fatal("全部折叠未隐藏嵌套内容")
	}
	e.UnfoldAll()
	tt.Frame()
	if len(e.Folded()) != 0 || e.WrapEnabled() || e.Buffer.CanUndo() || e.Buffer.String() != source {
		t.Fatal("视图操作不应改变文档或撤销历史")
	}
	e.Search()
	tt.Frame()
	if _, ok := tt.Find(locales.FindQuery()); !ok {
		t.Fatal("搜索未打开原生查找栏")
	}
	e.Clear()
	tt.Frame()
	if e.Buffer.String() != "" || len(e.Diagnostics()) != 0 {
		t.Fatal("清空应移除文本及诊断")
	}
	e.Clear()
	if !e.Buffer.Undo() || e.Buffer.String() != source || e.Buffer.CanUndo() {
		t.Fatal("重复清空不应新增空撤销步骤")
	}
}

func TestPreparedTransformVersionAndDiagnostics(t *testing.T) {
	e := NewFormatEditor("editor-1", "{}")
	mountEditor(e)
	version := e.Buffer.Version()
	e.Buffer.Set("[]")
	if e.ApplyTransform(PreparedTransform{Version: version, Text: "null", Changed: true}) || e.Buffer.String() != "[]" {
		t.Fatal("不应提交过期的预处理结果")
	}
	result := PreparedTransform{Version: e.Buffer.Version(), Text: "{", Changed: true, Diagnostics: json.Validate("{")}
	if !e.ApplyTransform(result) || e.Buffer.String() != "{" || len(e.Diagnostics()) != 1 {
		t.Fatal("有效预处理结果应同步提交文本及诊断")
	}
	version = e.Buffer.Version()
	if !e.ApplyTransform(PreparedTransform{Version: version}) || e.Buffer.Version() != version || len(e.Diagnostics()) != 0 {
		t.Fatal("内容不变的预处理结果应仅更新诊断")
	}
}

func TestFormatShortcut(t *testing.T) {
	e := NewFormatEditor("editor-1", `{"a":1}`)
	tt, wake := mountEditor(e)
	e.Focus()
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyF)
	waitForEditor(t, wake)
	tt.Frame()
	if e.Buffer.String() != "{\n  \"a\": 1\n}" {
		t.Fatalf("格式化快捷键未提交结果：%q", e.Buffer.String())
	}
}

func TestThemeChangesAndCanceledFocusPreserveEditingState(t *testing.T) {
	e := NewFormatEditor("editor-1", "{}")
	tt, _ := mountEditor(e)
	e.Buffer.Set(`{"中文":1}`)
	e.Select(2, 4)
	selection := e.Selections()[0]
	version := e.Buffer.Version()
	e.Focus()
	e.CancelFocus()
	tt.Frame()
	if e.Focused() {
		t.Fatal("撤销的聚焦请求不应抢占焦点")
	}
	for _, dark := range []bool{true, false} {
		tt.SetDark(dark)
		if e.Buffer.Version() != version || e.Selections()[0] != selection || e.Buffer.String() != `{"中文":1}` {
			t.Fatal("切换主题不应改变文档或选区")
		}
	}
	if !e.Buffer.Undo() || e.Buffer.String() != "{}" {
		t.Fatal("切换主题应保留原有撤销记录")
	}
}

func TestDiagnosticsReadImmutableSnapshotChunks(t *testing.T) {
	for _, source := range []string{
		"{\r\n  \"a\": 1\r\n}",
		`{"text":"` + strings.Repeat("中文🙂\\\"", 2000) + `"}`,
	} {
		e := NewFormatEditor("editor-1", source)
		// Remove the closing brace to create EOF after an edit to the text tree.
		e.Buffer.Edit(mygoeditor.Edit{Start: e.Buffer.Len() - 1, End: e.Buffer.Len()})
		snapshot := e.Buffer.Snapshot()
		want := editor.Diagnostics(json.Validate(snapshot.String()))
		e.Buffer.Set("{}")
		if got := e.diagnose(snapshot); len(got) != 1 || !reflect.DeepEqual(got, want) {
			t.Fatalf("快照分块诊断应保留编辑前的文本及位置：得到 %+v，期望 %+v", got, want)
		}
	}
}

func TestDebouncedDiagnosticsAndRecovery(t *testing.T) {
	e := NewFormatEditor("editor-1", "")
	tt, wake := mountEditor(e)
	e.Buffer.Set(`{"a":}`)
	tt.Frame()
	tt.Advance(200 * time.Millisecond)
	e.Buffer.Set(`{"b":}`)
	tt.Frame()
	tt.Advance(299 * time.Millisecond)
	if e.diagnosing || len(e.Diagnostics()) != 0 {
		t.Fatal("连续编辑应重新计时，300 毫秒前不执行诊断")
	}
	tt.Advance(time.Millisecond)
	waitForEditor(t, wake)
	tt.Frame()
	if ds := e.Diagnostics(); len(ds) != 1 || ds[0].Severity != mygoeditor.SeverityError || ds[0].Start != 5 {
		t.Fatalf("诊断未正确提交：%+v", ds)
	}
	e.Buffer.Set("{}")
	tt.Frame()
	if len(e.Diagnostics()) != 0 {
		t.Fatal("编辑时应立即清除旧诊断")
	}
	tt.Advance(300 * time.Millisecond)
	waitForEditor(t, wake)
	tt.Frame()
	if len(e.Diagnostics()) != 0 || e.diagnosing {
		t.Fatal("有效文档校验后应保持无错误状态")
	}
	e.Buffer.Undo()
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	waitForEditor(t, wake)
	tt.Frame()
	if len(e.Diagnostics()) != 1 {
		t.Fatal("撤销后应重新校验原有错误")
	}
	e.Clear()
	tt.Frame()
	tt.Advance(time.Second)
	if e.diagnosing || len(e.Diagnostics()) != 0 || !e.deadline.IsZero() {
		t.Fatal("空文档不应继续调度诊断")
	}
}

func TestBackgroundWorkWithoutInvalidate(t *testing.T) {
	e := NewFormatEditor("editor-1", "")
	tt := ui.NewTester(e.View, 900, 600)
	e.Buffer.Set(`{"a":}`)
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	select {
	case result := <-e.diagnosticResult:
		e.diagnosticResult <- result
	case <-time.After(5 * time.Second):
		t.Fatal("诊断未完成")
	}
	tt.Advance(16 * time.Millisecond)
	if e.diagnosing || len(e.Diagnostics()) != 1 {
		t.Fatal("没有刷新回调时，诊断应在定时帧提交")
	}

	e.Buffer.Set(`{"a":1}`)
	release := make(chan struct{})
	e.transform(func(source string) (string, error) {
		<-release
		return json.Format(source)
	})
	tt.Frame()
	close(release)
	select {
	case result := <-e.pendingTransform:
		e.pendingTransform <- result
	case <-time.After(5 * time.Second):
		t.Fatal("变换未完成")
	}
	tt.Advance(16 * time.Millisecond)
	if e.Loading() || e.Buffer.String() != "{\n  \"a\": 1\n}" {
		t.Fatalf("没有刷新回调时，变换应在定时帧提交：%q", e.Buffer.String())
	}
}

func TestStaleDiagnosticsAndEditorIsolation(t *testing.T) {
	first, second := NewFormatEditor("editor-1", ""), NewFormatEditor("editor-2", "")
	tt, _ := mountEditor(first)
	mountEditor(second)
	old := first.Buffer.Version()
	first.Buffer.Set("[]")
	first.diagnosing = true
	first.diagnosticResult <- diagnosticsResult{version: old, diagnostic: []mygoeditor.Diagnostic{{Start: 0, End: 1, Message: "旧错误"}}}
	tt.Frame()
	if len(first.Diagnostics()) != 0 || len(second.Diagnostics()) != 0 || first.diagnosing {
		t.Fatal("过期诊断应被丢弃且不影响其他编辑器")
	}
	first.diagnosticResult <- diagnosticsResult{version: first.Buffer.Version(), diagnostic: []mygoeditor.Diagnostic{{Start: 0, End: 1, Message: "当前错误"}}}
	tt.Frame()
	if len(first.Diagnostics()) != 1 || len(second.Diagnostics()) != 0 {
		t.Fatal("诊断应仅应用到所属编辑器")
	}
}
