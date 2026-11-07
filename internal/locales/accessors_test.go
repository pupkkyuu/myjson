package locales_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"myjson/internal/locales"
)

func TestMessageAccessorsMatchChineseCatalog(t *testing.T) {
	data, err := os.ReadFile("zh-Hans.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	messages := reflect.ValueOf(locales.In("zh-Hans"))
	for key, raw := range catalog {
		if strings.HasPrefix(key, "@") {
			continue
		}
		var metadata struct{ Placeholders map[string]any }
		if err := json.Unmarshal(catalog["@"+key], &metadata); err != nil {
			t.Fatal(err)
		}
		if len(metadata.Placeholders) != 0 {
			continue
		}
		t.Run(key, func(t *testing.T) {
			var want string
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			name := string(unicode.ToUpper(rune(key[0]))) + key[1:]
			method := messages.MethodByName(name)
			if !method.IsValid() || method.Type().NumIn() != 0 || method.Type().NumOut() != 1 || method.Type().Out(0).Kind() != reflect.String {
				t.Fatalf("消息 %q 缺少无参数访问器 %s", key, name)
			}
			if got := method.Call(nil)[0].String(); got != want {
				t.Errorf("%s() = %q，期望 %q", name, got, want)
			}
		})
	}
}

func TestCurrentLanguageAccessors(t *testing.T) {
	if got, want := locales.EditorLabel(), locales.In("zh-Hans").EditorLabel(); got != want {
		t.Errorf("当前语言消息 = %q，期望 %q", got, want)
	}
	if got, want := locales.WithShortcut("搜索", "⌘F"), locales.In("zh-Hans").WithShortcut("搜索", "⌘F"); got != want {
		t.Errorf("当前语言参数化消息 = %q，期望 %q", got, want)
	}
}
