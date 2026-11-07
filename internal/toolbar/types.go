package toolbar

import "github.com/egoist/mygo/ui"

type EditorKind uint8

const (
	NoEditor EditorKind = iota
	FormatEditor
	DiffEditor
)

type Action string

type Options struct {
	Title           string
	Kind            EditorKind
	WrapEnabled     bool
	CopyEnabled     bool
	Loading         bool
	ShowLoading     bool
	Comparing       bool
	DifferenceCount int
	DisabledActions []Action
}

type tool struct {
	action   Action
	icon     *ui.SVG
	label    string
	disabled bool
}
