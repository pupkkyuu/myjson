package app

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"myjson/assets"
	"myjson/internal/component"
	"myjson/internal/editor"
	"myjson/internal/locales"
	"myjson/internal/palette"
	"myjson/internal/settings"
	"myjson/pkg/icons/lucide"
)

var emptyBackground = func() *ui.Bitmap {
	bitmap, err := ui.DecodeBitmap(assets.Background)
	if err != nil {
		panic(err)
	}
	return bitmap
}()

const (
	minSidebarWidth     float32 = 180
	defaultSidebarWidth         = minSidebarWidth
	minEditorWidth      float32 = 480
	// cardGap 是卡片与窗口边缘、侧栏之间留出的缝，缝里露出整窗的底色。
	cardGap float32 = 8
	// trafficMid 是红绿灯的中心线：按钮 16 DIP 高，顶边在窗口下 20 DIP 处。
	// 侧栏开关也以这条线为中心，才和红绿灯对齐。
	trafficMid float32 = 28
)

func headerHeight(c *ui.Context) float32 {
	button := max(32, c.Theme().Rem(2))
	return max(c.TitleBar().Height, button+2*cardGap)
}

func (app *Application) view(c *ui.Context) { app.frame(c, runtime.GOOS) }

// frame 更新主题、快捷键和工作区，并将设置变化同步到编辑器。
func (app *Application) frame(c *ui.Context, platform string) {
	app.settingState.ZoomShortcuts(c)
	c.SetZoom(float32(app.settingState.ZoomPercent()) / 100)
	app.theme = resolveTheme(c, app.settingState.Wash(c.Theme().Dark))
	c.SetTheme(app.theme)
	app.shortcuts(c)
	app.workspace(c, platform)
	before := *app.settingState.Preferences()
	settings.View(c, &app.settingState, headerHeight(c))
	if after := *app.settingState.Preferences(); after != before {
		for _, e := range app.editors {
			if configurable, ok := e.(interface{ ApplyPreferences(editor.Preferences) }); ok {
				configurable.ApplyPreferences(after)
			}
		}
		c.Invalidate()
	}
	app.scheduleSettings(c)
	if app.settingsError != nil {
		c.AddToast(ui.Toast{Title: app.settingsError.Error(), Type: component.ToastError, Timeout: -1})
		app.settingsError = nil
	}
	app.syncMenu()
	component.Toasts(c, component.ToastOptions{Inset: headerHeight(c)})
}

func (app *Application) shortcuts(c *ui.Context) {
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		app.settingState.Open = true
	}
	if app.settingState.Open {
		return
	}
	if c.Shortcut(ui.Cmd, ui.KeyB) {
		app.sidebarOpen = !app.sidebarOpen
		c.Invalidate()
	}
	if c.Shortcut(ui.Cmd, ui.KeyT) {
		app.addFormatEditor(c)
		c.Invalidate()
	}
}

// backdrop 画整窗的底色：纯色主题是一块侧栏色，其余是从上到下的渐变，
// 再从右上角叠一层光晕。开了材质时渐变只留一点透出去，让文字不跟着壁纸变亮或变暗。
func (app *Application) backdrop(c *ui.Context, row ui.Element) {
	l := palette.LookFor(app.settingState.Wash(c.Theme().Dark), c.Theme().Dark)
	if l.Plain {
		bg := l.Sidebar
		if c.Vibrancy() {
			bg = bg.Alpha(0.82)
		}
		c.Root().Background(bg)
		return
	}
	base := l.Base
	if c.Vibrancy() {
		stops := make([]ui.GradientStop, len(base.Stops))
		for i, st := range base.Stops {
			stops[i] = ui.GradientStop{Color: st.Color.Alpha(0.95), At: st.At}
		}
		base.Stops = stops
	}
	c.Root().LinearGradient(base)
	row.LinearGradient(l.Glow)
}

// workspace 渲染工作区和侧栏；侧栏动画期间保持面板层级以保留编辑器状态。
// 整窗只有一块底色：侧栏直接坐在这层底上，工具栏和编辑器合在同一张卡片里。
func (app *Application) workspace(c *ui.Context, platform string) {
	root := ui.Column(c).Fill().Clip()
	app.backdrop(c, root)
	target := float32(0)
	if app.sidebarOpen {
		target = 1
	}
	progress := root.AnimateTo("sidebar", target, ui.Animation{Spring: ui.SpringOptions{Duration: 200 * time.Millisecond}})
	w, _ := c.Size()
	// 窗口变窄时侧栏让出编辑区，记住的宽度不变。
	width := min(max(app.sidebarWidth, minSidebarWidth), max(0, w-minEditorWidth-cardGap))
	root.Children(func() {
		row := ui.Row(c.Key("workspace")).Fill().AlignItems(ui.Stretch)
		row.Children(func() {
			pane := ui.Box(c).Width(width * progress).Shrink(0).Clip()
			if progress == 0 {
				pane.Invisible()
			}
			pane.Children(func() {
				ui.Column(c.Key("navigation")).Width(width).FillHeight().
					Disabled(!app.sidebarOpen || app.settingState.Open).Children(func() {
					ui.Column(c).FillWidth().Grow(1).MinHeight(0).Children(func() { app.sidebar(c) })
					ui.Toolbar(c, func() { app.sidebarFooter(c) }).FillWidth().Shrink(0).Padding(8)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Clip().Children(func() {
				app.documentCard(c, platform, progress, width)
			})
		})
		// 命中区域覆盖卡片左边缘，不占用布局空间或绘制分隔线。
		if app.sidebarOpen && progress >= 1 && !app.settingState.Open && w-minEditorWidth-cardGap >= minSidebarWidth {
			edge := ui.Box(c.Key("sidebar-resize")).Absolute().Left(width - 4).Top(cardGap).Bottom(cardGap).
				Width(8).Cursor(ui.CursorResizeEW)
			if dx, _, held := edge.Dragged(); held && dx != 0 {
				app.sidebarWidth = min(max(width+dx, minSidebarWidth), w-minEditorWidth-cardGap)
				c.Invalidate()
			}
		}
		app.sidebarToggle(c)
	})
}

// sidebarToggleX 将侧栏开关固定在系统窗口按钮右侧。
func sidebarToggleX(c *ui.Context) float32 {
	bar := c.TitleBar()
	if bar.Left > 0 {
		return bar.Left + 16
	}
	return 8
}

// sidebarToggle 保持固定位置，侧栏收起后叠在卡片标题行上。
func (app *Application) sidebarToggle(c *ui.Context) {
	size := max(32, c.Theme().Rem(2))
	ui.Box(c).Absolute().Left(sidebarToggleX(c)).Top(trafficMid-size/2).Size(size, size).Children(func() {
		if component.CircleIconButton(c.Key("sidebar-toggle"), icons.PanelLeft(), locales.SidebarToggle()).
			Disabled(app.settingState.Open).Clicked() {
			app.sidebarOpen = !app.sidebarOpen
			c.Invalidate()
		}
	})
}

// documentCard 把工具栏和编辑器收进同一张卡片：卡片不透明，上、右、下留缝露出整窗的底色。
// 左边的缝只在侧栏收起时出现，标题避让固定位置的侧栏开关。
func (app *Application) documentCard(c *ui.Context, platform string, progress, sidebarWidth float32) {
	t := c.Theme()
	bar := c.TitleBar()
	// 左边跟着侧栏收合，卡片顶部保留间距。
	gap := cardGap * (1 - progress)
	// 在两端的避让位置之间平滑过渡，避免标题接近开关时突然停住。
	buttonRight := sidebarToggleX(c) + max(32, t.Rem(2))
	closedInset := max(float32(8), buttonRight-cardGap)
	openInset := max(float32(8), buttonRight-sidebarWidth)
	leftInset := closedInset + (openInset-closedInset)*progress
	ui.Column(c.Key("document")).Fill().MinWidth(0).Padding(cardGap, cardGap, cardGap, gap).DragWindow().Children(func() {
		card := ui.Column(c).Fill().MinWidth(0).Radius(14).Clip().Background(t.Surface)
		if t.Dark {
			card.Border(1, t.Text.Alpha(0.12)).Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.2)).
				Shadow(0, 2, 2, -1, ui.RGBA(0, 0, 0, 0.2)).Shadow(0, 4, 6, -3, ui.RGBA(0, 0, 0, 0.24))
		} else {
			card.Border(0.5, ui.RGBA(0, 0, 0, 0.106)).InsetShadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.063))
		}
		card.Children(func() {
			if app.active == nil {
				ui.Row(c.Key("empty-workspace")).Absolute().Top(0).Bottom(0).Left(0).Right(0).
					PassThrough().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					ui.Image(c.Key("empty-background"), emptyBackground).Size(336, 336).
						MaxWidthPercent(80).MaxHeightPercent(80).Fit(ui.Contain).Opacity(0.4)
				})
			}
			// 按钮行与红绿灯同一中心线，并为滚动区的焦点环留出上下空间。
			buttonHeight := max(32, t.Rem(2))
			const topInset float32 = 4
			const bottomInset float32 = 6
			// 卡片从窗口顶部下移 cardGap，标题栏较高时仍避开原生窗口控件区域。
			toolbarHeight := max(buttonHeight+topInset+bottomInset, bar.Height-cardGap)
			ui.Row(c.Key("window-header")).FillWidth().Shrink(0).Height(toolbarHeight).
				AlignItems(ui.Center).Padding(0, max(8, bar.Right), toolbarHeight-buttonHeight-2*topInset, leftInset).DragWindow().Children(func() {
				ui.Row(c.Key("title-area")).Grow(1).MinWidth(0).FillHeight().Clip().Padding(0, 8).
					Disabled(app.settingState.Open).Children(func() { app.toolbar(c) })
				if platform == "windows" {
					var commands windowCommands
					if app.window != nil {
						commands = app.window
					}
					captionButtons(c, commands)
				}
			})
			app.content(c)
		})
	})
}

func (app *Application) content(c *ui.Context) {
	if app.active == nil {
		return
	}
	area := ui.Column(c.Key("editor-area")).Fill()
	if !c.Theme().Dark {
		// 编辑器会不透明地绘制背景，避开卡片的内阴影以保持边缘连续。
		area.Padding(0, 0.5, 0.5, 0.5)
	}
	area.Children(func() {
		app.active.View(c.Key(app.active.ID()))
	})
}

// scheduleSettings coalesces rapid slider changes. Pending changes are also
// flushed synchronously when the window closes.
func (app *Application) scheduleSettings(c *ui.Context) {
	if app.settingsPath == "" {
		return
	}
	current := app.settingState.Snapshot(mygo.Theme.Source())
	if current != app.lastSettings {
		app.lastSettings = current
		app.settingsPending = true
		app.settingsBlocked = false
		app.settingsDeadline = c.Now().Add(300 * time.Millisecond)
	}
	if !app.settingsPending || app.settingsBlocked {
		return
	}
	if wait := app.settingsDeadline.Sub(c.Now()); wait > 0 {
		c.After(wait)
	} else {
		app.flushSettings()
	}
}

func (app *Application) flushSettings() {
	if app.settingsPath == "" {
		return
	}
	current := app.settingState.Snapshot(mygo.Theme.Source())
	if current != app.lastSettings {
		app.lastSettings, app.settingsPending = current, true
	}
	if !app.settingsPending {
		return
	}
	if err := settings.Save(app.settingsPath, current); err != nil {
		app.settingsError = fmt.Errorf("保存设置失败，请修改设置后重试: %w", err)
		app.settingsBlocked = true
		log.Print(app.settingsError)
		return
	}
	app.settingsPending = false
	app.settingsBlocked = false
}
