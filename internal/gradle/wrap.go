package gradle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const DefaultTimeout = 15 * time.Minute

// FindWrapper walks up from cwd looking for ./gradlew (or gradlew.bat
// on Windows). Returns the absolute path or an error.
func FindWrapper(cwd string) (string, error) {
	name := "gradlew"
	if runtime.GOOS == "windows" {
		name = "gradlew.bat"
	}
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("./gradlew not found in this or any parent directory")
}

// Run executes gradlew with args. Stdout and stderr are streamed to the
// caller's writers in real time. Honours `timeout` (defaults to
// DefaultTimeout).
func Run(ctx context.Context, wrapper string, args []string, stdout, stderr io.Writer, timeout time.Duration) error {
	return RunWithEnv(ctx, wrapper, args, nil, stdout, stderr, timeout)
}

func RunWithEnv(ctx context.Context, wrapper string, args []string, extraEnv []string, stdout, stderr io.Writer, timeout time.Duration) error {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, wrapper, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = filepath.Dir(wrapper)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gradlew timed out after %s", timeout)
		}
		return err
	}
	return nil
}
