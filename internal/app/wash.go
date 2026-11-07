package app

import (
	"github.com/egoist/mygo/ui"

	"myjson/internal/palette"
)

const defaultWash = palette.Default

type look struct {
	base, glow ui.LinearGradient
	theme      *ui.Theme
	plain      bool
	sidebar    ui.Color
}

func lookFor(id string, dark bool) look {
	p := palette.LookFor(id, dark)
	return look{base: p.Base, glow: p.Glow, theme: p.Theme, plain: p.Plain, sidebar: p.Sidebar}
}
