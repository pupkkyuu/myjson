package app

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/internal/component"
	"myjson/internal/editor/format"
)

var nativeKeepAlive func()

func TestMain(m *testing.M) {
	if os.Getenv("MYJSON_NATIVE_SMOKE") != "1" {
		os.Exit(m.Run())
	}
	directory, err := os.MkdirTemp("", "myjson-native-")
	if err != nil {
		panic(err)
	}
	mygo.App.SetName("MyJson Verification")
	mygo.App.SetPath(mygo.PathUserData, directory)
	if runtime.GOOS == "darwin" {
		mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	}
	nativeKeepAlive = mygo.App.OnWindowAllClosed(func() {})
	result := make(chan int, 1)
	mygo.App.WhenReady(func() {
		go func() {
			result <- m.Run()
			mygo.App.Quit()
		}()
	})
	runErr := mygo.App.Run()
	code := 1
	if runErr == nil {
		code = <-result
	}
	if err := os.RemoveAll(directory); err != nil {
		runErr = errors.Join(runErr, err)
		code = 1
	}
	if runErr != nil {
		if _, err := fmt.Fprintln(os.Stderr, runErr); err != nil {
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func TestNativeLifecycle(t *testing.T) {
	if os.Getenv("MYJSON_NATIVE_SMOKE") != "1" || runtime.GOOS != "darwin" {
		t.Skip("set MYJSON_NATIVE_SMOKE=1 in a macOS desktop session")
	}
	app := newApplication()
	off := app.configureLifecycle(runtime.GOOS)
	defer off()
	nativeKeepAlive()
	defer func() { nativeKeepAlive = mygo.App.OnWindowAllClosed(func() {}) }()
	defer func() {
		mygo.RunOnMain(func() {
			if app.window != nil {
				app.window.Destroy()
			}
		})
	}()

	mygo.RunOnMain(func() {
		app.openWindow()
		e := format.NewFormatEditor(app.nextEditorID(), `{"message":"关闭窗口"}`)
		app.editors, app.active = append(app.editors, e), e
		app.settingState.Open = true
	})
	w := app.window
	w.Close()
	mygo.RunOnMain(func() {
		if !w.IsDestroyed() || app.window != nil || len(mygo.Windows()) != 0 {
			t.Error("关闭最后窗口后仍保留窗口引用")
		}
		if app.active != nil || len(app.editors) != 0 || app.settingState.Open {
			t.Error("关闭窗口后仍保留旧窗口的文档或弹窗状态")
		}
	})

	mygo.RunOnMain(func() {
		app.activate(false)
		if app.window == nil || app.window == w || app.window.IsDestroyed() || len(mygo.Windows()) != 1 {
			t.Error("无窗口时重新激活未创建唯一的新窗口")
			return
		}
		w = app.window
		app.activate(true)
		w.Hide()
		app.activate(false)
		if app.window != w || !w.IsVisible() || len(mygo.Windows()) != 1 {
			t.Error("重新激活隐藏窗口时未复用原窗口")
		}
	})
}

func TestNativeWorkspace(t *testing.T) {
	if os.Getenv("MYJSON_NATIVE_SMOKE") != "1" {
		t.Skip("set MYJSON_NATIVE_SMOKE=1 in a desktop session")
	}
	app := newApplication()
	mygo.RunOnMain(app.openWindow)
	w := app.window
	defer w.Destroy()
	if width, height := w.Size(); width != 1000 || height != 650 || w.Page() != nil {
		t.Fatalf("原生窗口的初始尺寸或内容类型错误：%dx%d", width, height)
	}
	for _, mode := range []struct {
		name     string
		theme    mygo.ThemeSource
		width    int
		editor   bool
		settings bool
	}{
		{"empty-light", mygo.ThemeLight, 1000, false, false},
		{"editor-light", mygo.ThemeLight, 1000, true, false},
		{"editor-dark", mygo.ThemeDark, 1000, true, false},
		{"settings-light", mygo.ThemeLight, 1000, true, true},
		{"settings-dark", mygo.ThemeDark, 760, true, true},
	} {
		mygo.Theme.SetSource(mode.theme)
		mygo.RunOnMain(func() {
			w.SetSize(mode.width, 650)
			app.settingState.Open = mode.settings
			if mode.editor {
				e := format.NewFormatEditor("editor-1", "{\n  \"message\": \"中文与主题\",\n  \"number\": 9007199254740993\n}")
				e.Invalidate = w.Invalidate
				app.editors = append(app.editors[:0], e)
				app.active = e
			}
			w.Invalidate()
		})
		saveNativeFrame(t, w, "native-"+mode.name+".png", mode.width, 220*time.Millisecond)
		mygo.RunOnMain(func() {
			if app.settingState.Open != mode.settings || mode.settings && app.settingState.Category != "appearance" {
				t.Errorf("%s 设置打开状态或默认分类错误：%+v", mode.name, app.settingState)
			}
			if app.theme.Dark != (mode.theme == mygo.ThemeDark) {
				t.Errorf("%s 主题未跟随原生窗口状态", mode.name)
			}
			if width, _ := w.Size(); width != mode.width {
				t.Errorf("%s 打开设置改变了窗口尺寸：%d", mode.name, width)
			}
		})
	}
	prevented := false
	off := w.OnClose(func(e *mygo.CloseEvent) { prevented = true; e.PreventDefault() })
	w.Close()
	mygo.RunOnMain(func() {
		if !prevented || w.IsDestroyed() {
			t.Error("原生关闭未保留取消能力")
		}
	})
	off()
	w.Close()
	if !w.IsDestroyed() {
		t.Fatal("原生窗口未关闭")
	}
}

func TestNativeToasts(t *testing.T) {
	if os.Getenv("MYJSON_NATIVE_SMOKE") != "1" {
		t.Skip("set MYJSON_NATIVE_SMOKE=1 in a desktop session")
	}
	app := newApplication()
	pending := false
	mygo.RunOnMain(func() {
		e := format.NewFormatEditor("editor-1", "{\n  \"message\": \"中文与提示\"\n}")
		app.editors, app.active = append(app.editors, e), e
		options := app.windowOptions(runtime.GOOS)
		options.Content = ui.View(func(c *ui.Context) {
			if pending {
				pending = false
				c.AddToast(ui.Toast{ID: "native-error", Title: "JSON 格式错误：第 2 行，第 8 列", Description: "原文保持不变，请检查 JSON 内容。", Type: component.ToastError, Timeout: -1})
			}
			app.view(c)
		})
		app.window = mygo.NewWindow(options)
	})
	w := app.window
	defer w.Destroy()
	for _, mode := range []struct {
		name  string
		theme mygo.ThemeSource
		width int
	}{{"light", mygo.ThemeLight, 1000}, {"dark", mygo.ThemeDark, 760}} {
		mygo.Theme.SetSource(mode.theme)
		mygo.RunOnMain(func() {
			pending = true
			w.SetSize(mode.width, 650)
			w.Invalidate()
		})
		saveNativeFrame(t, w, "native-toast-"+mode.name+".png", mode.width, 350*time.Millisecond)
	}
}

func saveNativeFrame(t *testing.T, w *mygo.Window, name string, width int, delay time.Duration) {
	t.Helper()
	if _, err := w.CapturePage(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(delay)
	data, err := w.CapturePage()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() < width {
		t.Fatalf("%s 原生截图错误：%v", name, err)
	}
	directory := os.Getenv("MYJSON_TEST_SCREENSHOTS")
	if directory == "" {
		directory = t.TempDir()
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), data, 0644); err != nil {
		t.Fatal(err)
	}
}
