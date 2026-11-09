package settings

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

const (
	applicationVersion    = "0.2.1"
	applicationAuthor     = "pupkkyuu"
	applicationRepository = "https://github.com/pupkkyuu/myjson"
)

func about(c *ui.Context) {
	t := c.Theme()
	version := mygo.App.Version()
	if version == "" {
		version = applicationVersion
	}
	for _, item := range []struct{ label, value string }{
		{locales.ApplicationVersion(), version},
		{locales.ApplicationAuthor(), applicationAuthor},
		{locales.ApplicationRepository(), applicationRepository},
	} {
		ui.Column(c.Key(item.label)).FillWidth().MinWidth(0).Gap(t.Space(1)).Children(func() {
			ui.Text(c, item.label).TextColor(t.TextMuted).FontSize(t.Rem(.875)).Wrap()
			if item.value == applicationRepository {
				ui.Link(c, item.value, applicationRepository).Wrap().MinWidth(0)
			} else {
				ui.Text(c, item.value).Wrap().Selectable()
			}
		})
	}
}
