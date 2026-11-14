package format

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/pkg/json"
)

var benchmarkDiagnostics []mygoeditor.Diagnostic

func BenchmarkDiagnostics(b *testing.B) {
	row := `{"id":123,"text":"hello","enabled":true},`
	valid := "[" + strings.Repeat(row, (4_860_000-6)/len(row)) + "null]"
	for _, tc := range []struct{ name, source string }{
		{"valid", valid}, {"EOF", valid[:len(valid)-1]},
	} {
		b.Run(tc.name, func(b *testing.B) {
			e := NewFormatEditor("editor-1", tc.source)
			snapshot := e.Buffer.Snapshot()
			for _, implementation := range []struct {
				name string
				run  func() []mygoeditor.Diagnostic
			}{
				{"previous", func() []mygoeditor.Diagnostic {
					source := snapshot.String()
					data := jsontext.Value(source)
					if data.IsValid() {
						return nil
					}
					return editor.Diagnostics(json.DiagnosticsForError(source, data.Format()))
				}},
				{"bytes", func() []mygoeditor.Diagnostic { return e.diagnose(snapshot) }},
			} {
				b.Run(implementation.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(snapshot.Len()))
					for b.Loop() {
						benchmarkDiagnostics = implementation.run()
					}
				})
			}
		})
	}
}
