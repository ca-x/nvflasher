package provision

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppearanceInstallation(t *testing.T) {
	root := t.TempDir()
	asset := filepath.Join(t.TempDir(), "test.png")
	f, err := os.Create(asset)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	o := Options{Wallpaper: asset, BootLogo: asset}
	if err := validateAppearance(o); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := installAppearance(root, o); err != nil {
			t.Fatal(err)
		}
	}
	profile, err := os.ReadFile(filepath.Join(root, "etc/dconf/profile/user"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(profile), "system-db:nvflasher") != 1 {
		t.Fatalf("profile duplicated: %s", profile)
	}
	for _, name := range []string{"usr/share/backgrounds/nvflasher-wallpaper", "etc/dconf/db/nvflasher.d/00-background", "usr/share/plymouth/themes/nvflasher/logo.png", "usr/share/plymouth/themes/nvflasher/nvflasher.plymouth"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAppearanceRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "usr")); err != nil {
		t.Fatal(err)
	}
	if err := appearanceFile(root, "usr/share/backgrounds/image", []byte("image")); err == nil {
		t.Fatal("accepted symlink ancestor")
	}
	files, err := os.ReadDir(outside)
	if err != nil || len(files) != 0 {
		t.Fatal("outside directory modified")
	}
}

func TestAppearanceRejectsInvalidImage(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(file, []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateAppearance(Options{BootLogo: file}); err == nil {
		t.Fatal("accepted invalid image")
	}
}
