package generator

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Run 执行图标生成命令；选项须放在命令之前，未指定目录时使用最近的项目中的 pkg/icons。
func Run(ctx context.Context, args []string, client *http.Client, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	flags := flag.NewFlagSet("icongen", flag.ContinueOnError)
	output := &recordingWriter{writer: out}
	flags.SetOutput(output)
	dir := flags.String("dir", "", "icon package directory (default: module-root/pkg/icons)")
	flags.Usage = func() {
		if _, err := fmt.Fprintln(output, "Usage: icongen [-dir DIRECTORY] add NAME... | sync | update [REF] | generate | check"); err == nil {
			flags.PrintDefaults()
		}
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return output.err
		}
		return errors.Join(err, output.err)
	}
	args = flags.Args()
	if len(args) == 0 {
		flags.Usage()
		return errors.Join(fmt.Errorf("a command is required"), output.err)
	}
	if *dir == "" {
		root, err := moduleRoot()
		if err != nil {
			return err
		}
		*dir = filepath.Join(root, "pkg", "icons")
	}
	g, err := New(Options{Directory: *dir, HTTPClient: client, Output: out})
	if err != nil {
		return err
	}
	command := args[0]
	switch command {
	case "generate", "check", "sync":
		if len(args) != 1 {
			return fmt.Errorf("%s takes no arguments", command)
		}
		switch command {
		case "generate":
			return g.Generate()
		case "check":
			return g.Check()
		default:
			return g.Sync(ctx)
		}
	case "add":
		return g.Add(ctx, args[1:]...)
	case "update":
		if len(args) > 2 {
			return fmt.Errorf("usage: update [REF]")
		}
		ref := ""
		if len(args) == 2 {
			ref = args[1]
		}
		return g.Update(ctx, ref)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found; supply -dir")
		}
		dir = parent
	}
}

type recordingWriter struct {
	writer io.Writer
	err    error
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = errors.Join(w.err, err)
	}
	return n, err
}
