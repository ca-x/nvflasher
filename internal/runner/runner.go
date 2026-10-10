package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var secret = regexp.MustCompile(`(?i)(-p\s+|password[=:]\s*)\S+`)

func Redact(line string) string { return secret.ReplaceAllString(line, "${1}***") }

type Command struct {
	Dir   string
	Args  []string
	Input string
	Env   []string
}

func Run(ctx context.Context, spec Command, emit func(string)) (result error) {
	if loggingEnabled.Load() {
		logger, file, err := commandLog()
		if err != nil {
			emit(fmt.Sprintf("Cannot write diagnostic log: %v", err))
		} else {
			defer file.Close()
			logger.Info("command started", "command", Redact(strings.Join(spec.Args, " ")), "directory", spec.Dir)
			emit("Diagnostic log: " + file.Name())
			original := emit
			emit = func(line string) {
				if loggingEnabled.Load() {
					logger.Info("output", "line", Redact(line))
				}
				original(line)
			}
			defer func() { logger.Info("command ended", "error", result) }()
		}
	}
	if len(spec.Args) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := exec.Command(spec.Args[0], spec.Args[1:]...)
	cmd.Dir = spec.Dir
	cmd.Stdin = strings.NewReader(spec.Input)
	if len(spec.Env) != 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	prepare(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	lines := make(chan string, 128)
	done := make(chan struct{}, 2)
	read := func(r io.Reader) {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			select {
			case lines <- Redact(scanner.Text()):
			case <-ctx.Done():
				return
			}
		}
	}
	go read(stdout)
	go read(stderr)
	finished := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			terminate(cmd, finished)
		case <-finished:
		}
	}()
	for count := 0; count < 2; {
		select {
		case line := <-lines:
			emit(line)
		case <-done:
			count++
		}
	}
	for len(lines) > 0 {
		emit(<-lines)
	}
	err = cmd.Wait()
	close(finished)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
