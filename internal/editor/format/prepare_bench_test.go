package format

import (
	"strings"
	"testing"

	mygoeditor "github.com/egoist/mygo/ui/editor"

	"myjson/pkg/json"
)

var benchmarkPreparedTransform PreparedTransform

func BenchmarkPrepareJSONTransform(b *testing.B) {
	source := "[" + strings.Repeat(`{"z":123,"a":"中文🙂"},`, 10_000) + "null]"
	snapshot := mygoeditor.NewBuffer(source).Snapshot()
	for _, transform := range []struct {
		name string
		run  func(string) (string, error)
	}{
		{"Format", json.Format},
		{"Compact", json.Compact},
		{"SortFields", json.SortFields},
		{"Escape", json.Escape},
	} {
		b.Run(transform.name, func(b *testing.B) {
			for _, policy := range []struct {
				name  string
				value OutputPolicy
			}{
				{"ValidateOutput", ValidateOutput},
				{"ValidJSONOutput", ValidJSONOutput},
			} {
				b.Run(policy.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(snapshot.Len()))
					for b.Loop() {
						result, err := PrepareTransform(snapshot, transform.run, policy.value)
						if err != nil {
							b.Fatal(err)
						}
						benchmarkPreparedTransform = result
					}
				})
			}
		})
	}
}
