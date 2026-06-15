package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type CommandSpec struct {
	Path string
	Args []string
	Env  []string
}

func runCommand(ctx context.Context, spec CommandSpec, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Env = spec.Env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	return cmd.Run()
}

func writeCommandOutput(ctx Context, spec CommandSpec, artifactPath string, compress bool) error {
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer io.Writer = file
	var gz *gzip.Writer
	if compress {
		gz = gzip.NewWriter(file)
		writer = gz
	}
	ctx.Logf("run: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if err := runCommand(context.Background(), spec, writer, os.Stderr, nil); err != nil {
		if gz != nil {
			_ = gz.Close()
		}
		return err
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			return err
		}
	}
	return file.Sync()
}

func commandInputFromArtifact(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		return readCloser{Reader: gz, close: func() error {
			err1 := gz.Close()
			err2 := file.Close()
			if err1 != nil {
				return err1
			}
			return err2
		}}, nil
	}
	return file, nil
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r readCloser) Close() error {
	return r.close()
}

func maskDumpArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--password="):
			out[i] = "--password=***"
		case strings.HasPrefix(arg, "MYSQL_PWD="):
			out[i] = "MYSQL_PWD=***"
		default:
			out[i] = arg
		}
	}
	return out
}

func boolArg(enabled bool, arg string) []string {
	if enabled {
		return []string{arg}
	}
	return nil
}

func portArg(port int) string {
	return strconv.Itoa(port)
}

func ensureDir(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	return os.MkdirAll(path, 0o755)
}
