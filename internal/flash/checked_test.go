package flash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nvflasher/internal/runner"
)

func TestPackagingFailureStopsIgnoredTarError(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "flash-started")
	err := runChecked(context.Background(), runner.Command{Dir: dir, Args: []string{"sh", "-c", "echo 'tar: ./etc/: Cannot savedir: Bad message' >&2; sleep 2; touch flash-started; echo Success"}}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "packaging failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("flashing stage reached: %v", err)
	}
}

func TestPackagingLineDetection(t *testing.T) {
	for _, line := range []string{"tar: Write checkpoint 10000", "Success", "rpcbind: another rpcbind is already running. Aborting"} {
		if fatalPackagingLine(line) {
			t.Fatalf("false positive: %s", line)
		}
	}
	if !fatalPackagingLine("tar: Exiting with failure status due to previous errors") {
		t.Fatal("missed tar failure")
	}
}
