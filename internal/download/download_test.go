package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestStartAndChecksum(t *testing.T) {
	content := []byte("downloaded payload")
	client := &http.Client{Transport: testTransport(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(content)), ContentLength: int64(len(content)), Header: make(http.Header)}, nil
	})}
	directory := t.TempDir()
	hash := sha256.Sum256(content)
	request := Request{URL: "https://example.test/bsp.tar.gz", Directory: directory, SHA256: hex.EncodeToString(hash[:])}
	if err := start(context.Background(), request, func(progress Progress) {
		if progress.Bytes != int64(len(content)) {
			t.Errorf("unexpected progress: %+v", progress)
		}
	}, client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "bsp.tar.gz")); err != nil {
		t.Fatal(err)
	}
	request.Filename = "invalid.tar.gz"
	request.SHA256 = hex.EncodeToString(make([]byte, 32))
	if err := start(context.Background(), request, func(Progress) {}, client); err == nil {
		t.Fatal("bad checksum accepted")
	}
	if _, err := os.Stat(filepath.Join(directory, "invalid.tar.gz.part")); !os.IsNotExist(err) {
		t.Fatal("partial file was not removed")
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (transport testTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := Request{URL: "http://127.0.0.1:1/bsp.tar.gz", Directory: t.TempDir()}
	if err := Start(ctx, request, func(Progress) {}); err == nil {
		t.Fatal("cancelled download succeeded")
	}
}
