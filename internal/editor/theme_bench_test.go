package editor_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
)

var benchmarkTheme *mygoeditor.Theme

func BenchmarkEditorTheme(b *testing.B) {
	palette := *ui.LightTheme()
	var cache editor.ThemeCache
	cache.Resolve(&palette)
	b.ReportAllocs()
	for b.Loop() {
		// 应用每帧重建 UI 主题；相同颜色不应重建编辑器主题。
		frameTheme := palette
		benchmarkTheme = cache.Resolve(&frameTheme)
	}
}
