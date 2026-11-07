package format

import (
	"time"

	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/editor"
	"myjson/pkg/json"
)

func (e *FormatEditor) refreshDiagnostics(c *ui.Context) {
	version := e.Buffer.Version()
	if e.version != version {
		e.version = version
		e.SetDiagnostics(version, nil)
		e.deadline = time.Time{}
		if e.Buffer.Len() != 0 {
			e.deadline = c.Now().Add(300 * time.Millisecond)
		}
	}

	select {
	case r := <-e.diagnosticResult:
		e.diagnosing = false
		e.diagnosticPolling = false
		if r.version == e.Buffer.Version() {
			e.SetDiagnostics(r.version, r.diagnostic)
			e.deadline = time.Time{}
		}
	default:
	}

	if !e.diagnosing && !e.deadline.IsZero() {
		if wait := e.deadline.Sub(c.Now()); wait > 0 {
			c.After(wait)
		} else {
			e.deadline = time.Time{}
			e.diagnosing = true

			snapshot := e.Buffer.Snapshot()
			result := e.diagnosticResult
			wake := e.Invalidate
			e.diagnosticPolling = wake == nil

			go func() {
				result <- diagnosticsResult{
					version:    snapshot.Version(),
					diagnostic: e.diagnose(snapshot.String()),
				}

				if wake != nil {
					wake()
				}
			}()

		}
	}
	if e.diagnosing && e.diagnosticPolling {
		c.After(16 * time.Millisecond)
	}
}

func (e *FormatEditor) diagnose(text string) []mygoeditor.Diagnostic {
	diagnostics := json.Validate(text)
	return editor.Diagnostics(diagnostics)
}
