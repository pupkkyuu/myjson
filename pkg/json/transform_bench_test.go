package json_test

import (
	"encoding/json/jsontext"
	"strings"
	"testing"

	"myjson/pkg/json"
)

var benchmarkTransform string

func BenchmarkTransforms(b *testing.B) {
	for _, transform := range []struct {
		name string
		row  string
		opts []jsontext.Options
		run  func(string) (string, error)
	}{
		{"Format", `{"id":123,"text":"hello","enabled":true},`, []jsontext.Options{jsontext.WithIndent("  ")}, json.Format},
		{"Compact", "  {\n    \"id\": 123,\n    \"text\": \"hello\",\n    \"enabled\": true\n  },\n", nil, json.Compact},
	} {
		b.Run(transform.name, func(b *testing.B) {
			for _, size := range []struct {
				name  string
				bytes int
			}{{"small", 1024}, {"4.86MB", 4_860_000}} {
				b.Run(size.name, func(b *testing.B) {
					source := "[" + strings.Repeat(transform.row, (size.bytes-6)/len(transform.row)) + "null]"
					for _, implementation := range []struct {
						name string
						run  func(string) (string, error)
					}{
						{"previous", func(source string) (string, error) {
							output := jsontext.Value(source)
							if err := output.Format(transform.opts...); err != nil {
								return "", err
							}
							return string(output), nil
						}},
						{"append", transform.run},
					} {
						b.Run(implementation.name, func(b *testing.B) {
							b.ReportAllocs()
							b.SetBytes(int64(len(source)))
							for b.Loop() {
								var err error
								benchmarkTransform, err = implementation.run(source)
								if err != nil {
									b.Fatal(err)
								}
							}
						})
					}
				})
			}
		})
	}
}
