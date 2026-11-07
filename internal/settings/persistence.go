package settings

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/egoist/mygo"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
)

// Saved holds the preferences that survive application restarts.
type Saved struct {
	Preferences editor.Preferences `json:"preferences"`
	Zoom        int                `json:"zoom"`
	Theme       mygo.ThemeSource   `json:"theme"`
	// LightWash 和 DarkWash 是浅色、深色外观各自用的主题名字。
	LightWash string `json:"lightWash"`
	DarkWash  string `json:"darkWash"`
}

// DefaultSaved returns the default persistent settings.
func DefaultSaved() Saved {
	return Saved{Preferences: editor.DefaultPreferences(), Zoom: 100, Theme: mygo.ThemeSystem, LightWash: defaultWash, DarkWash: defaultWash}
}

// Snapshot returns the current settings, including the supplied theme.
func (s *State) Snapshot(theme mygo.ThemeSource) Saved {
	return Saved{Preferences: *s.Preferences(), Zoom: s.ZoomPercent(), Theme: theme, LightWash: s.lightWash, DarkWash: s.darkWash}
}

// Restore applies saved editor preferences, zoom and themes to the state.
func (s *State) Restore(saved Saved) {
	p := saved.Preferences
	s.preferences = &p
	s.SetZoomPercent(saved.Zoom)
	s.lightWash, s.darkWash = saved.LightWash, saved.DarkWash
}

// ConfigPath returns the application configuration path.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user configuration directory: %w", err)
	}
	return filepath.Join(dir, "MyJson", "settings.json"), nil
}

// Load reads settings, returning defaults when the file does not exist.
func Load(path string) (Saved, error) {
	saved := DefaultSaved()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return saved, nil
	}
	if err != nil {
		return saved, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return DefaultSaved(), fmt.Errorf("parse settings: %w", err)
	}
	return validateSaved(saved), nil
}

func validateSaved(s Saved) Saved {
	d := DefaultSaved()
	p := &s.Preferences
	if p.FontSize < 10 || p.FontSize > 28 {
		p.FontSize = d.Preferences.FontSize
	}
	if p.LineHeight != 1.3 && p.LineHeight != 1.5 && p.LineHeight != 1.8 {
		p.LineHeight = d.Preferences.LineHeight
	}
	if p.TabSize != 2 && p.TabSize != 4 && p.TabSize != 8 {
		p.TabSize = d.Preferences.TabSize
	}
	if p.Whitespace > mygoeditor.WhitespaceBoundary {
		p.Whitespace = d.Preferences.Whitespace
	}
	if p.DiffLayout > mygoeditor.DiffUnified {
		p.DiffLayout = d.Preferences.DiffLayout
	}
	if p.DiffInline != mygoeditor.DiffInlineChar && p.DiffInline != mygoeditor.DiffInlineWord && p.DiffInline != mygoeditor.DiffInlineNone {
		p.DiffInline = d.Preferences.DiffInline
	}
	validZoom := false
	for _, z := range zoomLevels {
		if s.Zoom == z {
			validZoom = true
			break
		}
	}
	if !validZoom {
		s.Zoom = d.Zoom
	}
	if s.Theme != mygo.ThemeSystem && s.Theme != mygo.ThemeLight && s.Theme != mygo.ThemeDark {
		s.Theme = d.Theme
	}
	if !knownWash(s.LightWash) {
		s.LightWash = d.LightWash
	}
	if !knownWash(s.DarkWash) {
		s.DarkWash = d.DarkWash
	}
	return s
}

func knownWash(id string) bool {
	return slices.Contains(washIDs, id)
}

// Save writes a temporary file and replaces the configuration after closing it.
func Save(path string, saved Saved) (err error) {
	data, err := json.Marshal(saved, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary settings: %w", err)
	}
	name := f.Name()
	closed, renamed := false, false
	defer func() {
		if !closed {
			if closeErr := f.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close temporary settings: %w", closeErr))
			}
		}
		if !renamed {
			if removeErr := os.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove temporary settings: %w", removeErr))
			}
		}
	}()
	data = append(data, '\n')
	n, writeErr := f.Write(data)
	if writeErr != nil {
		return fmt.Errorf("write temporary settings: %w", writeErr)
	}
	if n != len(data) {
		return fmt.Errorf("write temporary settings: %w", io.ErrShortWrite)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync temporary settings: %w", err)
	}
	closeErr := f.Close()
	closed = true
	if closeErr != nil {
		return fmt.Errorf("close temporary settings: %w", closeErr)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	renamed = true
	return nil
}
