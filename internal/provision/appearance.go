package provision

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

func validateAppearance(o Options) error {
	for _, asset := range []struct {
		path    string
		pngOnly bool
	}{{o.Wallpaper, false}, {o.BootLogo, true}} {
		if asset.path == "" {
			continue
		}
		f, err := os.Open(asset.path)
		if err != nil {
			return fmt.Errorf("open appearance image: %w", err)
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 32*1024*1024 {
			f.Close()
			return fmt.Errorf("appearance image must be a regular file of at most 32 MiB")
		}
		config, format, err := image.DecodeConfig(f)
		f.Close()
		if err != nil || (format != "png" && format != "jpeg") || (asset.pngOnly && format != "png") || config.Width <= 0 || config.Height <= 0 || config.Width > 8192 || config.Height > 8192 {
			return fmt.Errorf("invalid appearance image %q: use PNG/JPEG up to 8192x8192 (boot logo requires PNG)", asset.path)
		}
	}
	return nil
}

// Make one directory at a time so symlink ancestors cannot redirect writes.
func appearanceDirectory(root, relative string) error {
	current := root
	for _, part := range strings.Split(relative, "/") {
		current = filepath.Join(current, part)
		if err := safeTarget(root, current); err != nil {
			return err
		}
		if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Stat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", current)
		}
	}
	return nil
}

func appearanceFile(root, relative string, data []byte) error {
	if err := appearanceDirectory(root, filepath.Dir(relative)); err != nil {
		return err
	}
	dest := filepath.Join(root, relative)
	if err := safeTarget(root, dest); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

func installAppearance(root string, o Options) error {
	if o.Wallpaper != "" {
		data, err := os.ReadFile(o.Wallpaper)
		if err != nil {
			return err
		}
		if err := appearanceFile(root, "usr/share/backgrounds/nvflasher-wallpaper", data); err != nil {
			return err
		}
		defaults := "[org/gnome/desktop/background]\npicture-uri='file:///usr/share/backgrounds/nvflasher-wallpaper'\npicture-uri-dark='file:///usr/share/backgrounds/nvflasher-wallpaper'\npicture-options='zoom'\n\n[org/gnome/desktop/screensaver]\npicture-uri='file:///usr/share/backgrounds/nvflasher-wallpaper'\n"
		if err := appearanceFile(root, "etc/dconf/db/nvflasher.d/00-background", []byte(defaults)); err != nil {
			return err
		}
		profile := filepath.Join(root, "etc/dconf/profile/user")
		if err := appearanceDirectory(root, "etc/dconf/profile"); err != nil {
			return err
		}
		if err := safeTarget(root, profile); err != nil {
			return err
		}
		data, err = os.ReadFile(profile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if len(data) == 0 {
			data = []byte("user-db:user\n")
		}
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "system-db:nvflasher" {
				found = true
			}
		}
		if !found {
			data = append(data, []byte("\nsystem-db:nvflasher\n")...)
			if err := os.WriteFile(profile, data, 0644); err != nil {
				return err
			}
		}
	}
	if o.BootLogo != "" {
		data, err := os.ReadFile(o.BootLogo)
		if err != nil {
			return err
		}
		if err := appearanceFile(root, "usr/share/plymouth/themes/nvflasher/logo.png", data); err != nil {
			return err
		}
		theme := "[Plymouth Theme]\nName=nvflasher\nDescription=Custom Linux boot logo\nModuleName=script\n\n[script]\nImageDir=/usr/share/plymouth/themes/nvflasher\nScriptFile=/usr/share/plymouth/themes/nvflasher/nvflasher.script\n"
		if err := appearanceFile(root, "usr/share/plymouth/themes/nvflasher/nvflasher.plymouth", []byte(theme)); err != nil {
			return err
		}
		script := "image = Image(\"logo.png\");\nsprite = Sprite(image);\nsprite.SetX(Window.GetWidth() / 2 - image.GetWidth() / 2);\nsprite.SetY(Window.GetHeight() / 2 - image.GetHeight() / 2);\n"
		if err := appearanceFile(root, "usr/share/plymouth/themes/nvflasher/nvflasher.script", []byte(script)); err != nil {
			return err
		}
	}
	return nil
}
