package generator_test

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"myjson/pkg/icons/generator"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicAPIInitializesIndependentPackage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "glyphs")
	calls := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M4 12h16"/></svg>`
		if strings.HasSuffix(r.URL.Path, "/LICENSE") {
			body = "test license"
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	g, err := generator.New(generator.Options{Directory: dir, HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("constructor touched the filesystem")
	}
	cfg := generator.Config{
		Repository: "lucide-icons/lucide",
		Ref:        strings.Repeat("1", 40),
		Package:    "glyphs",
		Icons:      []string{"search", "arrow-up"},
	}
	if err := g.Init(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Icons, []string{"search", "arrow-up"}) {
		t.Fatal("Init mutated caller-owned configuration")
	}
	if calls != 3 {
		t.Fatalf("expected two icons and license at full SHA, got %d requests", calls)
	}
	if err := g.Init(context.Background(), cfg); err == nil || calls != 3 {
		t.Fatal("Init replaced an existing manifest or accessed the network")
	}
	if err := g.Generate(); err != nil {
		t.Fatal(err)
	}
	if err := g.Check(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "icons_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "icons_gen.go", data, 0)
	if err != nil || file.Name.Name != "glyphs" {
		t.Fatalf("invalid custom package: %v", err)
	}
	for _, imported := range file.Imports {
		if imported.Path.Value != `"sync"` && imported.Path.Value != `"github.com/egoist/mygo/ui"` {
			t.Fatalf("generated package imports application/generator code: %s", imported.Path.Value)
		}
	}
	if !strings.Contains(string(data), "type lazyIcon struct") {
		t.Fatal("generated package requires a separate cache implementation")
	}
}

func TestPublicAPIRejectsInvalidConfigAndCanceledContext(t *testing.T) {
	if _, err := generator.New(generator.Options{}); err == nil {
		t.Fatal("accepted an empty directory")
	}
	dir := filepath.Join(t.TempDir(), "icons")
	g, err := generator.New(generator.Options{Directory: dir, HTTPClient: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid or canceled request accessed the network")
		return nil, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	cfg := generator.Config{Repository: "lucide-icons/lucide", Ref: "main", Icons: []string{}}
	for _, name := range []string{"_", "type", "../icons", "bad-name"} {
		cfg.Package = name
		if err := g.Init(context.Background(), cfg); err == nil {
			t.Fatalf("accepted invalid package %q", name)
		}
	}
	cfg.Package = ""
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := g.Init(ctx, cfg); !errors.Is(err, context.Canceled) {
		t.Fatalf("lost caller cancellation: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("failed initialization wrote files")
	}
}
