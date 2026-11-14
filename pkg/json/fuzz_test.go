package json_test

import (
	"bytes"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"reflect"
	"testing"
	"unicode/utf8"

	"myjson/pkg/json"
)

func FuzzEscapeRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "中文🙂\n", `{"a":"\\"}`, "\x00<>&", "\u2028"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if !utf8.ValidString(source) || len(source) > 64*1024 {
			t.Skip()
		}
		escaped, err := json.Escape(source)
		if err != nil || !stdjson.Valid([]byte(escaped)) {
			t.Fatalf("转义返回无效结果：%q，%v", escaped, err)
		}
		decoded, err := json.Unescape(escaped)
		if err != nil || decoded != source {
			t.Fatalf("转义往返改变原文：%q -> %q，%v", source, decoded, err)
		}
	})
}

func FuzzJSONTransforms(f *testing.F) {
	for _, seed := range []string{"{}", `{"z":1,"a":[true,null,{"b":2,"a":1}]}`, `{"a":1,"a":2}`, `"\ud800"`, "1e999", "{", "", `{"\ue000":1,"😀":2}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 64*1024 {
			t.Skip()
		}
		valid := jsontext.Value(source).IsValid()
		for _, transform := range []func(string) (string, error){json.Format, json.Compact, json.SortFields} {
			got, err := transform(source)
			if !valid {
				if err == nil || got != "" {
					t.Fatalf("非法输入被接受：%q -> %q，%v", source, got, err)
				}
				continue
			}
			if err != nil || !stdjson.Valid([]byte(got)) {
				t.Fatalf("有效输入转换失败：%q -> %q，%v", source, got, err)
			}
			again, err := transform(got)
			if err != nil || got != again {
				t.Fatalf("重复转换改变结果：%q -> %q，%v", got, again, err)
			}
			decode := func(text string) any {
				decoder := stdjson.NewDecoder(bytes.NewBufferString(text))
				decoder.UseNumber()
				var value any
				if err := decoder.Decode(&value); err != nil {
					t.Fatal(err)
				}
				return value
			}
			if !reflect.DeepEqual(decode(source), decode(got)) {
				t.Fatalf("转换改变 JSON 数据：%q -> %q", source, got)
			}
		}
		if valid {
			want := jsontext.Value(source)
			if err := want.Format(); err != nil {
				t.Fatal(err)
			}
			formatted, err := json.Format(source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Compact(formatted)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("格式化结果不符合 JSON 文本规范化：%q -> %q，期望 %q", source, got, want)
			}
		}
	})
}

func FuzzDiagnosticRanges(f *testing.F) {
	for _, seed := range []string{"", "{}", "{\"a\":中}", "{\"a\":\"中\n文\"}", "\xff", " \n", "{", "{} []", "true false", `{"a":1,"\u0061":2}`, `"\ud800"`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 64*1024 {
			t.Skip()
		}
		diagnostics := json.ValidateBytes([]byte(source))
		valid := json.IsBlank(source) || jsontext.Value(source).IsValid()
		if (len(diagnostics) == 0) != valid || !reflect.DeepEqual(diagnostics, json.Validate(source)) {
			t.Fatalf("字节和字符串校验应保持严格 JSON 语义：%q，诊断 %+v", source, diagnostics)
		}
		for _, d := range diagnostics {
			if d.Start < 0 || d.End < d.Start || d.End > len(source) || d.Line < 1 || d.Column < 1 || d.Detail == "" {
				t.Fatalf("越界或不完整的诊断：%+v，输入 %q", d, source)
			}
			if utf8.ValidString(source) && !utf8.ValidString(source[d.Start:d.End]) {
				t.Fatalf("诊断截断 UTF-8 字符：%+v，输入 %q", d, source)
			}
			_, err := json.Format(source)
			want := json.DiagnosticsForError(source, err)[0]
			if d.Start != want.Start || d.End != want.End || d.Line != want.Line || d.Column != want.Column {
				t.Fatalf("校验与变换的错误位置不一致：%q，诊断 %+v，期望 %+v", source, d, want)
			}
		}
	})
}
