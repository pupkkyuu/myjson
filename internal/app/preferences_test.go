package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func TestSettingsApplyToOpenAndNewDocuments(t *testing.T) {
	app := newApplication()
	tt := mountWorkspace(app)
	click(t, tt, locales.NewFormat())
	first := app.active.(*format.FormatEditor)
	click(t, tt, locales.NewFormat())
	second := app.active.(*format.FormatEditor)
	first.Buffer.Set("{}")
	first.ToggleWrap()
	tt.Key(ui.Cmd, ui.KeyComma)
	click(t, tt, locales.EditorSettings())
	click(t, tt, locales.SettingControl(locales.EditorFontSize()))
	tt.Key(0, ui.KeyEnd)
	click(t, tt, locales.SettingControl(locales.EditorIndentWidth()))
	click(t, tt, "4")
	for _, e := range []*format.FormatEditor{first, second} {
		if e.Font.Size != 28 || e.TabSize != 4 {
			t.Fatalf("设置未应用到所有打开的文档：字号 %g，缩进 %d", e.Font.Size, e.TabSize)
		}
	}
	if first.Buffer.String() != "{}" || !first.WrapEnabled() || second.WrapEnabled() {
		t.Fatal("修改设置改变了文档内容或各自的换行状态")
	}
	click(t, tt, locales.SettingControl(locales.EditorDefaultWrap()))
	tt.Key(0, ui.KeyEscape)
	tt.Key(ui.Cmd, ui.KeyT)
	created := app.active.(*format.FormatEditor)
	if created.Font.Size != 28 || created.TabSize != 4 || !created.WrapEnabled() || second.WrapEnabled() {
		t.Fatal("新文档未继承会话设置，或默认换行影响了已有文档")
	}
	if newApplication().settingState.Preferences().FontSize != 13 {
		t.Fatal("新应用实例不应继承上一会话的设置")
	}
}
