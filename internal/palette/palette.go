package palette

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// tone 是一套主题的色相：卡片和文字染多少这个色相，语法色有多鲜艳，
// 以及强调色（光标、选区、控件）用的色相。
type tone struct{ hue, tint, vivid, accent float32 }

// wash 是窗口底色的一套渐变：从上到下的色标，加上右上角散开的光晕。
type wash struct {
	id                  string
	light, dark         []string
	lightGlow, darkGlow ui.Color
	tone                tone
	plain               bool
}

// washes 是 kero 的十套主题，顺序即设置里的顺序。plain 没有渐变，
// 底色就是侧栏自己的颜色。
var washes = []wash{
	{
		id: "rex", light: []string{"#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf"}, lightGlow: ui.RGBA(247, 220, 241, 0.75),
		dark: []string{"#240f1d", "#26111a", "#271610"}, darkGlow: ui.RGBA(142, 58, 110, 0.3), tone: tone{350, 1, 0.95, 350},
	},
	{
		id: "mist", light: []string{"#e3e9f4", "#ebeff6", "#f1f2f3", "#f2f1ec"}, lightGlow: ui.RGBA(210, 225, 248, 0.75),
		dark: []string{"#0c1628", "#0d1827", "#111b23"}, darkGlow: ui.RGBA(46, 91, 166, 0.3), tone: tone{255, 1, 0.8, 255},
	},
	{
		id: "meadow", light: []string{"#e2eee5", "#ebf2e7", "#f3f2e2", "#f5f0d6"}, lightGlow: ui.RGBA(214, 240, 222, 0.75),
		dark: []string{"#081c0d", "#101d0c", "#1b1c0b"}, darkGlow: ui.RGBA(6, 113, 50, 0.3), tone: tone{140, 1, 0.9, 140},
	},
	{
		id: "dusk", light: []string{"#e4dbf3", "#efdeee", "#f6e3dc", "#f7e6cf"}, lightGlow: ui.RGBA(246, 206, 222, 0.8),
		dark: []string{"#190f2e", "#260f24", "#2d140b"}, darkGlow: ui.RGBA(125, 65, 138, 0.3), tone: tone{315, 1, 1.1, 315},
	},
	{
		id: "tide", light: []string{"#dbebef", "#e3eff1", "#ecf3f0", "#f2f2e8"}, lightGlow: ui.RGBA(194, 232, 238, 0.75),
		dark: []string{"#001b22", "#001e1f", "#0b1e1b"}, darkGlow: ui.RGBA(0, 112, 128, 0.3), tone: tone{205, 1, 0.85, 205},
	},
	{
		id: "apricot", light: []string{"#f5e1d6", "#f8e7dc", "#f8eedf", "#f6f0da"}, lightGlow: ui.RGBA(252, 212, 194, 0.78),
		dark: []string{"#2a1207", "#291507", "#25190a"}, darkGlow: ui.RGBA(151, 62, 22, 0.3), tone: tone{50, 1, 1.05, 50},
	},
	{
		id: "aurora", light: []string{"#dbefe8", "#e1ebf2", "#e8e4f4", "#f0e6f1"}, lightGlow: ui.RGBA(198, 238, 224, 0.75),
		dark: []string{"#001f13", "#06192b", "#1e122a"}, darkGlow: ui.RGBA(0, 115, 75, 0.3), tone: tone{170, 1, 1.15, 170},
	},
	{
		id: "dune", light: []string{"#ede4d8", "#f1e9de", "#f4eee4", "#f3efe8"}, lightGlow: ui.RGBA(242, 222, 194, 0.7),
		dark: []string{"#22190b", "#211a0e", "#201b12"}, darkGlow: ui.RGBA(135, 79, 0, 0.3), tone: tone{80, 0.7, 0.55, 80},
	},
	{
		id: "paper", light: []string{"#f6f6f7", "#f7f7f8", "#f8f8f8"}, lightGlow: ui.RGBA(255, 255, 255, 0),
		dark: []string{"#141416", "#151517", "#161618"}, darkGlow: ui.RGBA(90, 120, 200, 0.1), tone: tone{260, 0, 0.9, 262},
	},
	{
		id: "plain", light: []string{"#eceef4", "#eceef4"}, lightGlow: ui.RGBA(0, 0, 0, 0),
		dark: []string{"#141418", "#141418"}, darkGlow: ui.RGBA(0, 0, 0, 0), plain: true,
	},
}

// Default is the palette used when none has been selected.
const Default = "paper"

// IDs returns the available palettes in display order.
func IDs() []string {
	ids := make([]string, len(washes))
	for i, w := range washes {
		ids[i] = w.id
	}
	return ids
}

// Look contains the window gradients and UI colors for a palette and appearance.
// Theme and gradient Stops are shared and must not be modified; copy them first.
type Look struct {
	Base, Glow ui.LinearGradient
	Theme      *ui.Theme
	Plain      bool
	Sidebar    ui.Color
}

type lookKey struct {
	id   string
	dark bool
}

// The palette definitions are static, so build both appearances once.
var looks = func() map[lookKey]Look {
	all := make(map[lookKey]Look, 2*len(washes))
	for _, w := range washes {
		for _, dark := range []bool{false, true} {
			all[lookKey{w.id, dark}] = buildLook(w, dark)
		}
	}
	return all
}()

// LookFor resolves a palette for the requested appearance, falling back to Default.
// The returned Theme and gradient Stops are read-only shared data.
func LookFor(id string, dark bool) Look {
	if l, ok := looks[lookKey{id, dark}]; ok {
		return l
	}
	return looks[lookKey{Default, dark}]
}

func buildLook(w wash, dark bool) Look {
	hexes, glow := w.light, w.lightGlow
	if dark {
		hexes, glow = w.dark, w.darkGlow
	}
	stops := make([]ui.GradientStop, len(hexes))
	for i, h := range hexes {
		stops[i] = ui.GradientStop{Color: ui.Hex(h), At: float32(i) / float32(len(hexes)-1)}
	}
	l := Look{
		Base:    ui.LinearGradient{Angle: 180, Oklab: true, Stops: stops},
		Glow:    glowOf(glow),
		Plain:   w.plain,
		Sidebar: ui.Hex(hexes[0]),
	}
	if w.plain {
		l.Theme = plainTheme(dark)
	} else {
		l.Theme = tintedTheme(w.tone, dark)
	}
	return l
}

// glowOf 是从右上角散开的光晕：225°，到一半就淡成透明。
func glowOf(c ui.Color) ui.LinearGradient {
	return ui.LinearGradient{Angle: 225, Oklab: true, Stops: []ui.GradientStop{{Color: c, At: 0}, {Color: c.Alpha(0), At: 0.5}}}
}

// plainTheme 是纯色主题的界面色，不随色相染色。
func plainTheme(dark bool) *ui.Theme {
	if dark {
		t := ui.DarkTheme()
		t.Dark = true
		accent := ui.Hex("#6b93ff")
		t.Background = ui.Hex("#141418")
		t.Surface = ui.Hex("#232328")
		t.Text = ui.Hex("#ececf0")
		t.TextMuted = ui.Hex("#a9a9b4")
		t.Accent = accent
		t.AccentText = ui.Hex("#161618")
		t.Danger = ui.Hex("#ff6961")
		t.Warning = ui.Hex("#f0a33a")
		t.Success = ui.Hex("#4fd17a")
		t.Selection = ui.Hex("#2f4170")
		applyControlColors(t, dark)
		return t
	}
	t := ui.LightTheme()
	accent := ui.Hex("#3b6cf6")
	t.Background = ui.Hex("#eceef4")
	t.Surface = ui.Hex("#ffffff")
	t.Text = ui.Hex("#1d1d1f")
	t.TextMuted = ui.Hex("#6e6e78")
	t.Accent = accent
	t.AccentText = ui.Hex("#ffffff")
	t.Danger = ui.Hex("#d93a35")
	t.Warning = ui.Hex("#c27b0f")
	t.Success = ui.Hex("#22863a")
	t.Selection = ui.Hex("#cfdcfb")
	applyControlColors(t, dark)
	return t
}

// tintedTheme 用主题的色相给卡片、文字和强调色染色。
func tintedTheme(tn tone, dark bool) *ui.Theme {
	tint := func(l, c float32) ui.Color { return ui.Oklch(l, c*tn.tint, tn.hue) }
	if dark {
		t := ui.DarkTheme()
		t.Dark = true
		bg, fg := tint(0.255, 0.02), tint(0.92, 0.015)
		accent := ui.Oklch(0.74, 0.13, tn.accent)
		t.Background, t.Surface = bg, bg
		t.Text, t.TextMuted = fg, fg.Mix(bg, 0.38)
		t.Accent = accent
		t.AccentText = bg
		t.Danger, t.Warning, t.Success = toneStatus(tn, true)
		t.Selection = ui.Oklch(0.38, 0.06, tn.accent)
		applyControlColors(t, dark)
		return t
	}
	t := ui.LightTheme()
	// 染色越少纸越白：不染色的主题，卡片是白的。
	paper := 0.978 + 0.022*max(0, 1-tn.tint/0.5)
	bg, fg := tint(paper, 0.012), tint(0.28, 0.03)
	accent := ui.Oklch(0.55, 0.13, tn.accent)
	t.Background, t.Surface = bg, bg
	t.Text, t.TextMuted = fg, fg.Mix(bg, 0.38)
	t.Accent = accent
	t.AccentText = ui.Hex("#ffffff")
	t.Danger, t.Warning, t.Success = toneStatus(tn, false)
	t.Selection = ui.Oklch(0.9, 0.045, tn.accent)
	applyControlColors(t, dark)
	return t
}

// applyControlColors 按明暗外观设置控件状态，使用主题已确定的文字色和强调色。
func applyControlColors(t *ui.Theme, dark bool) {
	if dark {
		t.SurfaceHover = ui.RGBA(255, 255, 255, 0.05)
		t.SurfacePressed = ui.RGBA(255, 255, 255, 0.10)
		t.Border = ui.RGBA(255, 255, 255, 0.08)
		t.AccentHover = t.Accent.Mix(ui.RGB(255, 255, 255), 0.12)
		t.AccentPressed = t.Accent.Mix(ui.RGB(0, 0, 0), 0.12)
		t.Focus = t.Accent.Alpha(0.8)
		t.Scrollbar = t.Text.Alpha(.35)
	} else {
		t.SurfaceHover = ui.RGBA(0, 0, 0, 0.04)
		t.SurfacePressed = ui.RGBA(0, 0, 0, 0.08)
		t.Border = ui.RGBA(0, 0, 0, 0.07)
		t.AccentHover = t.Accent.Mix(ui.RGB(0, 0, 0), 0.12)
		t.AccentPressed = t.Accent.Mix(ui.RGB(0, 0, 0), 0.25)
		t.Focus = t.Accent.Alpha(0.55)
		t.Scrollbar = t.Text.Alpha(.30)
	}
	t.Radius = 8
}

// toneStatus 是红、黄、绿三种状态色，按主题色相偏一点，离得远的偏得少，避免搅成浑色。
func toneStatus(tn tone, dark bool) (danger, warning, success ui.Color) {
	hues := [3]float32{28, 85, 145}
	ls := [3]float32{0.56, 0.62, 0.56}
	cs := [3]float32{0.15, 0.12, 0.13}
	if dark {
		ls = [3]float32{0.70, 0.80, 0.74}
		cs = [3]float32{0.14, 0.12, 0.13}
	}
	out := [3]ui.Color{}
	for i := range 3 {
		d := float32(math.Remainder(float64(tn.hue-hues[i]), 360))
		h := hues[i] + d*0.35*(1-abs32(d)/180)
		c := cs[i] * tn.vivid * (1 + 0.45*max(0, 1-abs32(d)/60))
		out[i] = ui.Oklch(ls[i], c, h)
	}
	return out[0], out[1], out[2]
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
