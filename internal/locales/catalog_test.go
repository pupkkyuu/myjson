package locales_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

func TestChineseCatalogMetadataAndMessages(t *testing.T) {
	data, err := os.ReadFile("zh-Hans.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	for key, raw := range catalog {
		if strings.HasPrefix(key, "@@") {
			continue
		}
		if after, ok := strings.CutPrefix(key, "@"); ok {
			if _, ok := catalog[after]; !ok {
				t.Errorf("说明 %q 没有对应消息", key)
			}
			var metadata struct {
				Description string `json:"description"`
			}
			if err := json.Unmarshal(raw, &metadata); err != nil {
				t.Fatal(err)
			}
			if !strings.ContainsFunc(metadata.Description, func(r rune) bool { return unicode.Is(unicode.Han, r) }) {
				t.Errorf("消息说明 %q 应使用中文：%q", key, metadata.Description)
			}
			continue
		}
		var message string
		if err := json.Unmarshal(raw, &message); err != nil || message == "" {
			t.Errorf("消息 %q 为空或格式错误：%v", key, err)
		}
		if _, ok := catalog["@"+key]; !ok {
			t.Errorf("消息 %q 缺少说明", key)
		}
	}
}

func TestCatalogFormatsPlaceholdersAndFallback(t *testing.T) {
	for _, locale := range []string{"", "zh-Hans", "zh-CN", "en-US", "de"} {
		msgs := locales.In(locale)
		for _, tc := range []struct{ got, want string }{
			{msgs.EditorLabel(), "JSON 编辑器"},
			{msgs.FormatTabTitle("18446744073709551615"), "格式化 18446744073709551615"},
			{msgs.ComparisonTabTitle("2"), "对比 2"},
			{msgs.CountTabs(12345), "标签页（12345）"},
			{msgs.CloseTab("文档{}"), "关闭 文档{}"},
			{msgs.WithShortcut("格式化", "⌘⇧F"), "格式化（⌘⇧F）"},
			{msgs.JsonErrorLocation(1234, 56, "错误 {detail}"), "JSON 格式错误：第 1234 行，第 56 列：错误 {detail}"},
			{msgs.DiffLeftError("错误"), "左侧：错误"},
			{msgs.DiffRightError("错误"), "右侧：错误"},
		} {
			if tc.got != tc.want {
				t.Errorf("语言 %q 的结果 = %q，期望 %q", locale, tc.got, tc.want)
			}
		}
	}
}

func TestWindowLocaleFallsBackToChinese(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) { ui.Text(c, locales.EditorLabel()) }, 300, 100)
	for _, locale := range []string{"zh-Hans", "en-US", "ja-JP"} {
		tt.SetLocale(locale)
		if !tt.HasText("JSON 编辑器") {
			t.Fatalf("窗口语言 %q 未回退到中文", locale)
		}
	}
}
