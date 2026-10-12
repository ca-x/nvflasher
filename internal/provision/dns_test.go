package provision

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChrootDNSRestoration(t *testing.T) {
	for _, kind := range []string{"missing", "regular", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			etc := filepath.Join(root, "etc")
			if err := os.Mkdir(etc, 0755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(etc, "resolv.conf")
			switch kind {
			case "regular":
				if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("/run/systemd/resolve/stub-resolv.conf", target); err != nil {
					t.Fatal(err)
				}
			}
			restore, err := replaceChrootDNS(root, []byte("nameserver 1.1.1.1\n"))
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "nameserver 1.1.1.1\n" {
				t.Fatalf("temporary DNS: %q, %v", data, err)
			}
			if err := restore(); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing":
				if _, err := os.Lstat(target); !os.IsNotExist(err) {
					t.Fatalf("expected absent entry: %v", err)
				}
			case "regular":
				data, err := os.ReadFile(target)
				if err != nil || string(data) != "original" {
					t.Fatalf("original DNS: %q, %v", data, err)
				}
				info, err := os.Stat(target)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0600 {
					t.Fatal("original permissions lost")
				}
			case "symlink":
				link, err := os.Readlink(target)
				if err != nil || link != "/run/systemd/resolve/stub-resolv.conf" {
					t.Fatalf("original symlink: %q, %v", link, err)
				}
			}
			entries, err := os.ReadDir(etc)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != "resolv.conf" {
					t.Fatalf("backup leaked: %s", entry.Name())
				}
			}
		})
	}
}

func TestChrootDNSRejectsEscapingParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "etc")); err != nil {
		t.Fatal(err)
	}
	if _, err := replaceChrootDNS(root, []byte("host")); err == nil {
		t.Fatal("accepted escaping parent")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("modified host directory")
	}
}
