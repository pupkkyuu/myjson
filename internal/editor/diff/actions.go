package diff

import (
	"strings"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor/format"
	"myjson/pkg/json"
)

func (e *DiffEditor) Format() {
	indent := strings.Repeat(" ", max(1, e.left.TabSize))
	e.transform(format.SkipBlank(func(source string) (string, error) { return json.FormatWithIndent(source, indent) }))
}

func (e *DiffEditor) Compact() {
	e.transform(format.SkipBlank(json.Compact))
}

func (e *DiffEditor) Escape() {
	e.transform(json.Escape)
}

func (e *DiffEditor) Unescape() {
	e.transform(json.Unescape)
}

func (e *DiffEditor) SortFields() {
	indent := strings.Repeat(" ", max(1, e.left.TabSize))
	e.transform(format.SkipBlank(func(source string) (string, error) { return json.SortFieldsWithIndent(source, indent) }))
}

func (e *DiffEditor) FoldAll() {
	if e.comparing {
		e.preview.CollapseAll()
	} else {
		e.left.Run(mygoeditor.FoldAll)
		e.right.Run(mygoeditor.FoldAll)
	}
	e.Focus()
}

func (e *DiffEditor) UnfoldAll() {
	if e.comparing {
		e.preview.ExpandAll()
	} else {
		e.left.Run(mygoeditor.UnfoldAll)
		e.right.Run(mygoeditor.UnfoldAll)
	}
	e.Focus()
}

func (e *DiffEditor) ToggleWrap() {
	enabled := !e.WrapEnabled()
	e.left.SoftWrap, e.right.SoftWrap, e.preview.Wrap = enabled, enabled, enabled
	e.Focus()
}

func (e *DiffEditor) WrapEnabled() bool {
	return e.left.SoftWrap
}

func (e *DiffEditor) Search() {
	e.requestFocus = false
	e.left.CancelFocus()
	e.right.CancelFocus()
	if e.comparing {
		e.preview.Run(mygoeditor.Find)
	} else {
		[2]*format.FormatEditor{e.left, e.right}[e.activeSide].Search()
	}
}

func (e *DiffEditor) CopyEnabled() bool {
	return false
}

func (e *DiffEditor) CopyAll(writeClipboard func(string)) {
	writeClipboard(e.left.Buffer.String() + "\n" + e.right.Buffer.String())
	e.Focus()
}

func (e *DiffEditor) Clear() {
	e.transformResults = nil
	e.transformCanceled, e.transformPolling = false, false
	e.left.Clear()
	e.right.Clear()
	e.comparing, e.activeSide = false, 0
	e.currentHunk = -1
	e.syncVersions()
	e.Focus()
}

func (e *DiffEditor) Focus() {
	e.requestFocus = true
}

func (e *DiffEditor) ToggleComparison() {
	e.transformCanceled = true
	e.comparing = !e.comparing
	e.currentHunk = -1
	e.Focus()
}

func (e *DiffEditor) NextDifference() {
	e.goDifference(1)
}

func (e *DiffEditor) PreviousDifference() {
	e.goDifference(-1)
}

func (e *DiffEditor) syncVersions() {
	versions := [2]uint64{e.left.Buffer.Version(), e.right.Buffer.Version()}
	if e.versions != versions {
		e.versions = versions
		e.currentHunk = -1
	}
}

func (e *DiffEditor) goDifference(direction int) {
	if !e.comparing {
		return
	}
	e.syncVersions()
	count := e.DifferenceCount()
	if count == 0 {
		e.currentHunk = -1
		return
	}
	if e.currentHunk < 0 {
		if direction > 0 {
			e.currentHunk = 0
		} else {
			e.currentHunk = count - 1
		}
	} else {
		e.currentHunk = (e.currentHunk + direction + count) % count
	}
	e.preview.GoToHunk(e.currentHunk)
	e.Focus()
}
