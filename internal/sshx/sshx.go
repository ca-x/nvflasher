package sshx

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	SudoPassword string `json:"sudoPassword"`
}

func Trigger(ctx context.Context, r Request) error {
	if r.Host == "" || r.User == "" || r.Password == "" {
		return fmt.Errorf("host, user and SSH password required")
	}
	if r.Port == "" {
		r.Port = "22"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	callback, err := knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		return fmt.Errorf("known_hosts required: %w", err)
	}
	cfg := &ssh.ClientConfig{User: r.User, Auth: []ssh.AuthMethod{ssh.Password(r.Password)}, HostKeyCallback: callback, Timeout: 12 * time.Second}
	address := net.JoinHostPort(r.Host, r.Port)
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	clientConn, channels, requests, err := ssh.NewClientConn(conn, address, cfg)
	if err != nil {
		conn.Close()
		return err
	}
	client := ssh.NewClient(clientConn, channels, requests)
	defer client.Close()
	run := func(command, input string) error {
		session, e := client.NewSession()
		if e != nil {
			return e
		}
		defer session.Close()
		session.Stdin = strings.NewReader(input)
		var output bytes.Buffer
		session.Stdout = &output
		session.Stderr = &output
		e = session.Run(command)
		if e != nil {
			return fmt.Errorf("remote command failed: %s: %w", output.String(), e)
		}
		return nil
	}
	if err = run("sudo -n true", ""); err == nil {
		return run("sudo -n reboot forced-recovery", "")
	}
	pass := r.SudoPassword
	if pass == "" {
		pass = r.Password
	}
	return run("sudo -S -p '' reboot forced-recovery", pass+"\n")
}
