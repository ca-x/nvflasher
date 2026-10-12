package provision

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var timezoneName = regexp.MustCompile(`^[A-Za-z0-9_+-]+(?:/[A-Za-z0-9_+-]+)*$`)
var localeName = regexp.MustCompile(`^[a-z]{2}_[A-Z]{2}\.UTF-8$`)

func systemConfiguration(o Options) (string, error) {
	var script strings.Builder
	script.WriteString("set -eu\nexport DEBIAN_FRONTEND=noninteractive\n")
	if o.AptMirror != "" {
		u, err := url.Parse(o.AptMirror)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.TrimRight(u.Path, "/") != "/ubuntu-ports" || strings.ContainsAny(o.AptMirror, "'\"\\\n\r\t &|;") {
			return "", fmt.Errorf("APT mirror must be an HTTP(S) ubuntu-ports URL without credentials or query")
		}
		// Only ubuntu-ports URLs are replaced, including deb822 URIs. NVIDIA
		// repositories and other third-party sources are deliberately untouched.
		fmt.Fprintf(&script, `found=0
for file in /etc/apt/sources.list /etc/apt/sources.list.d/*.list /etc/apt/sources.list.d/*.sources; do
 [ -f "$file" ] || continue
 [ ! -L "$file" ] || { echo "Refusing symlink APT source: $file" >&2; exit 1; }
 if grep -Eq 'https?://[^[:space:]/]+/ubuntu-ports/?([[:space:]]|$)' "$file"; then
  sed -Ei 's@https?://[^[:space:]/]+/ubuntu-ports/?([[:space:]]|$)@%s\1@g' "$file"
  found=1
 fi
done
[ "$found" = 1 ] || { echo "No Ubuntu ARM64 ubuntu-ports source found; sources unchanged" >&2; exit 1; }
`, strings.TrimRight(o.AptMirror, "/"))
	}
	if o.Timezone != "" {
		if !timezoneName.MatchString(o.Timezone) {
			return "", fmt.Errorf("invalid IANA timezone: %q", o.Timezone)
		}
		fmt.Fprintf(&script, "test -f '/usr/share/zoneinfo/%s' || { echo 'Timezone not available in rootfs' >&2; exit 1; }\nln -snf '/usr/share/zoneinfo/%s' /etc/localtime\nprintf '%%s\\n' '%s' > /etc/timezone\n", o.Timezone, o.Timezone, o.Timezone)
	}
	if o.Locale != "" {
		if !localeName.MatchString(o.Locale) {
			return "", fmt.Errorf("locale must use language_REGION.UTF-8 format")
		}
		fmt.Fprintf(&script, "apt-get update\napt-get install -y --no-install-recommends locales\ngrep -Fxq '%s UTF-8' /usr/share/i18n/SUPPORTED || { echo 'Unsupported locale' >&2; exit 1; }\ngrep -Fxq '%s UTF-8' /etc/locale.gen || printf '%%s\\n' '%s UTF-8' >> /etc/locale.gen\nlocale-gen '%s'\nupdate-locale LANG='%s'\n", o.Locale, o.Locale, o.Locale, o.Locale, o.Locale)
	}
	if o.AptMirror == "" && o.Timezone == "" && o.Locale == "" {
		return "", nil
	}
	return script.String(), nil
}

func appearanceConfiguration(o Options) string {
	var script strings.Builder
	if o.Wallpaper == "" && o.BootLogo == "" {
		return ""
	}
	script.WriteString("set -eu\nexport DEBIAN_FRONTEND=noninteractive\n")
	if o.Wallpaper != "" {
		script.WriteString("command -v dconf >/dev/null || { echo 'GNOME wallpaper requires dconf-cli in rootfs' >&2; exit 1; }\ndconf update\n")
	}
	if o.BootLogo != "" {
		script.WriteString(`command -v plymouth >/dev/null && command -v update-initramfs >/dev/null || { echo 'Boot logo requires Plymouth and initramfs-tools in rootfs' >&2; exit 1; }
found=0
for initrd in /boot/initrd.img-*; do [ ! -f "$initrd" ] || found=1; done
[ "$found" = 1 ] || { echo 'No supported initrd.img-* found; Jetson BSP boot integration requires manual configuration' >&2; exit 1; }
update-alternatives --install /usr/share/plymouth/themes/default.plymouth default.plymouth /usr/share/plymouth/themes/nvflasher/nvflasher.plymouth 100
update-alternatives --set default.plymouth /usr/share/plymouth/themes/nvflasher/nvflasher.plymouth
update-initramfs -u -k all
echo 'Linux boot logo configured; boot arguments must enable splash. UEFI/NVIDIA firmware logo is unchanged.'
`)
	}
	return script.String()
}
