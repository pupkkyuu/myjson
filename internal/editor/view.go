package editor

import (
	"github.com/egoist/mygo/ui"
	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/internal/locales"
	"myjson/pkg/json"
)

type Palette struct {
	text, key, stringValue, number, comment string
}

var vsLight = buildTheme(Palette{
	text:        "#000000",
	key:         "#A31515",
	stringValue: "#0451A5",
	number:      "#098658",
	comment:     "#008000",
})

var vsDark = buildTheme(Palette{
	text:        "#DCDCDC",
	key:         "#9CDCFE",
	stringValue: "#CE9178",
	number:      "#B5CEA8",
	comment:     "#608B4E",
})

func Theme(t *ui.Theme) *mygoeditor.Theme {
	base := vsLight
	if t.Dark {
		base = vsDark
	}
	theme := *base
	theme.Background = t.Surface
	return &theme
}

func buildTheme(palette Palette) *mygoeditor.Theme {
	t := &mygoeditor.Theme{Text: ui.Hex(palette.text)}
	colors := map[mygoeditor.TokenKind]string{
		mygoeditor.TokenText:        palette.text,
		mygoeditor.TokenProperty:    palette.key,
		mygoeditor.TokenString:      palette.stringValue,
		mygoeditor.TokenEscape:      palette.stringValue,
		mygoeditor.TokenNumber:      palette.number,
		mygoeditor.TokenConstant:    palette.stringValue,
		mygoeditor.TokenKeyword:     palette.stringValue,
		mygoeditor.TokenComment:     palette.comment,
		mygoeditor.TokenOperator:    palette.text,
		mygoeditor.TokenPunctuation: palette.text,
	}
	for kind, hex := range colors {
		t.Tokens[kind].Color = ui.Hex(hex)
	}
	return t
}

// Diagnostics 将 JSON 诊断转换为编辑器诊断。
func Diagnostics(diagnostics []json.Diagnostic) []mygoeditor.Diagnostic {
	if len(diagnostics) == 0 {
		return nil
	}
	result := make([]mygoeditor.Diagnostic, len(diagnostics))
	for i, d := range diagnostics {
		result[i] = mygoeditor.Diagnostic{Start: d.Start, End: d.End, Message: locales.JsonErrorLocation(d.Line, d.Column, d.Detail), Severity: mygoeditor.SeverityError, Source: "JSON"}
	}
	return result
}
