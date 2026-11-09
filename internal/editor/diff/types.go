package diff

import (
	"strings"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/editor/format"
	"myjson/internal/locales"
)

type DiffEditor struct {
	id    string
	title string

	left  *format.FormatEditor
	right *format.FormatEditor

	preview      mygoeditor.DiffState
	inputSplit   float32
	comparing    bool
	activeSide   int
	requestFocus bool
	currentHunk  int
	versions     [2]uint64

	transformResults  chan [2]diffPreparedTransform
	transformCanceled bool
	transformPolling  bool

	Invalidate func()
}

type diffPreparedTransform struct {
	format.PreparedTransform
	err error
}

var _ editor.ComparisonEditor = (*DiffEditor)(nil)

func NewDiffEditor(id string, left, right string, preferences ...editor.Preferences) *DiffEditor {
	p := editor.DefaultPreferences()
	if len(preferences) != 0 {
		p = preferences[0]
	}
	d := &DiffEditor{
		id:          id,
		title:       locales.ComparisonTabTitle(strings.TrimPrefix(id, "editor-")),
		left:        format.NewFormatEditor(id, left, p),
		right:       format.NewFormatEditor(id, right, p),
		currentHunk: -1,
		inputSplit:  50,
	}
	d.left.Label, d.right.Label = locales.LeftEditorLabel(), locales.RightEditorLabel()
	d.left.Placeholder, d.right.Placeholder = locales.LeftEditorPlaceholder(), locales.RightEditorPlaceholder()
	d.preview = mygoeditor.DiffState{
		Old:             d.left.Buffer,
		New:             d.right.Buffer,
		ExpandUnchanged: true,
		RenderFindBar:   editor.RenderFindBar,
	}
	d.ApplyPreferences(p)
	d.preview.Wrap = p.DefaultWrap
	for _, binding := range mygoeditor.DefaultBindings() {
		if binding.Command == mygoeditor.ToggleSoftWrap {
			binding.Command = mygoeditor.NoCommand
			d.preview.Bindings = append(d.preview.Bindings, binding)
			d.left.Bindings = append(d.left.Bindings, binding)
			d.right.Bindings = append(d.right.Bindings, binding)
		}
	}
	d.syncVersions()
	return d
}

func (e *DiffEditor) ApplyPreferences(p editor.Preferences) {
	e.left.ApplyPreferences(p)
	e.right.ApplyPreferences(p)
	e.preview.Font.Size = p.FontSize
	e.preview.LineHeight = p.LineHeight
	e.preview.TabSize = p.TabSize
	e.preview.Layout = p.DiffLayout
	e.preview.Inline = p.DiffInline
}

func (e *DiffEditor) ID() string {
	return e.id
}

func (e *DiffEditor) Title() string {
	return e.title
}

func (e *DiffEditor) Comparing() bool { return e.comparing }

func (e *DiffEditor) DifferenceCount() int { return len(e.preview.Hunks()) }
