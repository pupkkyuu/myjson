package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

func clickPreferenceControl(t *testing.T, tt *ui.Tester, role ui.Role, label string) {
	t.Helper()
	control, ok := tt.AccessByRole(role, label)
	if !ok {
		t.Fatalf("未找到设置控件 %q", label)
	}
	r := control.Bounds
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
}

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
	clickPreferenceControl(t, tt, ui.RoleSlider, locales.EditorFontSize())
	tt.Key(0, ui.KeyEnd)
	clickPreferenceControl(t, tt, ui.RolePopUpButton, locales.EditorIndentWidth())
	click(t, tt, "4")
	for _, e := range []*format.FormatEditor{first, second} {
		if e.Font.Size != 28 || e.TabSize != 4 {
			t.Fatalf("设置未应用到所有打开的文档：字号 %g，缩进 %d", e.Font.Size, e.TabSize)
		}
	}
	if first.Buffer.String() != "{}" || !first.WrapEnabled() || second.WrapEnabled() {
		t.Fatal("修改设置改变了文档内容或各自的换行状态")
	}
	clickPreferenceControl(t, tt, ui.RoleSwitch, locales.EditorDefaultWrap())
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
