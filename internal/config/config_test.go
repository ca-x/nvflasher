package config

import (
	"os"
	"strings"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	original := Config{L4T: "/opt/Linux_for_Tegra", LogLimit: 5000, Proxy: Proxy{Enabled: true, URL: "socks5://secret:pass@127.0.0.1:1080"}}
	if err := Save(original); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Proxy.URL != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy credentials persisted: %q", got.Proxy.URL)
	}
	path, _ := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret") {
		t.Fatal("credentials leaked")
	}
	if got.L4T != original.L4T {
		t.Fatalf("roundtrip failed: %+v", got)
	}
}
