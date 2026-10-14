package settings

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func applyTheme(c *ui.Context, source mygo.ThemeSource) {
	mygo.Theme.SetSource(source)
	c.Invalidate()
}
