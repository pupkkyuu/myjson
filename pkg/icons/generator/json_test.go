package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadJSONRejectsInvalidDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	for _, source := range []string{
		`{"repository":"owner/repo","unknown":true}`,
		`{"repository":"owner/repo","repository":"other/repo"}`,
		`{"repository":"owner/repo"} {}`,
		`{"Repository":"owner/repo"}`,
	} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		var cfg Config
		if err := readJSON(path, &cfg); err == nil {
			t.Errorf("accepted invalid configuration %s", source)
		}
	}
}

func TestJSONBytesOrdersMapKeys(t *testing.T) {
	got := string(jsonBytes(map[string]string{"z": "last", "a": "first"}))
	want := "{\n  \"a\": \"first\",\n  \"z\": \"last\"\n}\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
