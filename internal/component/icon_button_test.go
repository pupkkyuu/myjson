package component_test

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/pkg/icons"
)

func TestIconButtonInputDisabledAndKeepFocus(t *testing.T) {
	disabled, clicks := false, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Button(c, "Focus target")
			b := component.IconButton(c.Key("copy"), icons.Copy(), "Copy document").Disabled(disabled).KeepFocus()
			if b.Clicked() {
				clicks++
			}
		})
	}, 300, 200)
	if err := tt.Click("Focus target"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Copy document"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 || !tt.Focused("Focus target") {
		t.Fatal("图标按钮未派发点击或抢走了编辑焦点")
	}
	disabled = true
	tt.Frame()
	if err := tt.Click("Copy document"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatal("禁用图标按钮仍派发点击")
	}
	disabled = false
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	count := 0
	for _, text := range tt.Texts() {
		if text == "Copy document" {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("图标按钮提示丢失：%v", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if clicks != 2 {
		t.Fatal("图标按钮键盘访问失效")
	}
}
