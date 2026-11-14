package settings

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/palette"
)

func TestSettingsPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MyJson", "settings.json")
	want := DefaultSaved()
	want.Preferences.FontSize = 22
	want.Preferences.DefaultWrap = true
	want.Preferences.TabSize = 4
	want.Preferences.DiffLayout = mygoeditor.DiffUnified
	want.Zoom = 125
	want.Theme = mygo.ThemeDark
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	want.Theme = mygo.ThemeLight
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil || got != want {
		t.Fatalf("replacement got %+v, %v", got, err)
	}
}

func TestSavedSettingsUseGoFieldNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := Save(path, DefaultSaved()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]jsontext.Value
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Sorted(maps.Keys(root)), []string{"DarkWash", "LightWash", "Preferences", "Theme", "Zoom"}; !slices.Equal(got, want) {
		t.Fatalf("配置顶层字段 = %v，期望 %v", got, want)
	}
	var preferences map[string]jsontext.Value
	if err := json.Unmarshal(root["Preferences"], &preferences); err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Sorted(maps.Keys(preferences)), []string{"ActiveLine", "AutoClose", "DefaultWrap", "DiffInline", "DiffLayout", "FontSize", "IndentGuides", "LineHeight", "TabSize", "Whitespace"}; !slices.Equal(got, want) {
		t.Fatalf("编辑器偏好字段 = %v，期望 %v", got, want)
	}
}

func TestLoadSettingsAcceptsLegacyAndGoNames(t *testing.T) {
	contents := `{
		"Preferences": {
			"FontSize": 22, "LineHeight": 1.8, "TabSize": 8,
			"DefaultWrap": true, "Whitespace": 2, "IndentGuides": false,
			"ActiveLine": true, "AutoClose": false, "DiffLayout": 1, "DiffInline": 1
		},
		"Zoom": 150, "Theme": "dark", "LightWash": "rex", "DarkWash": "tide"
	}`
	want := DefaultSaved()
	want.Preferences.FontSize = 22
	want.Preferences.LineHeight = 1.8
	want.Preferences.TabSize = 8
	want.Preferences.DefaultWrap = true
	want.Preferences.Whitespace = mygoeditor.WhitespaceAll
	want.Preferences.IndentGuides = false
	want.Preferences.ActiveLine = true
	want.Preferences.AutoClose = false
	want.Preferences.DiffLayout = mygoeditor.DiffUnified
	want.Preferences.DiffInline = mygoeditor.DiffInlineWord
	want.Zoom, want.Theme = 150, mygo.ThemeDark
	want.LightWash, want.DarkWash = "rex", "tide"
	legacy := strings.NewReplacer(
		`"Preferences"`, `"preferences"`, `"Zoom"`, `"zoom"`, `"Theme"`, `"theme"`,
		`"LightWash"`, `"lightWash"`, `"DarkWash"`, `"darkWash"`,
	).Replace(contents)
	camelCase := strings.NewReplacer(
		`"FontSize"`, `"fontSize"`, `"LineHeight"`, `"lineHeight"`, `"TabSize"`, `"tabSize"`,
		`"DefaultWrap"`, `"defaultWrap"`, `"Whitespace"`, `"whitespace"`, `"IndentGuides"`, `"indentGuides"`,
		`"ActiveLine"`, `"activeLine"`, `"AutoClose"`, `"autoClose"`, `"DiffLayout"`, `"diffLayout"`, `"DiffInline"`, `"diffInline"`,
	).Replace(legacy)
	for _, tc := range []struct{ name, contents string }{
		{"Go字段名", contents}, {"旧混合命名", legacy}, {"小驼峰", camelCase},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(tc.contents), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := Load(path)
			if err != nil || got != want {
				t.Fatalf("读取配置 = %+v，错误 %v；期望 %+v", got, err, want)
			}
			if err := Save(path, got); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"preferences"`) || strings.Contains(string(data), `"fontSize"`) || strings.Contains(string(data), `"lightWash"`) {
				t.Fatalf("重新保存后仍含旧字段名：%s", data)
			}
			if reloaded, err := Load(path); err != nil || reloaded != want {
				t.Fatalf("统一字段名后设置丢失：%+v，%v", reloaded, err)
			}
		})
	}
}

func TestSeparateAppearanceColorsSurviveRestart(t *testing.T) {
	var state State
	state.SetWash(false, "rex")
	state.SetWash(true, "tide")
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := Save(path, state.Snapshot(mygo.ThemeSystem)); err != nil {
		t.Fatal(err)
	}
	saved, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	restored.Restore(saved)
	if restored.Wash(false) != "rex" || restored.Wash(true) != "tide" {
		t.Fatalf("明亮和黑暗配色未分别保存：%+v", saved)
	}
	restored.SetWash(false, "meadow")
	if restored.Wash(true) != "tide" {
		t.Fatal("修改明亮配色不应改变黑暗配色")
	}
	restored.SetWash(true, "dusk")
	if restored.Wash(false) != "meadow" {
		t.Fatal("修改黑暗配色不应改变明亮配色")
	}
}

func TestInvalidSettingsAndPartialDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"preferences":{"FontSize":999,"TabSize":4},"zoom":13,"theme":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Preferences.FontSize != 13 || got.Preferences.TabSize != 4 || !got.Preferences.AutoClose || got.Zoom != 100 || got.Theme != mygo.ThemeSystem || got.LightWash != palette.Default || got.DarkWash != palette.Default {
		t.Fatalf("invalid or missing values: %+v", got)
	}
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path); err == nil {
		t.Fatal("malformed settings should be reported")
	}
}

func TestLoadRejectsDuplicateSettings(t *testing.T) {
	for _, contents := range []string{
		`{"zoom":100,"zoom":125}`,
		`{"zoom":100,"Zoom":125}`,
		`{"preferences":{},"Preferences":{}}`,
		`{"Preferences":{"FontSize":18,"fontSize":22}}`,
		`{"lightWash":"rex","LightWash":"tide"}`,
	} {
		t.Run(contents, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("duplicate settings should be rejected")
			}
		})
	}
}

func TestLoadMissingFileAndReadError(t *testing.T) {
	dir := t.TempDir()
	legacyDir := filepath.Join(dir, "JSONPal")
	if err := os.MkdirAll(legacyDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "editor-settings.json"), []byte(`{"fontSize":19}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(filepath.Join(dir, "MyJson", "settings.json"))
	if err != nil || got != DefaultSaved() {
		t.Fatalf("missing settings: got %+v, error %v", got, err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected read error")
	}
}

func TestEmptyThemeConfigurationDefaultsToPaper(t *testing.T) {
	for _, tc := range []struct {
		name, contents, light, dark string
	}{
		{name: "missing file", light: "paper", dark: "paper"},
		{name: "empty object", contents: `{}`, light: "paper", dark: "paper"},
		{name: "empty values", contents: `{"lightWash":"","darkWash":""}`, light: "paper", dark: "paper"},
		{name: "one selected", contents: `{"lightWash":"rex","darkWash":""}`, light: "rex", dark: "paper"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if tc.contents != "" {
				if err := os.WriteFile(path, []byte(tc.contents), 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if got.LightWash != tc.light || got.DarkWash != tc.dark {
				t.Fatalf("配色 = (%q, %q)，期望 (%q, %q)", got.LightWash, got.DarkWash, tc.light, tc.dark)
			}
		})
	}
}

func TestSaveReportsFailure(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "file")
	if err := os.WriteFile(parent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Save(filepath.Join(parent, "settings.json"), DefaultSaved()); err == nil {
		t.Fatal("expected save to report directory error")
	}
}
