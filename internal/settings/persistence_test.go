package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	mygoeditor "github.com/egoist/mygo/ui/editor"
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
	if got.Preferences.FontSize != 13 || got.Preferences.TabSize != 4 || !got.Preferences.AutoClose || got.Zoom != 100 || got.Theme != mygo.ThemeSystem || got.LightWash != defaultWash || got.DarkWash != defaultWash {
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
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"zoom":100,"zoom":125}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("duplicate settings should be rejected")
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
