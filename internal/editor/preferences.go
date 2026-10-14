package editor

import mygoeditor "github.com/egoist/mygo/ui/editor"

type Preferences struct {
	FontSize     float32
	LineHeight   float32
	TabSize      int
	DefaultWrap  bool
	Whitespace   mygoeditor.Whitespace
	IndentGuides bool
	ActiveLine   bool
	AutoClose    bool
	DiffLayout   mygoeditor.DiffLayout
	DiffInline   mygoeditor.DiffInline
}

func DefaultPreferences() Preferences {
	return Preferences{
		FontSize: 13, LineHeight: 1.5, TabSize: 2,
		Whitespace: mygoeditor.WhitespaceSelection, IndentGuides: true, AutoClose: true,
		DiffLayout: mygoeditor.DiffSplit, DiffInline: mygoeditor.DiffInlineChar,
	}
}

// Apply 更新外观和输入行为，保留文本及当前文档的换行设置。
func (p Preferences) Apply(st *mygoeditor.State) {
	st.Font.Size = p.FontSize
	st.LineHeight = p.LineHeight
	st.TabSize = p.TabSize
	st.Whitespace = p.Whitespace
	st.NoIndentGuides = !p.IndentGuides
	st.NoActiveLine = !p.ActiveLine
	st.NoAutoClose = !p.AutoClose
}
