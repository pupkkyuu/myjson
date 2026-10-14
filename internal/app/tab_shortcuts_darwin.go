package app

import "github.com/ebitengine/purego/objc"

// installWindowTabShortcuts handles Ctrl+Tab before AppKit consumes it for
// key-view navigation. The monitor and its callback live on the main thread.
func (app *Application) installWindowTabShortcuts() func() {
	window := app.window
	if window == nil {
		return func() {}
	}
	native := objc.ID(window.NativeHandle())
	if native == 0 {
		return func() {}
	}
	eventClass := objc.ID(objc.GetClass("NSEvent"))
	application := objc.Send[objc.ID](objc.ID(objc.GetClass("NSApplication")), objc.RegisterName("sharedApplication"))
	block := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		if app.window != window || window.IsDestroyed() || app.settingState.Open ||
			objc.Send[objc.ID](event, objc.RegisterName("window")) != native ||
			!objc.Send[bool](native, objc.RegisterName("isKeyWindow")) ||
			objc.Send[objc.ID](native, objc.RegisterName("attachedSheet")) != 0 ||
			objc.Send[objc.ID](application, objc.RegisterName("modalWindow")) != 0 {
			return event
		}
		const (
			keyTab  = 0x30
			shift   = 1 << 17
			control = 1 << 18
			option  = 1 << 19
			command = 1 << 20
			modMask = shift | control | option | command
		)
		if objc.Send[uint16](event, objc.RegisterName("keyCode")) != keyTab {
			return event
		}
		mods := objc.Send[uint](event, objc.RegisterName("modifierFlags")) & modMask
		direction := 0
		switch mods {
		case control:
			direction = 1
		case control | shift:
			direction = -1
		default:
			return event
		}
		app.cycleEditor(direction)
		window.Invalidate()
		return 0
	})
	monitor := objc.Send[objc.ID](eventClass, objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"), uint(1<<10), block)
	if monitor == 0 {
		block.Release()
		return func() {}
	}
	objc.Send[objc.ID](monitor, objc.RegisterName("retain"))
	return func() {
		objc.Send[objc.ID](eventClass, objc.RegisterName("removeMonitor:"), monitor)
		objc.Send[objc.ID](monitor, objc.RegisterName("release"))
		block.Release()
	}
}
