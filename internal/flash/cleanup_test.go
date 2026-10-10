package flash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupRefusesActiveProcesses(t *testing.T) {
	bin := t.TempDir()
	scripts := map[string]string{
		"losetup": "#!/bin/sh\nprintf '%s\\n' '{\"loopdevices\":[{\"name\":\"/dev/loop999\",\"back-file\":\"/bsp/bootloader/system.img\"}]}'\n",
		"pgrep":   "#!/bin/sh\nexit 0\n",
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	err := cleanupStaleImage(context.Background(), "/bsp", func(string) { t.Fatal("must not mutate") })
	if err == nil || !strings.Contains(err.Error(), "processes are active") {
		t.Fatalf("got %v", err)
	}
}

func TestCleanupIgnoresUnrelatedImages(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' '{\"loopdevices\":[{\"name\":\"/dev/loop999\",\"back-file\":\"/other/system.img\"}]}'\n"
	if err := os.WriteFile(filepath.Join(bin, "losetup"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if err := cleanupStaleImage(context.Background(), "/bsp", func(string) { t.Fatal("must not mutate") }); err != nil {
		t.Fatal(err)
	}
}
