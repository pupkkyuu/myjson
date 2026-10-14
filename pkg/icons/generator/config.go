package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	configFile  = "lucide.json"
	lockFile    = "lucide.lock.json"
	codeFile    = "icons_gen.go"
	licenseFile = "LICENSE.lucide"
)

var (
	iconNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	repoPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hashPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Config struct {
	Repository string   `json:"repository"`
	Ref        string   `json:"ref"`
	Package    string   `json:"package,omitempty"`
	Icons      []string `json:"icons"`
}

type lock struct {
	Repository    string            `json:"repository"`
	Commit        string            `json:"commit"`
	Icons         map[string]string `json:"icons"`
	LicenseSHA256 string            `json:"license_sha256"`
}

func readConfig(dir string) (Config, error) {
	var cfg Config
	if err := readJSON(filepath.Join(dir, configFile), &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if name := c.packageName(); !token.IsIdentifier(name) || name == "_" {
		return fmt.Errorf("invalid Go package name %q", name)
	}
	if !repoPattern.MatchString(c.Repository) || strings.TrimSpace(c.Ref) == "" {
		return fmt.Errorf("%s requires a repository in owner/name format and a nonempty ref", configFile)
	}
	if c.Icons == nil {
		return fmt.Errorf("%s requires an icons array (use [] for an empty collection)", configFile)
	}
	sort.Strings(c.Icons)
	symbols := map[string]string{}
	for _, name := range c.Icons {
		symbol, err := identifier(name)
		if err != nil {
			return err
		}
		if previous, ok := symbols[symbol]; ok {
			return fmt.Errorf("duplicate or conflicting icon names %q and %q (%s)", previous, name, symbol)
		}
		symbols[symbol] = name
	}
	return nil
}

func (c Config) packageName() string {
	if c.Package == "" {
		return "icons"
	}
	return c.Package
}

func readLock(dir string, cfg Config) (*lock, error) {
	var locked lock
	if err := readJSON(filepath.Join(dir, lockFile), &locked); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if locked.Repository != cfg.Repository {
		return nil, fmt.Errorf("repository differs from %s; restore it or remove the old lock before changing source", lockFile)
	}
	if !commitPattern.MatchString(locked.Commit) || !hashPattern.MatchString(locked.LicenseSHA256) {
		return nil, fmt.Errorf("invalid commit or license checksum in %s", lockFile)
	}
	for name, hash := range locked.Icons {
		if _, err := identifier(name); err != nil || !hashPattern.MatchString(hash) {
			return nil, fmt.Errorf("invalid icon entry %q in %s", name, lockFile)
		}
	}
	return &locked, nil
}

func identifier(name string) (string, error) {
	if !iconNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid icon name %q: use upstream kebab-case names", name)
	}
	parts := strings.Split(name, "-")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	symbol := strings.Join(parts, "")
	if symbol[0] >= '0' && symbol[0] <= '9' {
		symbol = "Icon" + symbol
	}
	if !token.IsIdentifier(symbol) {
		return "", fmt.Errorf("invalid Go identifier for %q", name)
	}
	return symbol, nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target, json.RejectUnknownMembers(true)); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func jsonBytes(value any) []byte {
	data, err := json.Marshal(value, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
