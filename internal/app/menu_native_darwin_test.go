//go:build darwin

package app

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"

	"myjson/internal/editor/diff"
	"myjson/internal/editor/format"
)

func TestNativeMenuKeyEquivalents(t *testing.T) {
	if os.Getenv("MYJSON_NATIVE_SMOKE") != "1" {
		t.Skip("set MYJSON_NATIVE_SMOKE=1 in a macOS desktop session")
	}
	lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := purego.Dlsym(lib, "objc_msgSend")
	if err != nil {
		t.Fatal(err)
	}
	type point struct{ X, Y float64 }
	var event func(objc.ID, objc.SEL, uint, point, uint, float64, int, objc.ID, objc.ID, objc.ID, bool, uint16) objc.ID
	purego.RegisterFunc(&event, addr)
	send := func(obj objc.ID, selector string, args ...uintptr) objc.ID {
		a := append([]uintptr{uintptr(obj), uintptr(objc.RegisterName(selector))}, args...)
		r, _, _ := purego.SyscallN(addr, a...)
		return objc.ID(r)
	}
	press := func(key string) bool {
		text := append([]byte(key), 0)
		chars := send(objc.ID(objc.GetClass("NSString")), "stringWithUTF8String:", uintptr(unsafe.Pointer(&text[0])))
		runtime.KeepAlive(text)
		ev := event(objc.ID(objc.GetClass("NSEvent")), objc.RegisterName("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"), 10, point{}, 1<<20, 0, 0, 0, chars, chars, false, 0)
		nsapp := send(objc.ID(objc.GetClass("NSApplication")), "sharedApplication")
		return send(send(nsapp, "mainMenu"), "performKeyEquivalent:", uintptr(ev)) != 0
	}
	hasFileMenu := func() bool {
		text := append([]byte("文件"), 0)
		label := send(objc.ID(objc.GetClass("NSString")), "stringWithUTF8String:", uintptr(unsafe.Pointer(&text[0])))
		runtime.KeepAlive(text)
		nsapp := send(objc.ID(objc.GetClass("NSApplication")), "sharedApplication")
		index := send(send(nsapp, "mainMenu"), "indexOfItemWithTitle:", uintptr(label))
		return int64(index) >= 0
	}
	app := newApplication()
	mygo.RunOnMain(func() {
		app.menu = app.applicationMenu()
		mygo.App.SetMenu(app.menu)
		app.openWindow()
		if len(app.editors) != 1 || activeID(app) != "editor-1" {
			t.Error("新窗口应默认打开一个格式化标签")
		} else if _, ok := app.active.(*format.FormatEditor); !ok {
			t.Error("新窗口的默认标签应为格式化标签")
		}
		if !hasFileMenu() {
			t.Error("native menu is missing the Chinese file menu")
		}
	})
	defer mygo.RunOnMain(func() {
		if app.window != nil {
			app.window.Destroy()
		}
		mygo.App.SetMenu(nil)
	})
	mygo.RunOnMain(func() {
		for _, tc := range []struct {
			key  string
			zoom int
		}{{"=", 110}, {"-", 100}, {"0", 100}} {
			if !press(tc.key) {
				t.Errorf("menu did not handle Cmd+%s", tc.key)
			}
			if got := app.settingState.ZoomPercent(); got != tc.zoom {
				t.Errorf("Cmd+%s: zoom %d, want %d", tc.key, got, tc.zoom)
			}
		}
		if !press("t") || len(app.editors) != 2 {
			t.Errorf("Cmd+T should create exactly one editor; got %d", len(app.editors))
		}
		if !press("w") || app.window == nil || len(app.editors) != 1 {
			t.Error("Cmd+W should close the active tab and keep the window open")
		}
		if !press("w") || app.window != nil {
			t.Error("Cmd+W on the last tab should close the window")
		}
		if !press("t") || app.window == nil || len(app.editors) != 1 {
			t.Error("Cmd+T should reopen a window with one editor")
		}
		app.settingState.Open = true
		app.syncMenu()
		if app.menu.ItemByID("new-format").IsEnabled() {
			t.Error("new document enabled while settings are open")
		}
		app.settingState.Open = false
		app.window.Destroy()
		if app.window != nil || !app.menu.ItemByID("new-format").IsEnabled() {
			t.Error("new document should remain available after closing the window")
		} else if !press("t") || app.window == nil || len(app.editors) != 1 {
			t.Error("new document should reopen a window with exactly one editor")
		}
		if app.window != nil {
			app.window.Destroy()
			app.menuNewEditor(true)
			if len(app.editors) != 2 {
				t.Error("new comparison in a new window should retain its default format editor")
			} else if _, ok := app.active.(*diff.DiffEditor); !ok {
				t.Error("the newly created comparison should be active")
			}
		}
	})
}
