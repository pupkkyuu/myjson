//go:build darwin

package app

import (
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
)

// TestNativeTabShortcuts routes NSEvents through NSApplication so AppKit's
// event monitors and window dispatch both participate. It requires a desktop
// session and does not simulate a physical keyboard or system-wide shortcuts.
func TestNativeTabShortcuts(t *testing.T) {
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
	var makeEvent func(objc.ID, objc.SEL, uint, point, uint, float64, int, objc.ID, objc.ID, objc.ID, bool, uint16) objc.ID
	purego.RegisterFunc(&makeEvent, addr)
	send := func(obj objc.ID, selector string, args ...uintptr) objc.ID {
		a := append([]uintptr{uintptr(obj), uintptr(objc.RegisterName(selector))}, args...)
		r, _, _ := purego.SyscallN(addr, a...)
		return objc.ID(r)
	}
	app := newApplication()
	mygo.RunOnMain(func() {
		app.openWindow()
		app.createFormatEditor(app.window.Invalidate)
		app.createFormatEditor(app.window.Invalidate)
		win := objc.ID(app.window.NativeHandle())
		send(win, "makeKeyAndOrderFront:", 0)
		nsapp := send(objc.ID(objc.GetClass("NSApplication")), "sharedApplication")
		send(nsapp, "activateIgnoringOtherApps:", 1)
	})
	defer mygo.RunOnMain(func() {
		if app.window != nil {
			app.window.Destroy()
		}
	})
	time.Sleep(250 * time.Millisecond)

	pressWindow := func(target *mygo.Window, mods uint, key uint16) {
		t.Helper()
		mygo.RunOnMain(func() {
			win := objc.ID(target.NativeHandle())
			charsBytes := []byte{'\t', 0}
			if mods&(1<<17) != 0 {
				charsBytes[0] = 0x19 // AppKit's backtab character.
			}
			chars := send(objc.ID(objc.GetClass("NSString")), "stringWithUTF8String:", uintptr(unsafe.Pointer(&charsBytes[0])))
			runtime.KeepAlive(charsBytes)
			ignoreBytes := []byte{'\t', 0}
			ignoring := send(objc.ID(objc.GetClass("NSString")), "stringWithUTF8String:", uintptr(unsafe.Pointer(&ignoreBytes[0])))
			runtime.KeepAlive(ignoreBytes)
			ev := makeEvent(objc.ID(objc.GetClass("NSEvent")), objc.RegisterName("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"), 10, point{}, mods, 0, int(send(win, "windowNumber")), 0, chars, ignoring, false, key)
			nsapp := send(objc.ID(objc.GetClass("NSApplication")), "sharedApplication")
			send(nsapp, "sendEvent:", uintptr(ev))
		})
		time.Sleep(120 * time.Millisecond)
	}
	press := func(mods uint, key uint16) { pressWindow(app.window, mods, key) }
	check := func(want string) {
		t.Helper()
		mygo.RunOnMain(func() {
			if got := activeID(app); got != want {
				t.Errorf("活动标签 = %s，期望 %s", got, want)
			}
		})
	}
	const ctrl, shift, alt = 1 << 18, 1 << 17, 1 << 19
	check("editor-3")
	press(ctrl, 48)
	check("editor-1") // forward wraps and executes exactly once
	press(ctrl, 48)
	check("editor-2")
	press(ctrl|shift, 48)
	check("editor-1")
	press(ctrl|shift, 48)
	check("editor-3") // backward wraps
	press(ctrl|alt, 48)
	check("editor-3") // unrelated modifiers are left alone
	mygo.RunOnMain(func() {
		app.settingState.Open = true
		app.window.Invalidate()
	})
	time.Sleep(120 * time.Millisecond)
	press(ctrl, 48)
	press(ctrl|shift, 48)
	check("editor-3")
	mygo.RunOnMain(func() {
		app.settingState.Open = false
		app.window.Invalidate()
	})
	press(ctrl, 48)
	check("editor-1")

	// A different frontmost window must not drive the document workspace.
	var other *mygo.Window
	mygo.RunOnMain(func() {
		other = mygo.NewWindow(app.windowOptions("darwin"))
		send(objc.ID(other.NativeHandle()), "makeKeyAndOrderFront:", 0)
	})
	time.Sleep(120 * time.Millisecond)
	pressWindow(other, ctrl, 48)
	check("editor-1")
	mygo.RunOnMain(func() {
		other.Destroy()
		app.window.Destroy()
		app.openWindow()
		app.createFormatEditor(app.window.Invalidate)
		app.createFormatEditor(app.window.Invalidate)
		send(objc.ID(app.window.NativeHandle()), "makeKeyAndOrderFront:", 0)
	})
	time.Sleep(250 * time.Millisecond)
	check("editor-6")
	press(ctrl, 48)
	check("editor-4") // reopened window has one working monitor
}
