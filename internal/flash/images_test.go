package flash

import (
	"path/filepath"
	"testing"
)

func TestRejectAttachedBSPImage(t *testing.T) {
	dir := t.TempDir()
	for _, image := range []string{"bootloader/system.img.raw", "bootloader/system.img"} {
		if err := rejectImageLoops(dir, filepath.Join(dir, image)+"\n"); err == nil {
			t.Fatal("accepted attached image")
		}
	}
	for _, output := range []string{"", dir + "-other/system.img.raw\n", "/unrelated/disk.img\n"} {
		if err := rejectImageLoops(dir, output); err != nil {
			t.Fatal(err)
		}
	}
}
