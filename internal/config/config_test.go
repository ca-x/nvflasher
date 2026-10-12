package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyProvisionDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"provision":{"username":"dev"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Provision == nil || *got.Provision != (ProvisionForm{Username: "dev"}) {
		t.Fatalf("unexpected legacy defaults: %+v", got.Provision)
	}
}

func TestSaveLoad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	original := Config{L4T: "/opt/Linux_for_Tegra", LogLimit: 5000, Proxy: Proxy{Enabled: true, URL: "socks5://secret:pass@127.0.0.1:1080"}, Provision: &ProvisionForm{ChineseSupport: true, Suspend: "disable", ScreenLock: "enable", ScreenBlank: "disable", Autostart: "/opt/myapp/myapp", DebFiles: "/tmp/application.deb\n/tmp/dependency.deb", Timezone: "Asia/Shanghai", Locale: "zh_CN.UTF-8", AptMirror: "https://mirror.example/ubuntu-ports", Wallpaper: "/tmp/wallpaper.png", BootLogo: "/tmp/logo.png"}}
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
	if got.Provision == nil || *got.Provision != *original.Provision {
		t.Fatalf("provision settings did not roundtrip: %+v", got.Provision)
	}
	if got.L4T != original.L4T {
		t.Fatalf("roundtrip failed: %+v", got)
	}
}
