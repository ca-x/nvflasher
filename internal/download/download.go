package download

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	URL       string `json:"url"`
	Directory string `json:"directory"`
	Filename  string `json:"filename"`
	ProxyURL  string `json:"proxyUrl"`
}
type Progress struct {
	Bytes int64  `json:"bytes"`
	Total int64  `json:"total"`
	File  string `json:"file"`
}

func client(proxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
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
	client, err := client(r.ProxyURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", r.URL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
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
	var count int64
	last := time.Now()
	for {
		size, readErr := response.Body.Read(buffer)
		if size > 0 {
			if _, err = file.Write(buffer[:size]); err != nil {
				return err
			}
			count += int64(size)
			if time.Since(last) > 250*time.Millisecond {
				emit(Progress{count, response.ContentLength, target})
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
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(target+".part", target); err != nil {
		return err
	}
	emit(Progress{count, response.ContentLength, target})
	return nil
}
func TestProxy(ctx context.Context, address string) error {
	client, err := client(address)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://developer.nvidia.com/embedded/jetson-linux-archive", nil)
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
