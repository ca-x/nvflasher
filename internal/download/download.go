package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Request struct {
	URL       string `json:"url"`
	Directory string `json:"directory"`
	Filename  string `json:"filename"`
	ProxyURL  string `json:"proxyUrl"`
	SHA256    string `json:"sha256"`
}
type Progress struct {
	Bytes int64  `json:"bytes"`
	Total int64  `json:"total"`
	File  string `json:"file"`
	Rate  int64  `json:"rate"`
	ETA   int64  `json:"eta"`
}

func client(proxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(parsed)
		case "socks5":
			dialer, err := proxy.FromURL(parsed, proxy.Direct)
			if err != nil {
				return nil, err
			}
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				if contextual, ok := dialer.(proxy.ContextDialer); ok {
					return contextual.DialContext(ctx, network, address)
				}
				return dialer.Dial(network, address)
			}
		default:
			return nil, fmt.Errorf("unsupported proxy scheme")
		}
	}
	return &http.Client{Transport: transport, Timeout: 0, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL.Scheme != "https" && req.URL.Scheme != "http" {
			return fmt.Errorf("unsupported redirect")
		}
		return nil
	}}, nil
}
func Start(ctx context.Context, r Request, emit func(Progress)) error {
	httpClient, err := client(r.ProxyURL)
	if err != nil {
		return err
	}
	return start(ctx, r, emit, httpClient)
}

func start(ctx context.Context, r Request, emit func(Progress), httpClient *http.Client) error {
	source, err := url.Parse(r.URL)
	if err != nil || source.Host == "" || (source.Scheme != "http" && source.Scheme != "https") {
		return fmt.Errorf("HTTP(S) direct URL required")
	}
	name := r.Filename
	if name == "" {
		name = filepath.Base(source.Path)
	}
	if name == "" || name == "." || name == "/" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid filename")
	}
	if err = os.MkdirAll(r.Directory, 0755); err != nil {
		return err
	}
	target := filepath.Join(r.Directory, name)
	if _, err = os.Stat(target); err == nil {
		return fmt.Errorf("file already exists: %s", target)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", r.URL, nil)
	if err != nil {
		return err
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("server returned %s", response.Status)
	}
	file, err := os.OpenFile(target+".part", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer os.Remove(target + ".part")
	defer file.Close()
	buffer := make([]byte, 64*1024)
	hasher := sha256.New()
	var count int64
	last, started := time.Now(), time.Now()
	for {
		size, readErr := response.Body.Read(buffer)
		if size > 0 {
			if _, err = file.Write(buffer[:size]); err != nil {
				return err
			}
			_, _ = hasher.Write(buffer[:size])
			count += int64(size)
			if time.Since(last) > 250*time.Millisecond {
				rate := int64(float64(count) / time.Since(started).Seconds())
				eta := int64(0)
				if response.ContentLength > 0 && rate > 0 {
					eta = (response.ContentLength - count) / rate
				}
				emit(Progress{Bytes: count, Total: response.ContentLength, File: target, Rate: rate, ETA: eta})
				last = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if r.SHA256 != "" {
		if len(r.SHA256) != 64 {
			return fmt.Errorf("SHA256 must contain 64 hexadecimal characters")
		}
		if _, err = hex.DecodeString(r.SHA256); err != nil {
			return fmt.Errorf("invalid SHA256: %w", err)
		}
		if !strings.EqualFold(hex.EncodeToString(hasher.Sum(nil)), r.SHA256) {
			return fmt.Errorf("SHA256 mismatch; downloaded file discarded")
		}
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(target+".part", target); err != nil {
		return err
	}
	emit(Progress{Bytes: count, Total: response.ContentLength, File: target, Rate: int64(float64(count) / time.Since(started).Seconds())})
	return nil
}

func Show(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{"-R", path}
	case "windows":
		command, args = "explorer", []string{"/select,", path}
	default:
		command, args = "xdg-open", []string{filepath.Dir(path)}
	}
	return exec.Command(command, args...).Start()
}
func TestProxy(ctx context.Context, address string) error {
	return TestProxyFor(ctx, address, "https://developer.nvidia.com/embedded/jetson-linux-archive")
}

func TestProxyFor(ctx context.Context, address, target string) error {
	client, err := client(address)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("target must be an HTTP(S) URL")
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", target, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("proxy test returned %s", response.Status)
	}
	return nil
}
