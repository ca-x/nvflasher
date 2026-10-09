package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"log"
	"nvflasher/internal/capability"
	"nvflasher/internal/config"
	"nvflasher/internal/download"
	"nvflasher/internal/flash"
	"nvflasher/internal/massflash"
	"nvflasher/internal/provision"
	"nvflasher/internal/runner"
	"nvflasher/internal/sshx"
	"nvflasher/internal/tegra"
	"nvflasher/internal/usbwatch"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Environment struct{}

func (Environment) Detect(dir string) tegra.Report       { return tegra.Inspect(dir) }
func (Environment) Capabilities() capability.Status      { return capability.Detect() }
func (Environment) GetConfig() (config.Config, error)    { return config.Load() }
func (Environment) SaveConfig(value config.Config) error { return config.Save(value) }
func (Environment) Presets() []tegra.Preset              { return tegra.Presets }
func (Environment) Browse(ctx context.Context, directory bool) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Directory: directory, Title: "Choose path"})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}
func (Environment) RestartAsRoot() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Linux only")
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	return exec.Command("pkexec", self).Start()
}

type Devices struct{}

var DevicesChanged = mygo.NewEvent[[]usbwatch.Device]("devices:changed")

func (Devices) List() []usbwatch.Device {
	if !capability.Detect().Host {
		return []usbwatch.Device{}
	}
	return usbwatch.Scan("/sys/bus/usb/devices")
}
func (d Devices) StartWatch(ctx context.Context) {
	window := mygo.CallerWindow(ctx)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := DevicesChanged.Emit(window, d.List()); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type operation struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (o *operation) begin(ctx context.Context) (context.Context, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancel != nil {
		return nil, fmt.Errorf("an operation is already running")
	}
	next, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	return next, nil
}
func (o *operation) end() { o.mu.Lock(); o.cancel(); o.cancel = nil; o.mu.Unlock() }
func (o *operation) Cancel() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancel != nil {
		o.cancel()
	}
}
func requireHost() error {
	state := capability.Detect()
	if !state.Host {
		return fmt.Errorf("flashing requires a Linux x86_64 host")
	}
	if !state.Root {
		return fmt.Errorf("root permissions required; start with sudo or pkexec")
	}
	return nil
}
func requireReady(dir string) error {
	if err := requireHost(); err != nil {
		return err
	}
	for _, item := range tegra.Inspect(dir).Checks {
		if item.Status == "error" {
			return fmt.Errorf("%s: %s", item.Name, item.Hint)
		}
	}
	return nil
}

type Flash struct{ operation }

func (f *Flash) Start(ctx context.Context, dir string, opts tegra.Options, lines *mygo.Channel[string]) error {
	if err := requireReady(dir); err != nil {
		return err
	}
	found := false
	for _, device := range usbwatch.Scan("/sys/bus/usb/devices") {
		if device.Mode == "recovery" {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("connect a Recovery-mode Jetson before flashing")
	}
	ctx, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer f.end()
	return flash.Run(ctx, dir, opts, func(line string) { _ = lines.Send(line) })
}

type Massflash struct{ operation }

func (m *Massflash) Generate(ctx context.Context, dir string, opts tegra.Options, count int, lines *mygo.Channel[string]) error {
	if err := requireReady(dir); err != nil {
		return err
	}
	ctx, err := m.begin(ctx)
	if err != nil {
		return err
	}
	defer m.end()
	return massflash.Generate(ctx, dir, opts, count, func(line string) { _ = lines.Send(line) })
}
func (m *Massflash) Flash(ctx context.Context, dir string, count int, show bool, lines *mygo.Channel[string]) error {
	if err := requireHost(); err != nil {
		return err
	}
	recovery := false
	for _, device := range usbwatch.Scan("/sys/bus/usb/devices") {
		if device.Mode == "recovery" {
			recovery = true
		}
	}
	if !recovery {
		return fmt.Errorf("no Recovery devices detected")
	}
	ctx, err := m.begin(ctx)
	if err != nil {
		return err
	}
	defer m.end()
	if filepath.Ext(dir) == ".gz" {
		settings, loadErr := config.Load()
		if loadErr != nil {
			return loadErr
		}
		workspace := settings.Workspace
		if workspace == "" {
			workspace, loadErr = os.UserCacheDir()
			if loadErr != nil {
				return loadErr
			}
			workspace = filepath.Join(workspace, "nvflasher")
		}
		dir, loadErr = massflash.Prepare(ctx, dir, workspace)
		if loadErr != nil {
			return loadErr
		}
		_ = lines.Send("MFI extracted to " + dir)
	}
	return massflash.Flash(ctx, dir, count, show, func(line string) { _ = lines.Send(line) })
}

type Provision struct{ operation }

func (p *Provision) Run(ctx context.Context, dir string, opts provision.Options, lines *mygo.Channel[string]) error {
	if err := requireReady(dir); err != nil {
		return err
	}
	ctx, err := p.begin(ctx)
	if err != nil {
		return err
	}
	defer p.end()
	return provision.Run(ctx, dir, opts, func(line string) { _ = lines.Send(runner.Redact(line)) })
}
func (p *Provision) GeneratePassword() (string, error) { return provision.GeneratePassword() }

type Template struct {
	Name    string `json:"name"`
	Script  string `json:"script"`
	BuiltIn bool   `json:"builtIn"`
}

var examples = []Template{{"base-packages", "apt-get update && apt-get install -y --no-install-recommends curl htop ca-certificates", true}, {"timezone-locale", "ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime\nDEBIAN_FRONTEND=noninteractive dpkg-reconfigure tzdata\nlocale-gen en_US.UTF-8", true}, {"dev-tools", "apt-get update && apt-get install -y --no-install-recommends git build-essential python3", true}}

func templateDir() (string, error) {
	root, err := os.UserConfigDir()
	return filepath.Join(root, "nvflasher", "templates"), err
}
func templatePath(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("invalid template name")
	}
	root, err := templateDir()
	return filepath.Join(root, name+".sh"), err
}
func (*Provision) ListTemplates() ([]Template, error) {
	result := append([]Template{}, examples...)
	dir, err := templateDir()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sh" {
			data, e := os.ReadFile(filepath.Join(dir, file.Name()))
			if e != nil {
				return nil, e
			}
			result = append(result, Template{filepath.Base(file.Name()[:len(file.Name())-3]), string(data), false})
		}
	}
	return result, nil
}
func (*Provision) SaveTemplate(name, script string) error {
	for _, item := range examples {
		if item.Name == name {
			return fmt.Errorf("built-in template is read-only")
		}
	}
	path, err := templatePath(name)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(script), 0600)
}
func (p *Provision) LoadTemplate(name string) (Template, error) {
	items, err := p.ListTemplates()
	if err != nil {
		return Template{}, err
	}
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}
	return Template{}, fmt.Errorf("template not found")
}
func (*Provision) DeleteTemplate(name string) error {
	for _, item := range examples {
		if item.Name == name {
			return fmt.Errorf("built-in template is read-only")
		}
	}
	path, err := templatePath(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
func (p *Provision) RenameTemplate(oldName, newName string) error {
	item, err := p.LoadTemplate(oldName)
	if err != nil {
		return err
	}
	if item.BuiltIn {
		return fmt.Errorf("built-in template is read-only")
	}
	if err = p.SaveTemplate(newName, item.Script); err != nil {
		return err
	}
	return p.DeleteTemplate(oldName)
}

type Recovery struct{}

func (Recovery) Trigger(ctx context.Context, request sshx.Request) error {
	return sshx.Trigger(ctx, request)
}

type Download struct{ operation }

func (d *Download) Start(ctx context.Context, request download.Request, events *mygo.Channel[download.Progress]) error {
	ctx, err := d.begin(ctx)
	if err != nil {
		return err
	}
	defer d.end()
	return download.Start(ctx, request, func(progress download.Progress) { _ = events.Send(progress) })
}
func (d *Download) TestProxy(ctx context.Context, address string) error {
	return download.TestProxy(ctx, address)
}
func main() {
	mygo.Bind(Environment{}, Devices{}, &Flash{}, &Massflash{}, &Provision{}, Recovery{}, &Download{})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "nvflasher", Width: 1120, Height: 760, MinWidth: 780, MinHeight: 560, BackgroundColor: "#101820", StateKey: "main", URL: "/"})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
