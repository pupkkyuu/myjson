package format

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/internal/locales"
	"myjson/pkg/json"
)

type FormatEditor struct {
	id    string
	title string
	focus ui.Handle
	mygoeditor.State

	pendingTransform  chan transformResult
	diagnosticResult  chan diagnosticsResult
	transformPolling  bool
	diagnosticPolling bool

	version    uint64
	deadline   time.Time
	diagnosing bool

	Invalidate func()
}

type transformResult struct {
	version uint64
	text    string
	err     error
}

type diagnosticsResult struct {
	version    uint64
	diagnostic []mygoeditor.Diagnostic
}

type PreparedTransform struct {
	Version     uint64
	Text        string
	Changed     bool
	Diagnostics []json.Diagnostic
}

var _ editor.Editor = (*FormatEditor)(nil)

func NewFormatEditor(id string, content string, preferences ...editor.Preferences) *FormatEditor {
	buffer := mygoeditor.NewBuffer(content)
	buffer.SetLanguage(mygoeditor.LanguageFor("document.json"))
	e := &FormatEditor{
		id:               id,
		title:            locales.FormatTabTitle(strings.TrimPrefix(id, "editor-")),
		diagnosticResult: make(chan diagnosticsResult, 1),
		version:          buffer.Version(),
		Buffer:           buffer,
		Label:            locales.EditorLabel(),
		Placeholder:      locales.EditorPlaceholder(),
		NoActiveLine:     true,
		Bindings:         []mygoeditor.Binding{{Mods: ui.Cmd | ui.Shift, Key: ui.KeyF, Command: mygoeditor.NoCommand}},
		RenderFindBar:    editor.RenderFindBar,
	}
	p := editor.DefaultPreferences()
	if len(preferences) != 0 {
		p = preferences[0]
	}
	e.ApplyPreferences(p)
	e.SoftWrap = p.DefaultWrap

	if buffer.Len() != 0 {
		e.deadline = time.Now().Add(300 * time.Millisecond)
	}

	return e
}

func (e *FormatEditor) ApplyPreferences(p editor.Preferences) { p.Apply(&e.State) }

func (e *FormatEditor) ID() string {
	return e.id
}

func (e *FormatEditor) Title() string {
	return e.title
}
