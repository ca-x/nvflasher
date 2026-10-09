package massflash

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Prepare(ctx context.Context, archive, workspace string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(archive), ".tar.gz") {
		return archive, nil
	}
	if err := os.MkdirAll(workspace, 0700); err != nil {
		return "", err
	}
	root, err := os.MkdirTemp(workspace, "mfi-")
	if err != nil {
		return "", err
	}
	valid := false
	defer func() {
		if !valid {
			_ = os.RemoveAll(root)
		}
	}()
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	type link struct{ path, target string }
	var links []link
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		name := filepath.Clean(header.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("archive contains unsafe path: %s", header.Name)
		}
		if name == "." {
			continue
		}
		destination := filepath.Join(root, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(destination, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
				return "", err
			}
			output, createErr := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, header.FileInfo().Mode().Perm())
			if createErr != nil {
				return "", createErr
			}
			_, err = io.Copy(output, &cancellableReader{ctx: ctx, reader: reader})
			closeErr := output.Close()
			if err != nil {
				return "", err
			}
			if closeErr != nil {
				return "", closeErr
			}
		case tar.TypeSymlink:
			target := filepath.Clean(filepath.Join(filepath.Dir(name), header.Linkname))
			if filepath.IsAbs(header.Linkname) || target == ".." || strings.HasPrefix(target, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("archive contains unsafe symlink: %s", name)
			}
			links = append(links, link{destination, header.Linkname})
		default:
			return "", fmt.Errorf("archive contains unsupported file type: %s", name)
		}
	}
	for _, item := range links {
		if err := os.MkdirAll(filepath.Dir(item.path), 0755); err != nil {
			return "", err
		}
		if err := os.Symlink(item.target, item.path); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(filepath.Join(root, "tools/kernel_flash/l4t_initrd_flash.sh")); err == nil {
		valid = true
		return root, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		candidate := filepath.Join(root, entries[0].Name())
		if _, err := os.Stat(filepath.Join(candidate, "tools/kernel_flash/l4t_initrd_flash.sh")); err == nil {
			valid = true
			return candidate, nil
		}
	}
	return "", fmt.Errorf("archive does not contain an extracted MFI flash tool")
}

type cancellableReader struct {
	ctx    context.Context
	reader io.Reader
}

func (source *cancellableReader) Read(buffer []byte) (int, error) {
	if err := source.ctx.Err(); err != nil {
		return 0, err
	}
	return source.reader.Read(buffer)
}
