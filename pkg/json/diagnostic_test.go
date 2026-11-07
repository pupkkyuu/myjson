package json_test

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"myjson/pkg/json"
)

func TestValidateAcceptsValidOrBlankInput(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"", " \t\r\n", `{"a":1,"b":2}`} {
		if got := json.Validate(source); len(got) != 0 {
			t.Errorf("%q 不应产生诊断：%+v", source, got)
		}
	}
}

func TestValidateErrorRanges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, source             string
		start, end, line, column int
	}{
		{"非法值", `{"a":?}`, 5, 6, 1, 6},
		{"Unicode 列号", "{\n  \"中文🙂\": ?\n}", 18, 19, 2, 10},
		{"多字节非法字符", `{"a":中}`, 5, 8, 1, 6},
		{"缺少结束符", "{\"a\":1  \n", 5, 6, 1, 6},
		{"字符串内非法换行", "{\"a\":\"中\n文\"}", 6, 9, 1, 7},
		{"回车换行", "{\r\n\"a\":?}", 7, 8, 2, 5},
		{"额外文档", "{} []", 3, 4, 1, 4},
		{"不接受注释", "// comment\n{}", 0, 1, 1, 1},
		{"不接受不换行空格", "\u00a0", 0, 2, 1, 1},
		{"不接受全角空格", " \n\u3000", 2, 5, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := json.Validate(tc.source)
			if len(got) != 1 {
				t.Fatalf("应仅返回首个错误，得到 %+v", got)
			}
			d := got[0]
			if d.Start != tc.start || d.End != tc.end || d.Line != tc.line || d.Column != tc.column || d.Detail == "" {
				t.Fatalf("诊断 = %+v；期望范围 [%d,%d)，位置 %d:%d", d, tc.start, tc.end, tc.line, tc.column)
			}
			if !utf8.ValidString(tc.source[d.Start:d.End]) {
				t.Fatalf("诊断截断了 UTF-8 字符：%q", tc.source[d.Start:d.End])
			}
		})
	}
}

func TestLocationErrorPreservesCauseAndUnicodePosition(t *testing.T) {
	t.Parallel()
	source := "{\n\"中文🙂\":?}"
	_, err := json.Format(source)
	var located *json.LocationError
	var syntax *jsontext.SyntacticError
	if !errors.As(err, &located) || !errors.As(err, &syntax) {
		t.Fatalf("应保留错误链：%v", err)
	}
	if located.Line != 2 || located.Column != 7 || located.Unwrap() != syntax {
		t.Fatalf("位置或错误链不正确：%+v", located)
	}
	if !strings.Contains(located.Error(), "line 2, column 7:") || !strings.Contains(located.Error(), syntax.Error()) {
		t.Fatalf("错误文本缺少位置或原始详情：%q", located.Error())
	}
	wrapper := fmt.Errorf("转换失败：%w", err)
	ds := json.DiagnosticsForError(source, wrapper)
	if len(ds) != 1 || ds[0].Column != 7 || ds[0].Detail != syntax.Error() {
		t.Fatalf("包装错误诊断 = %+v", ds)
	}
}

func TestDiagnosticsForNonSyntaxErrors(t *testing.T) {
	t.Parallel()
	if got := json.DiagnosticsForError("{}", nil); got != nil {
		t.Fatalf("无错误应返回 nil：%+v", got)
	}
	for _, source := range []string{"", "中", "abc"} {
		ds := json.DiagnosticsForError(source, errors.New("自定义错误"))
		_, size := utf8.DecodeRuneInString(source)
		if len(ds) != 1 || ds[0].Start != 0 || ds[0].End != min(len(source), size) || ds[0].Line != 1 || ds[0].Column != 1 || ds[0].Detail != "自定义错误" {
			t.Fatalf("非语法错误诊断 = %+v，输入 %q", ds, source)
		}
	}
}

func TestUnescapeErrorOffsetsReferToOriginalInput(t *testing.T) {
	t.Parallel()
	_, err := json.Unescape(`中\q`)
	var located *json.LocationError
	var syntax *jsontext.SyntacticError
	if !errors.As(err, &located) || !errors.As(err, &syntax) || syntax.ByteOffset != 3 || located.Column != 2 {
		t.Fatalf("去转义错误偏移应对应原文：%v", err)
	}
}
