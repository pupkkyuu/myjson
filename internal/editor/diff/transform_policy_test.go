package diff

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor"
	"myjson/internal/editor/format"
	"myjson/pkg/json"
)

func transformBothEditors(t *testing.T, source, right string, singleAction func(*format.FormatEditor), pairedAction func(*DiffEditor)) (*format.FormatEditor, *DiffEditor) {
	t.Helper()
	single := format.NewFormatEditor("editor-1", source)
	wake := make(chan struct{}, 8)
	single.Invalidate = func() { wake <- struct{}{} }
	singleView := ui.NewTester(single.View, 900, 600)
	paired := NewDiffEditor("editor-2", source, right)
	pairedView, pairedWake := mountDiff(paired)
	for _, pane := range []*format.FormatEditor{single, paired.left, paired.right} {
		pane.Select(0, min(2, pane.Buffer.Len()))
	}
	singleAction(single)
	pairedAction(paired)
	awaitDiff(t, wake)
	awaitDiff(t, pairedWake)
	singleView.Frame()
	pairedView.Frame()
	if single.Loading() || paired.Loading() {
		t.Fatal("变换完成后应退出忙碌状态")
	}
	return single, paired
}

func TestWhitespaceTransformPoliciesMatch(t *testing.T) {
	for _, action := range []struct {
		name      string
		skipBlank bool
		run       func(string) (string, error)
		single    func(*format.FormatEditor)
		paired    func(*DiffEditor)
	}{
		{"格式化", true, json.Format, (*format.FormatEditor).Format, (*DiffEditor).Format},
		{"压缩", true, json.Compact, (*format.FormatEditor).Compact, (*DiffEditor).Compact},
		{"排序", true, json.SortFields, (*format.FormatEditor).SortFields, (*DiffEditor).SortFields},
		{"转义", false, json.Escape, (*format.FormatEditor).Escape, (*DiffEditor).Escape},
		{"去转义", false, json.Unescape, (*format.FormatEditor).Unescape, (*DiffEditor).Unescape},
	} {
		t.Run(action.name, func(t *testing.T) {
			for _, input := range []struct{ name, source string }{
				{"空串", ""}, {"空格", " "}, {"制表符", "\t"}, {"回车", "\r"}, {"换行", "\n"},
				{"混合JSON空白", " \t\r\n"}, {"NBSP", "\u00a0"}, {"全角空格", "\u3000"}, {"垂直制表符", "\v"},
			} {
				t.Run(input.name, func(t *testing.T) {
					// 缓冲区内部使用 LF；转换库继续负责实际转义和严格 JSON 语义。
					source := strings.ReplaceAll(input.source, "\r\n", "\n")
					want := source
					var err error
					if !action.skipBlank || !json.IsBlank(source) {
						want, err = action.run(source)
					}
					diagnostics := json.Validate(want)
					right := `{ "z": 2, "a": 1 }`
					wantRight, rightErr := action.run(right)
					if rightErr != nil {
						t.Fatal(rightErr)
					}
					if err != nil {
						want, wantRight = source, right
						diagnostics = json.DiagnosticsForError(source, err)
					}
					single, paired := transformBothEditors(t, input.source, right, action.single, action.paired)
					for _, pane := range []*format.FormatEditor{single, paired.left} {
						if pane.Buffer.String() != want || !reflect.DeepEqual(pane.Diagnostics(), editor.Diagnostics(diagnostics)) {
							t.Fatalf("两种编辑器应遵循同一规则：文本 %q，诊断 %+v；期望 %q，%+v", pane.Buffer.String(), pane.Diagnostics(), want, diagnostics)
						}
						if want == source && (pane.Buffer.Version() != 0 || pane.Buffer.CanUndo() || pane.Cursor() != min(2, len(source))) {
							t.Fatal("跳过或失败的变换不应修改版本、撤销历史和位置")
						}
						ending := "\n"
						if strings.Contains(input.source, "\r\n") {
							ending = "\r\n"
						}
						if pane.Buffer.LineEnding() != ending {
							t.Fatal("变换不应改变缓冲区的换行保存偏好")
						}
					}
					if paired.right.Buffer.String() != wantRight {
						t.Fatalf("另一侧提交状态错误：%q，期望 %q", paired.right.Buffer.String(), wantRight)
					}
					if err != nil && (paired.right.Buffer.CanUndo() || len(paired.right.Diagnostics()) != 0) {
						t.Fatal("一侧失败时不应提交另一侧的文本或诊断")
					}
				})
			}
		})
	}
}

func TestSkippedBlankSideDoesNotCommitWhenOtherSideFails(t *testing.T) {
	for _, action := range []struct {
		name   string
		single func(*format.FormatEditor)
		paired func(*DiffEditor)
	}{
		{"格式化", (*format.FormatEditor).Format, (*DiffEditor).Format},
		{"压缩", (*format.FormatEditor).Compact, (*DiffEditor).Compact},
		{"排序", (*format.FormatEditor).SortFields, (*DiffEditor).SortFields},
	} {
		t.Run(action.name, func(t *testing.T) {
			blank, invalid := " \t\n", `{"a":}`
			_, paired := transformBothEditors(t, blank, invalid, action.single, action.paired)
			if paired.left.Buffer.String() != blank || paired.right.Buffer.String() != invalid || paired.left.Buffer.CanUndo() || paired.right.Buffer.CanUndo() || len(paired.left.Diagnostics()) != 0 || len(paired.right.Diagnostics()) != 1 {
				t.Fatal("跳过空白侧后，另一侧失败时应保留两侧文档并只诊断失败侧")
			}
		})
	}
}
