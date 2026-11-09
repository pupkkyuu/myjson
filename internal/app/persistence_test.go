package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/format"
	"myjson/internal/locales"
	"myjson/internal/settings"
)

func TestSettingsApplyImmediatelyAndSaveAfterDebounce(t *testing.T) {
	app := newApplication()
	app.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	app.lastSettings = app.settingState.Snapshot(mygo.Theme.Source())
	if err := settings.Save(app.settingsPath, app.lastSettings); err != nil {
		t.Fatal(err)
	}

	tt := mountWorkspace(app)
	click(t, tt, locales.NewFormat())
	ed := app.active.(*format.FormatEditor)
	tt.Key(ui.Cmd, ui.KeyComma)
	click(t, tt, locales.EditorSettings())
	clickPreferenceControl(t, tt, ui.RoleSlider, locales.EditorFontSize())
	tt.Key(0, ui.KeyEnd)
	if ed.Font.Size != 28 {
		t.Fatalf("font size not applied immediately: got %g", ed.Font.Size)
	}
	assertSavedFontSize := func(want float32) {
		t.Helper()
		got, err := settings.Load(app.settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		if got.Preferences.FontSize != want {
			t.Fatalf("saved font size = %g, want %g", got.Preferences.FontSize, want)
		}
	}
	assertSavedFontSize(13)
	tt.Advance(200 * time.Millisecond)
	tt.Key(0, ui.KeyHome)
	if ed.Font.Size != 10 {
		t.Fatalf("second font size not applied immediately: got %g", ed.Font.Size)
	}
	tt.Advance(299 * time.Millisecond)
	assertSavedFontSize(13)
	tt.Advance(time.Millisecond)
	assertSavedFontSize(10)

	tt.Key(0, ui.KeyEnd)
	if ed.Font.Size != 28 {
		t.Fatalf("font size before flush = %g, want 28", ed.Font.Size)
	}
	assertSavedFontSize(10)
	app.flushSettings()
	assertSavedFontSize(28)
}

func TestFlushSettingsFailureAndRetry(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "blocked")
	if err := os.WriteFile(parent, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	app := newApplication()
	app.settingsPath = filepath.Join(parent, "settings.json")
	app.lastSettings = app.settingState.Snapshot(mygo.Theme.Source())
	app.settingState.Preferences().FontSize = 21
	app.flushSettings()
	if app.settingsError == nil {
		t.Fatal("expected save failure")
	}
	if !app.settingsPending || !app.settingsBlocked {
		t.Fatal("failed write must remain pending without retrying each frame")
	}
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	app.settingsError = nil
	app.flushSettings()
	if app.settingsError != nil {
		t.Fatal(app.settingsError)
	}
	got, err := settings.Load(app.settingsPath)
	if err != nil || got.Preferences.FontSize != 21 {
		t.Fatalf("retry: %+v, %v", got, err)
	}
}
