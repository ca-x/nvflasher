package flash

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchHost(t *testing.T) {
	for _, input := range []string{"ID=arch\n", "ID=omarchy\nID_LIKE=arch\n", "ID=endeavouros\nID_LIKE=arch\n"} {
		if !isArch(input) {
			t.Errorf("Arch host not detected: %q", input)
		}
	}
	if isArch("ID=ubuntu\nID_LIKE=debian\n") {
		t.Fatal("Debian host detected as Arch")
	}
}

func TestArchServiceAdapterOnlyHandlesNFS(t *testing.T) {
	directory, err := serviceAdapter()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	bin := t.TempDir()
	result := filepath.Join(bin, "result")
	systemctl := "#!/bin/sh\nprintf '%s %s' \"$1\" \"$2\" > \"$RESULT\"\n"
	if err := os.WriteFile(filepath.Join(bin, "systemctl"), []byte(systemctl), 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(directory, "service"), "nfs-kernel-server", "restart")
	command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "RESULT="+result)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("service adapter: %v: %s", err, output)
	}
	if data, err := os.ReadFile(result); err != nil || string(data) != "restart nfs-server.service" {
		t.Fatalf("unexpected systemctl call: %q, %v", data, err)
	}
	command = exec.Command(filepath.Join(directory, "service"), "rpcbind", "restart")
	command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "RESULT="+result)
	if output, err := command.CombinedOutput(); err == nil || !strings.Contains(string(output), "Unsupported NVIDIA service call") {
		t.Fatalf("unsupported service accepted: %v: %s", err, output)
	}
}
