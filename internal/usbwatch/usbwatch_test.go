package usbwatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	for _, c := range []struct{ vendor, product, want string }{{"0955", "7523", "recovery"}, {"0955", "7f23", "recovery"}, {"0955", "7035", "initrd"}, {"0955", "7023", "recovery"}, {"0001", "7523", ""}, {"0955", "7g23", ""}} {
		if got := Parse(c.vendor, c.product); got != c.want {
			t.Errorf("%q:%q got %q want %q", c.vendor, c.product, got, c.want)
		}
	}
}
func TestScan(t *testing.T) {
	dir := t.TempDir()
	device := filepath.Join(dir, "1-1")
	if err := os.Mkdir(device, 0755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"idVendor": "0955\n", "idProduct": "7523\n"} {
		if err := os.WriteFile(filepath.Join(device, name), []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	found := Scan(dir)
	if len(found) != 1 || found[0].Mode != "recovery" {
		t.Fatalf("%v", found)
	}
}
