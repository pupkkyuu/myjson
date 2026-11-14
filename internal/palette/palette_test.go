package palette

import (
	"reflect"
	"testing"
)

var benchmarkLook Look

func TestLookForFallback(t *testing.T) {
	for _, dark := range []bool{false, true} {
		want := LookFor(Default, dark)
		for _, id := range []string{"", "missing"} {
			if got := LookFor(id, dark); !reflect.DeepEqual(got, want) {
				t.Fatalf("LookFor(%q, %t) 未回退到默认配色", id, dark)
			}
		}
	}
}

func TestLookForDoesNotAllocate(t *testing.T) {
	for _, id := range append(IDs(), "", "missing") {
		for _, dark := range []bool{false, true} {
			allocs := testing.AllocsPerRun(100, func() {
				benchmarkLook = LookFor(id, dark)
			})
			if allocs != 0 {
				t.Fatalf("LookFor(%q, %t) 每次分配 %g 次", id, dark, allocs)
			}
		}
	}
}

func BenchmarkLookFor(b *testing.B) {
	for _, w := range washes {
		b.Run(w.id, func(b *testing.B) {
			for _, appearance := range []struct {
				name string
				dark bool
			}{{"light", false}, {"dark", true}} {
				b.Run(appearance.name, func(b *testing.B) {
					b.Run("build", func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							benchmarkLook = buildLook(w, appearance.dark)
						}
					})
					b.Run("lookup", func(b *testing.B) {
						b.ReportAllocs()
						for b.Loop() {
							benchmarkLook = LookFor(w.id, appearance.dark)
						}
					})
				})
			}
		})
	}
}
