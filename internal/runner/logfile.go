package runner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var loggingEnabled atomic.Bool

func SetLogging(enabled bool) { loggingEnabled.Store(enabled) }

// commandLog keeps diagnostics beside the executable, not the working directory.
func commandLog() (*slog.Logger, *os.File, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	dir := filepath.Join(filepath.Dir(exe), "logs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, err
	}
	file, err := os.CreateTemp(dir, fmt.Sprintf("command-%s-*.log", time.Now().Format("20060102-150405")))
	if err != nil {
		return nil, nil, err
	}
	return slog.New(slog.NewTextHandler(file, nil)), file, nil
}
