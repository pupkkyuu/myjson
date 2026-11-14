package settings

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/locales"
)

const applicationRepository = "https://github.com/pupkkyuu/myjson"

func about(c *ui.Context) {
	t := c.Theme()
	for _, item := range []struct{ label, value string }{
		{locales.ApplicationVersion(), mygo.App.Version()},
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
