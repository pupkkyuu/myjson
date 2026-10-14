package json_test

import (
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"

	"myjson/pkg/json"
)

func TestFormat(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, want string }{
		{"嵌套对象", ` {"b":[1,{"a":true}],"c":null} `, "{\n  \"b\": [\n    1,\n    {\n      \"a\": true\n    }\n  ],\n  \"c\": null\n}"},
		{"保留数字", `{"z":900719925474099312345,"a":1.2300e+40,"b":-0}`, "{\n  \"z\": 900719925474099312345,\n  \"a\": 1.2300e+40,\n  \"b\": -0\n}"},
		{"规范化字符串转义", `{"a":"\u4e2d\/\\\""}`, "{\n  \"a\": \"中/\\\\\\\"\"\n}"},
		{"空对象", " \n{}\t", "{}"},
		{"顶层标量", " 1.00e-3\n", "1.00e-3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Format(tc.source)
			if err != nil || got != tc.want {
				t.Fatalf("格式化结果 = %q，错误 = %v；期望 %q", got, err, tc.want)
			}
		})
	}
}

func TestCompact(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, want string }{
		{"仅移除字符串外空白", " \n{ \"s\": \" a b \\t\", \"a\": [1, 2] }\n", `{"s":" a b \t","a":[1,2]}`},
		{"保留数字并规范化字符串", ` {"z":900719925474099312345, "n":1.2300e+40, "s":"\u4e2d\/"} `, `{"z":900719925474099312345,"n":1.2300e+40,"s":"中/"}`},
		{"顶层字符串", ` "中文 🙂" `, `"中文 🙂"`},
		{"顶层标量", "\r\n false\t", "false"},
		{"空容器", "[ { }, [ ] ]", "[{},[]]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Compact(tc.source)
			if err != nil || got != tc.want {
				t.Fatalf("压缩结果 = %q，错误 = %v；期望 %q", got, err, tc.want)
			}
		})
	}
}

func TestStructuredTransformsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	for name, transform := range map[string]func(string) (string, error){
		"格式化": json.Format, "压缩": json.Compact, "字段排序": json.SortFields,
	} {
		t.Run(name, func(t *testing.T) {
			for _, source := range []string{"", "{", "{}[]", "{\"a\":\"换\n行\"}"} {
				t.Run(source, func(t *testing.T) {
					got, err := transform(source)
					var located *json.LocationError
					var syntax *jsontext.SyntacticError
					if got != "" || !errors.As(err, &located) || !errors.As(err, &syntax) {
						t.Fatalf("非法输入应返回空结果及可解包的语法错误，得到 %q，%v", got, err)
					}
					if located.Line < 1 || located.Column < 1 {
						t.Fatalf("错误位置必须从 1 开始：%+v", located)
					}
				})
			}
		})
	}
}

func TestStrictJSONTextValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source string }{
		{"重复键", `{"a":1,"a":2}`},
		{"转义后的重复键", `{"a":1,"\u0061":2}`},
		{"无效 UTF-8", "{\"s\":\"\xff\"}"},
		{"孤立代理项", `"\ud800"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := json.Validate(tc.source); len(got) != 1 {
				t.Fatalf("应报告无效输入：%+v", got)
			}
			for name, transform := range map[string]func(string) (string, error){
				"格式化": json.Format, "压缩": json.Compact, "字段排序": json.SortFields,
			} {
				if got, err := transform(tc.source); err == nil || got != "" {
					t.Errorf("%s 应拒绝无效输入：%q，%v", name, got, err)
				}
			}
		})
	}
}

func TestEscapeAndUnescape(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"", "普通文本", "中文🙂\n\t\r\x00", `<script>&/`, `{"a":"\u4e2d"}`, "\u2028\u2029", "\\\""} {
		t.Run(source, func(t *testing.T) {
			escaped, err := json.Escape(source)
			if err != nil || !stdjson.Valid([]byte(escaped)) || strings.HasSuffix(escaped, "\n") {
				t.Fatalf("转义必须返回有效的 JSON 字符串且不附加换行：%q，%v", escaped, err)
			}
			var decoded string
			if err := stdjson.Unmarshal([]byte(escaped), &decoded); err != nil || decoded != source {
				t.Fatalf("标准解析器解码结果 = %q，错误 = %v", decoded, err)
			}
			got, err := json.Unescape(escaped)
			if err != nil || got != source {
				t.Fatalf("转义往返结果 = %q，错误 = %v；期望 %q", got, err, source)
			}
		})
	}
	got, err := json.Escape("<>&")
	if err != nil || got != `"<>&"` {
		t.Fatalf("不应对 HTML 字符额外转义：%q，%v", got, err)
	}
}

func TestUnescapeInputForms(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, want string }{
		{"解除一层", `"\"文字\""`, `"文字"`},
		{"无外层引号", `\"a\"\n\t`, "\"a\"\n\t"},
		{"普通文本", "普通文本", "普通文本"},
		{"空文本", "", ""},
		{"引号外空白", " \t\"中文\"\n", "中文"},
		{"保留对象及空白", " \n{\"a\":1} ", " \n{\"a\":1} "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Unescape(tc.source)
			if err != nil || got != tc.want {
				t.Fatalf("去转义结果 = %q，错误 = %v；期望 %q", got, err, tc.want)
			}
		})
	}
	for _, source := range []string{`\q`, `\u12`, "末尾\\", "非法\n换行"} {
		t.Run("非法转义/"+source, func(t *testing.T) {
			got, err := json.Unescape(source)
			if err == nil || got != "" {
				t.Fatalf("非法转义应失败：%q，%v", got, err)
			}
		})
	}
}

func TestSortFields(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, compactWant string }{
		{"递归排序并保留数组顺序", `{"z":[{"b":2,"a":1},3,2],"a":{"d":4,"c":3}}`, `{"a":{"c":3,"d":4},"z":[{"a":1,"b":2},3,2]}`},
		{"UTF-16 顺序", `{"\ue000":1,"😀":2,"a":3}`, `{"a":3,"😀":2,"":1}`},
		{"按解码后的键排序", `{"z":1,"\u0061":2,"":3}`, `{"":3,"a":2,"z":1}`},
		{"保留数字并规范化转义", `{"z":900719925474099312345,"b":-0,"a":"\u4e2d\/","c":1.00e+99}`, `{"a":"中/","b":-0,"c":1.00e+99,"z":900719925474099312345}`},
		{"空容器", `[{},[]]`, `[{},[]]`},
		{"含转义引号的键", `{"z":1,"a\"b":2,"a\\b":3}`, `{"a\"b":2,"a\\b":3,"z":1}`},
		{"顶层标量", `1.00e+9`, `1.00e+9`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.SortFields(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			compact, err := json.Compact(got)
			if err != nil || compact != tc.compactWant {
				t.Fatalf("排序结果 = %q，错误 = %v；期望 %q", compact, err, tc.compactWant)
			}
			formatted, err := json.Format(tc.compactWant)
			if err != nil || got != formatted {
				t.Fatalf("排序结果未使用 2 空格缩进：%q", got)
			}
		})
	}
}
