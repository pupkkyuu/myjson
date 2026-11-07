package generator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	Directory  string
	HTTPClient *http.Client
	Output     io.Writer
}

type Generator struct {
	dir    string
	client *http.Client
	out    io.Writer
}

// New 创建生成器并解析目标目录，不读取文件或访问网络。
func New(options Options) (*Generator, error) {
	if options.Directory == "" {
		return nil, fmt.Errorf("icon package directory is required")
	}
	dir, err := filepath.Abs(options.Directory)
	if err != nil {
		return nil, err
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	out := options.Output
	if out == nil {
		out = io.Discard
	}
	return &Generator{dir: dir, client: client, out: out}, nil
}

// Init 初始化图标包；已有清单时拒绝覆盖，网络或校验失败时不修改目录。
// 空图标集合须使用空切片而非 nil。
func (g *Generator) Init(ctx context.Context, cfg Config) error {
	if _, err := os.Stat(filepath.Join(g.dir, configFile)); err == nil {
		return fmt.Errorf("%s already exists; use Sync, Add or Update", configFile)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if cfg.Icons != nil {
		cfg.Icons = append([]string{}, cfg.Icons...)
	}
	return g.synchronize(ctx, cfg, false, cfg.Ref)
}

// Sync 根据锁定的提交恢复资源并生成访问函数；没有锁文件时解析并锁定配置的引用。
func (g *Generator) Sync(ctx context.Context) error {
	cfg, err := readConfig(g.dir)
	if err != nil {
		return err
	}
	return g.synchronize(ctx, cfg, false, cfg.Ref)
}

// Add 在锁定的提交中添加图标，忽略已有名称，并在所有资源校验通过后更新清单。
func (g *Generator) Add(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return fmt.Errorf("add requires at least one upstream icon name, e.g. arrow-up")
	}
	cfg, err := readConfig(g.dir)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(cfg.Icons))
	for _, name := range cfg.Icons {
		seen[name] = true
	}
	for _, name := range names {
		if !seen[name] {
			cfg.Icons = append(cfg.Icons, name)
			seen[name] = true
		}
	}
	return g.synchronize(ctx, cfg, false, cfg.Ref)
}

// Update 将选中资源更新到指定引用对应的提交；空引用使用清单中的 Ref。
func (g *Generator) Update(ctx context.Context, ref string) error {
	cfg, err := readConfig(g.dir)
	if err != nil {
		return err
	}
	if ref == "" {
		ref = cfg.Ref
	}
	return g.synchronize(ctx, cfg, true, ref)
}

// Generate 离线校验内嵌 SVG 后重新生成代码；源数据缺失或损坏时须先运行 Sync。
func (g *Generator) Generate() error {
	return g.generate(false)
}

// Check 离线校验内嵌 SVG、许可证和生成代码，不写入文件。
func (g *Generator) Check() error {
	return g.generate(true)
}

func (g *Generator) generate(check bool) error {
	cfg, err := readConfig(g.dir)
	if err != nil {
		return err
	}
	return generateLocal(g.dir, cfg, check, g.out)
}

func (g *Generator) synchronize(ctx context.Context, cfg Config, update bool, ref string) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	return synchronize(ctx, g.dir, cfg, remote{client: g.client}, update, ref, g.out)
}
