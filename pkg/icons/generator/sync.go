package generator

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const maxDownload = 2 << 20

type remote struct{ client *http.Client }

func (r remote) fetch(ctx context.Context, address string) (data []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "lucide-icongen")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close response body: %w", closeErr))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d (icon may be missing/renamed, or GitHub may be rate limiting)", address, resp.StatusCode)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, fmt.Errorf("GET %s: exceeds %d bytes", address, maxDownload)
	}
	return data, nil
}

func (r remote) resolve(ctx context.Context, repository, ref string) (string, error) {
	if commitPattern.MatchString(ref) {
		return ref, nil
	}
	data, err := r.fetch(ctx, "https://api.github.com/repos/"+repository+"/commits/"+url.PathEscape(ref))
	if err != nil {
		return "", err
	}
	var revision struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(data, &revision); err != nil {
		return "", err
	}
	if !commitPattern.MatchString(revision.SHA) {
		return "", fmt.Errorf("GitHub returned an invalid commit for %q", ref)
	}
	return revision.SHA, nil
}

func (r remote) asset(ctx context.Context, repository, commit, path string) ([]byte, error) {
	return r.fetch(ctx, "https://raw.githubusercontent.com/"+repository+"/"+commit+"/"+path)
}

func synchronize(ctx context.Context, dir string, cfg Config, source remote, update bool, ref string, out io.Writer) error {
	old, err := readLock(dir, cfg)
	if err != nil {
		return err
	}
	commit := ""
	if old != nil && !update {
		commit = old.Commit
	} else {
		commit, err = source.resolve(ctx, cfg.Repository, ref)
		if err != nil {
			return err
		}
	}
	next := lock{Repository: cfg.Repository, Commit: commit, Icons: make(map[string]string)}
	embedded, err := readEmbeddedSVGs(dir)
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	sources := make(map[string][]byte)
	load := func(upstreamPath string, cached []byte, expected string) ([]byte, error) {
		if expected != "" && digest(cached) == expected {
			return cached, nil
		}
		data, err := source.asset(ctx, cfg.Repository, commit, upstreamPath)
		if err != nil {
			return nil, err
		}
		if expected != "" && digest(data) != expected {
			return nil, fmt.Errorf("%s at locked commit %s differs from its checksum; no files changed", upstreamPath, commit)
		}
		return data, nil
	}
	for _, name := range cfg.Icons {
		expected := ""
		if old != nil && old.Commit == commit {
			expected = old.Icons[name]
		}
		symbol, _ := identifier(name)
		data, err := load("icons/"+name+".svg", embedded[symbol], expected)
		if err != nil {
			return err
		}
		if err := validateSVG(name, data); err != nil {
			return err
		}
		next.Icons[name] = digest(data)
		sources[name] = data
	}
	licenseHash := ""
	if old != nil && old.Commit == commit {
		licenseHash = old.LicenseSHA256
	}
	previousLicense, err := os.ReadFile(filepath.Join(dir, licenseFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	license, err := load("LICENSE", previousLicense, licenseHash)
	if err != nil {
		return err
	}
	if len(license) == 0 {
		return fmt.Errorf("upstream LICENSE is empty")
	}
	next.LicenseSHA256 = digest(license)
	files := make(map[string][]byte)
	files[licenseFile] = license
	files[configFile], files[lockFile] = jsonBytes(cfg), jsonBytes(next)
	files[codeFile], err = render(cfg, sources)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeFiles(dir, files); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Synced %d icons at %s\n", len(cfg.Icons), commit); err != nil {
		return fmt.Errorf("write sync status (files already updated): %w", err)
	}
	return nil
}
