package icons_test

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"

	"myjson/pkg/icons"
	"myjson/pkg/icons/generator"
)

func TestGeneratedIconsMatchLock(t *testing.T) {
	g, err := generator.New(generator.Options{Directory: "."})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Check(); err != nil {
		t.Fatal(err)
	}
}

func TestAccessorSharesLazilyParsedSVG(t *testing.T) {
	var results [32]*ui.SVG
	var workers sync.WaitGroup
	for i := range results {
		workers.Go(func() { results[i] = icons.Search() })
	}
	workers.Wait()
	for _, svg := range results {
		if svg == nil || svg != results[0] {
			t.Fatal("concurrent callers did not receive the same cached SVG")
		}
	}
	if w, h := results[0].Size(); w != 24 || h != 24 {
		t.Fatalf("unexpected SVG size: %gx%g", w, h)
	}
}
