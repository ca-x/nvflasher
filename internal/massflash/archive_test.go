package massflash

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nvflasher/internal/tegra"
)

func TestPrepare(t *testing.T) {
	for _, name := range []string{"mfi_orin/tools/kernel_flash/l4t_initrd_flash.sh", "../../escape"} {
		archive := filepath.Join(t.TempDir(), "mfi.tar.gz")
		output, err := os.Create(archive)
		if err != nil {
			t.Fatal(err)
		}
		compressed := gzip.NewWriter(output)
		writer := tar.NewWriter(compressed)
		content := []byte("test")
		if err := writer.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0755, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
		_ = compressed.Close()
		_ = output.Close()
		result, err := Prepare(context.Background(), archive, t.TempDir())
		if name == "../../escape" {
			if err == nil {
				t.Fatal("unsafe archive accepted")
			}
		} else if err != nil || filepath.Base(result) != "mfi_orin" {
			t.Fatalf("result %q, error %v", result, err)
		}
	}
}

func TestGenerateArgumentOrder(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "tools/kernel_flash/l4t_initrd_flash.sh")
	if err := os.MkdirAll(filepath.Dir(tool), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > args.txt\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Generate(context.Background(), dir, tegra.Presets[0].Options, 5, func(string) {}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "--no-flash\n--massflash\n5\njetson-orin-nano-devkit-super\ninternal\n") {
		t.Fatalf("incorrect argument order: %s", data)
	}
}
