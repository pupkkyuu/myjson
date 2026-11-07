package settings

import (
	"github.com/egoist/mygo/ui"

	"myjson/internal/editor"
)

const (
	appearanceCategory = "appearance"
	colorCategory      = "color"
	editorCategory     = "editor"
	diffCategory       = "diff"
	aboutCategory      = "about"
)

type State struct {
	Open        bool
	Category    string
	preferences *editor.Preferences
	zoomPercent int
	// lightWash 和 darkWash 是浅色、深色外观各自用的主题，空着时用默认主题。
	lightWash string
	darkWash  string
}

var zoomLevels = [...]int{75, 80, 90, 100, 110, 125, 150, 175, 200}

func (s *State) ZoomPercent() int {
	if s.zoomPercent == 0 {
		return 100
	}
	return s.zoomPercent
}

func (s *State) SetZoomPercent(value int) {
	for _, level := range zoomLevels {
		if value == level {
			s.zoomPercent = value
			return
		}
	}
}

func (s *State) StepZoom(delta int) {
	for i, level := range zoomLevels {
		if level == s.ZoomPercent() {
			s.zoomPercent = zoomLevels[max(0, min(len(zoomLevels)-1, i+delta))]
			return
		}
	}
}

func (s *State) ResetZoom() { s.zoomPercent = 100 }

// Wash 是当前外观用的主题名字。
func (s *State) Wash(dark bool) string {
	if dark {
		return s.darkWash
	}
	return s.lightWash
}

// SetWash 记下一种外观用的主题。
func (s *State) SetWash(dark bool, id string) {
	if dark {
		s.darkWash = id
		return
	}
	s.lightWash = id
}

// ZoomShortcuts 处理界面缩放快捷键；弹窗打开时须在弹窗内调用。
func (s *State) ZoomShortcuts(c *ui.Context) {
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyEqual) || c.Shortcut(ui.Cmd|ui.Shift, ui.KeyEqual):
		s.StepZoom(1)
	case c.Shortcut(ui.Cmd, ui.KeyMinus):
		s.StepZoom(-1)
	case c.Shortcut(ui.Cmd, ui.Key0):
		s.ResetZoom()
	default:
		return
	}
	c.Invalidate()
}

func (s *State) Preferences() *editor.Preferences {
	if s.preferences == nil {
		p := editor.DefaultPreferences()
		s.preferences = &p
	}
	return s.preferences
}
