package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	firstCommit = "1111111111111111111111111111111111111111"
	nextCommit  = "2222222222222222222222222222222222222222"
	testSVG     = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 12h16"/></svg>`
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type upstreamFixture struct {
	commit   string
	missing  string
	badSVG   bool
	requests []string
}

func (f *upstreamFixture) client() *http.Client {
	return &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		f.requests = append(f.requests, r.URL.String())
		if r.Header.Get("User-Agent") == "" {
			return nil, fmt.Errorf("missing user agent")
		}
		body, status := "", http.StatusOK
		switch {
		case r.URL.Host == "api.github.com":
			body = fmt.Sprintf(`{"sha":%q}`, f.commit)
		case strings.HasSuffix(r.URL.Path, "/"+f.missing+".svg"):
			status = http.StatusNotFound
		case strings.HasSuffix(r.URL.Path, "/LICENSE"):
			body = "fixture license " + f.commit
		default:
			if !strings.Contains(r.URL.Path, "/"+f.commit+"/") {
				return nil, fmt.Errorf("asset was not fetched at pinned commit: %s", r.URL)
			}
			body = testSVG
			if f.badSVG {
				body = "<html>error</html>"
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}
}

func setup(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	cfg := Config{Repository: "lucide-icons/lucide", Ref: "main", Icons: names}
	if err := os.WriteFile(filepath.Join(dir, configFile), jsonBytes(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func setupSynced(t *testing.T, names ...string) (string, *upstreamFixture) {
	t.Helper()
	dir := setup(t, names...)
	fixture := &upstreamFixture{commit: firstCommit}
	if err := invoke(t, dir, fixture.client(), "sync"); err != nil {
		t.Fatal(err)
	}
	return dir, fixture
}

func invoke(t *testing.T, dir string, client *http.Client, args ...string) error {
	t.Helper()
	return Run(context.Background(), append([]string{"-dir", dir}, args...), client, io.Discard)
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files[rel] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestSyncPinsSourceAndWorksOffline(t *testing.T) {
	dir := setup(t, "search", "arrow-up")
	f := &upstreamFixture{commit: firstCommit}
	if err := invoke(t, dir, f.client(), "sync"); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 4 {
		t.Fatalf("expected commit lookup, two icons and license; got %v", f.requests)
	}
	cfg, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := readLock(dir, cfg)
	if err != nil || locked == nil || locked.Commit != firstCommit || len(locked.Icons) != 2 {
		t.Fatalf("invalid lock: %+v, %v", locked, err)
	}
	before := snapshot(t, dir)
	if len(before) != 4 {
		t.Fatalf("expected only config, lock, license and Go code; got %v", before)
	}
	noNetwork := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network must not be used")
	})}
	for _, cmd := range []string{"sync", "generate", "check"} {
		if err := invoke(t, dir, noNetwork, cmd); err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("sync/generate/check changed reproducible files")
	}
}

func TestAddUsesLockedRevisionAndFailurePreservesFiles(t *testing.T) {
	dir, f := setupSynced(t, "search")
	f.requests = nil
	if err := invoke(t, dir, f.client(), "add", "copy"); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 1 || !strings.HasSuffix(f.requests[0], "/"+firstCommit+"/icons/copy.svg") {
		t.Fatalf("add must fetch only the new icon at the locked revision: %v", f.requests)
	}
	before := snapshot(t, dir)
	f.missing = "missing"
	if err := invoke(t, dir, f.client(), "add", "arrow-up", "missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected missing icon error, got %v", err)
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("failed add partially wrote config, lock, or generated code")
	}
}

func TestUpdateResolvesOneCommitAndRejectsBadSVG(t *testing.T) {
	dir, f := setupSynced(t, "search")
	before := snapshot(t, dir)
	f.commit, f.badSVG = nextCommit, true
	if err := invoke(t, dir, f.client(), "update"); err == nil {
		t.Fatal("accepted invalid SVG")
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("failed update changed existing assets")
	}
	f.badSVG, f.requests = false, nil
	if err := invoke(t, dir, f.client(), "update"); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 3 {
		t.Fatalf("expected one resolution and two pinned assets: %v", f.requests)
	}
	cfg, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := readLock(dir, cfg)
	if err != nil || locked == nil {
		t.Fatalf("read lock: %v", err)
	}
	if locked.Commit != nextCommit {
		t.Fatal("update did not advance the lock")
	}
}

func TestExplicitRevisionDoesNotFetchBranch(t *testing.T) {
	dir := setup(t, "search")
	f := &upstreamFixture{commit: firstCommit}
	if err := invoke(t, dir, f.client(), "update", firstCommit); err != nil {
		t.Fatal(err)
	}
	for _, address := range f.requests {
		if strings.Contains(address, "api.github.com") {
			t.Fatal("full SHA should not require ref resolution")
		}
	}
}

func TestCheckDetectsDriftWithoutWritingAndSyncRepairsIt(t *testing.T) {
	dir, f := setupSynced(t, "search")
	path := filepath.Join(dir, codeFile)
	code, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.ReplaceAll(code, []byte("M4 12h16"), []byte("M5 12h15")), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	if err := invoke(t, dir, f.client(), "check"); err == nil {
		t.Fatal("check accepted modified embedded SVG")
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("check wrote files")
	}
	f.requests = nil
	if err := invoke(t, dir, f.client(), "sync"); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 1 {
		t.Fatalf("expected only the damaged SVG to be fetched: %v", f.requests)
	}
	if err := os.WriteFile(path, append(code, []byte("\n// stale comment\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if err := invoke(t, dir, f.client(), "check"); err == nil {
		t.Fatal("check accepted stale generated code")
	}
	if err := invoke(t, dir, f.client(), "generate"); err != nil {
		t.Fatal(err)
	}
	if err := invoke(t, dir, f.client(), "check"); err != nil {
		t.Fatal(err)
	}
}

func TestSyncRemovesAccessorsWithoutTouchingUnmanagedFiles(t *testing.T) {
	dir, f := setupSynced(t, "search", "copy")
	custom := filepath.Join(dir, "custom.svg")
	if err := os.WriteFile(custom, []byte(testSVG), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Icons = []string{"search"}
	if err := os.WriteFile(filepath.Join(dir, configFile), jsonBytes(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	f.requests = nil
	if err := invoke(t, dir, f.client(), "sync"); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(dir, codeFile))
	if err != nil || bytes.Contains(code, []byte("func Copy()")) || len(f.requests) != 0 {
		t.Fatalf("removing an icon failed or fetched upstream: %v, %v", err, f.requests)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Fatal("unmanaged SVG was removed")
	}
}

func TestSyncRestoresMissingOrUnusableGoSources(t *testing.T) {
	for _, damage := range []string{"missing", "syntax", "expression"} {
		t.Run(damage, func(t *testing.T) {
			dir, f := setupSynced(t, "search")
			path := filepath.Join(dir, codeFile)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "missing":
				err = os.Remove(path)
			case "syntax":
				err = os.WriteFile(path, []byte("broken Go source"), 0644)
			case "expression":
				err = os.WriteFile(path, bytes.ReplaceAll(original, fmt.Appendf(nil, "get(%q)", testSVG), fmt.Appendf(nil, "get(%q + %q)", testSVG, "")), 0644)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := snapshot(t, dir)
			for _, command := range []string{"check", "generate"} {
				if err := invoke(t, dir, f.client(), command); err == nil || !strings.Contains(err.Error(), "sync") {
					t.Fatalf("%s should require sync after %s damage: %v", command, damage, err)
				}
			}
			if !reflect.DeepEqual(before, snapshot(t, dir)) {
				t.Fatal("failed local operation changed files")
			}
			f.requests = nil
			if err := invoke(t, dir, f.client(), "sync"); err != nil {
				t.Fatal(err)
			}
			if len(f.requests) != 1 || !reflect.DeepEqual(original, []byte(snapshot(t, dir)[codeFile])) {
				t.Fatalf("sync did not restore only the locked SVG: %v", f.requests)
			}
		})
	}
}

func TestNamesAndGeneratedAccessors(t *testing.T) {
	for name, want := range map[string]string{"arrow-up": "ArrowUp", "3d-box": "Icon3dBox", "x": "X"} {
		if got, err := identifier(name); err != nil || got != want {
			t.Errorf("%q: got %q, %v", name, got, err)
		}
	}
	for _, names := range [][]string{{"../search"}, {"Search"}, {"search", "search"}, {"foo1", "foo-1"}} {
		cfg := Config{Repository: "lucide-icons/lucide", Ref: "main", Icons: names}
		if err := cfg.validate(); err == nil {
			t.Errorf("accepted invalid names: %v", names)
		}
	}
	code, err := render(Config{Icons: []string{"search"}}, map[string][]byte{"search": []byte(testSVG)})
	if err != nil || !bytes.Contains(code, fmt.Appendf(nil, "return cacheSearch.get(%q)", testSVG)) {
		t.Fatalf("missing inline SVG and lazy accessor: %s, %v", code, err)
	}
	if bytes.Contains(code, []byte("go:embed")) {
		t.Fatal("generated code must build without the local SVG cache")
	}
	if bytes.Contains(code, []byte("func init(")) {
		t.Fatal("generated accessors eagerly initialize resources")
	}
}

func TestInvalidManifestNeverFetchesOrChangesFiles(t *testing.T) {
	for _, manifest := range []string{
		`{"repository":"lucide-icons/lucide","ref":"main","icon":["search"]}`,
		`{"repository":"lucide-icons/lucide","ref":"main"}`,
		`{"repository":"lucide-icons/lucide","ref":"main","icons":["../search"]}`,
	} {
		dir := setup(t, "search")
		if err := os.WriteFile(filepath.Join(dir, configFile), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		before := snapshot(t, dir)
		f := &upstreamFixture{commit: firstCommit}
		if err := invoke(t, dir, f.client(), "sync"); err == nil {
			t.Fatalf("accepted invalid manifest: %s", manifest)
		}
		if len(f.requests) != 0 || !reflect.DeepEqual(before, snapshot(t, dir)) {
			t.Fatal("invalid manifest caused network activity or writes")
		}
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type failingCloseBody struct {
	io.Reader
	err error
}

func (b failingCloseBody) Close() error { return b.err }

func TestOutputErrorsAreReturned(t *testing.T) {
	writeErr := errors.New("output failed")
	writer := failingWriter{err: writeErr}
	if err := Run(context.Background(), []string{"-help"}, nil, writer); !errors.Is(err, writeErr) {
		t.Fatalf("help output error lost: %v", err)
	}
	dir, fixture := setupSynced(t, "search")
	for _, command := range []string{"check", "generate", "sync"} {
		err := Run(context.Background(), []string{"-dir", dir, command}, fixture.client(), writer)
		if !errors.Is(err, writeErr) {
			t.Errorf("%s output error lost: %v", command, err)
		}
	}
}

func TestFetchReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: failingCloseBody{Reader: strings.NewReader("body"), err: closeErr}}, nil
	})}
	_, err := (remote{client: client}).fetch(context.Background(), "https://example.com")
	if !errors.Is(err, closeErr) {
		t.Fatalf("close error lost: %v", err)
	}
}

func TestSyncReportsUnreadableCachedLicense(t *testing.T) {
	dir, fixture := setupSynced(t, "search")
	path := filepath.Join(dir, licenseFile)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	fixture.requests = nil
	if err := invoke(t, dir, fixture.client(), "sync"); err == nil {
		t.Fatal("expected unreadable cached license error")
	}
	if len(fixture.requests) != 0 {
		t.Fatalf("unexpected remote fetch: %v", fixture.requests)
	}
}
