package flash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupMountedImageSafety(t *testing.T) {
	for _, busy := range []bool{false, true} {
		t.Run(map[bool]string{false: "idle", true: "busy"}[busy], func(t *testing.T) {
			bin := t.TempDir()
			trace := filepath.Join(bin, "trace")
			fuser := "exit 1"
			if busy {
				fuser = "echo '1234'; exit 0"
			}
			scripts := map[string]string{
				"losetup": "case \"$1\" in --json) echo '{\"loopdevices\":[{\"name\":\"/dev/loop999\",\"back-file\":\"/bsp/bootloader/system.img\"}]}' ;; --noheadings) echo '/bsp/bootloader/system.img' ;; --detach) echo detach >> \"$TRACE\" ;; *) exit 2 ;; esac",
				"findmnt": "echo '/bsp/bootloader/mnt'",
				"pgrep":   "exit 1",
				"fuser":   fuser,
				"umount":  "echo unmount >> \"$TRACE\"",
			}
			for name, script := range scripts {
				if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+script+"\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)
			t.Setenv("TRACE", trace)
			err := cleanupStaleImage(context.Background(), "/bsp", func(string) {})
			data, _ := os.ReadFile(trace)
			if busy {
				if err == nil || len(data) != 0 {
					t.Fatalf("unsafe cleanup: %v %s", err, data)
				}
			} else if err != nil || strings.TrimSpace(string(data)) != "unmount\ndetach" {
				t.Fatalf("cleanup: %v %s", err, data)
			}
		})
	}
}
