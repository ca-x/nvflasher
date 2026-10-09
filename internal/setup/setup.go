package setup

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nvflasher/internal/download"
	"nvflasher/internal/runner"
	"nvflasher/internal/tegra"
)

func verify(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected) {
		return fmt.Errorf("SHA1 mismatch for %s", filepath.Base(path))
	}
	return nil
}

func Download(ctx context.Context, modelID, directory, proxyURL string, emit func(string)) error {
	model, err := Lookup(modelID)
	if err != nil {
		return err
	}
	if directory == "" {
		return fmt.Errorf("choose a download directory")
	}
	for _, pkg := range model.Packages {
		path := filepath.Join(directory, pkg.Name)
		if _, err := os.Stat(path); err == nil {
			if err := verify(path, pkg.SHA1); err != nil {
				return err
			}
			emit("Verified cached archive: " + path)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		emit("Downloading " + pkg.Name)
		lastPercent := int64(-1)
		err := download.Start(ctx, download.Request{URL: pkg.URL, Directory: directory, Filename: pkg.Name, ProxyURL: proxyURL}, func(progress download.Progress) {
			if progress.Total > 0 {
				percent := progress.Bytes * 100 / progress.Total
				if percent != lastPercent {
					lastPercent = percent
					emit(fmt.Sprintf("%s: %d%%", pkg.Name, percent))
				}
			}
		})
		if err != nil {
			return err
		}
		if err := verify(path, pkg.SHA1); err != nil {
			_ = os.Remove(path)
			return err
		}
		emit("Verified SHA1: " + pkg.Name)
	}
	return nil
}

func Location(modelID, directory string) (string, error) {
	model, err := Lookup(modelID)
	if err != nil {
		return "", err
	}
	if directory == "" {
		return "", fmt.Errorf("choose a download directory")
	}
	return filepath.Join(directory, "r"+model.Release, "Linux_for_Tegra"), nil
}

func Prepare(ctx context.Context, modelID, directory string, emit func(string)) (string, error) {
	model, err := Lookup(modelID)
	if err != nil {
		return "", err
	}
	return prepare(ctx, model, directory, emit)
}

func prepare(ctx context.Context, model Model, directory string, emit func(string)) (string, error) {
	if directory == "" {
		return "", fmt.Errorf("choose a download directory")
	}
	for _, pkg := range model.Packages {
		if err := verify(filepath.Join(directory, pkg.Name), pkg.SHA1); err != nil {
			return "", fmt.Errorf("verify %s: %w", pkg.Name, err)
		}
	}
	l4t := filepath.Join(directory, "r"+model.Release, "Linux_for_Tegra")
	workspace := filepath.Dir(l4t)
	report := tegra.Inspect(l4t)
	if strings.Contains(report.Version, "R36 (release), REVISION: 5.0") {
		ready := true
		for _, check := range report.Checks {
			ready = ready && check.Status == "ok"
		}
		if ready {
			return l4t, nil
		}
	}
	if _, err := exec.LookPath("dpkg"); err != nil {
		return "", fmt.Errorf("NVIDIA BSP preparation needs dpkg on the host; install it from the Environment page before retrying: %w", err)
	}
	if entries, err := os.ReadDir(workspace); err == nil && len(entries) != 0 && (len(entries) != 1 || entries[0].Name() != "Linux_for_Tegra") {
		return "", fmt.Errorf("workspace %s contains unexpected files; select another directory", workspace)
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	flashScript := filepath.Join(l4t, "tools/kernel_flash/l4t_initrd_flash.sh")
	rootfs := filepath.Join(l4t, "rootfs")
	bspExists := fileExists(flashScript)
	rootfsExists := fileExists(filepath.Join(rootfs, "etc/os-release"))
	if !bspExists {
		if _, err := os.Stat(l4t); err == nil {
			return "", fmt.Errorf("incomplete BSP extraction at %s; select a new download directory", l4t)
		}
	}
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return "", err
	}
	if !bspExists {
		emit("Extracting NVIDIA BSP (includes flash tools)")
		if err := extract(ctx, filepath.Join(directory, model.Packages[0].Name), workspace, "BSP", emit); err != nil {
			return "", err
		}
	} else {
		emit("Reusing extracted NVIDIA BSP")
	}
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return "", err
	}
	if !rootfsExists {
		emit("Extracting sample rootfs")
		if err := extract(ctx, filepath.Join(directory, model.Packages[1].Name), rootfs, "rootfs", emit); err != nil {
			return "", err
		}
	} else {
		emit("Reusing extracted sample rootfs")
	}
	emit("Applying NVIDIA binaries")
	if err := runner.Run(ctx, runner.Command{Dir: l4t, Args: []string{"./apply_binaries.sh"}, Env: []string{"PATH=/usr/local/sbin:/usr/sbin:/sbin:" + os.Getenv("PATH")}}, emit); err != nil {
		return "", err
	}
	for _, check := range tegra.Inspect(l4t).Checks {
		if check.Status != "ok" {
			return "", fmt.Errorf("NVIDIA BSP preparation incomplete: %s: %s", check.Name, check.Hint)
		}
	}
	return l4t, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
