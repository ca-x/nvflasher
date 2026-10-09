package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"nvflasher/internal/tegra"
)

type Proxy struct {
	Enabled             bool   `json:"enabled"`
	URL                 string `json:"url"`
	RememberCredentials bool   `json:"rememberCredentials"`
}
type Config struct {
	L4T         string         `json:"l4t"`
	Workspace   string         `json:"workspace"`
	DownloadDir string         `json:"downloadDir"`
	LogLimit    int            `json:"logLimit"`
	Proxy       Proxy          `json:"proxy"`
	Flash       *tegra.Options `json:"flash,omitempty"`
	Provision   *ProvisionForm `json:"provision,omitempty"`
}

type ProvisionForm struct {
	Username       string `json:"username"`
	Hostname       string `json:"hostname"`
	Autologin      bool   `json:"autologin"`
	PublicKey      string `json:"publicKey"`
	Packages       string `json:"packages"`
	Script         string `json:"script"`
	Overlay        string `json:"overlay"`
	ClearKeys      bool   `json:"clearKeys"`
	ClearMachineID bool   `json:"clearMachineId"`
	FirstBoot      bool   `json:"firstBoot"`
}

func Path() (string, error) {
	base, err := os.UserConfigDir()
	return filepath.Join(base, "nvflasher", "config.json"), err
}
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	c := Config{LogLimit: 5000}
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		c.DownloadDir = filepath.Join(home, "Downloads")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	if c.LogLimit < 100 {
		c.LogLimit = 5000
	}
	return c, err
}
func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if !c.Proxy.RememberCredentials {
		if i := strings.Index(c.Proxy.URL, "@"); i >= 0 {
			if j := strings.Index(c.Proxy.URL, "://"); j >= 0 {
				c.Proxy.URL = c.Proxy.URL[:j+3] + c.Proxy.URL[i+1:]
			}
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
