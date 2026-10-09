package setup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type progressReader struct {
	reader io.Reader
	total  int64
	bytes  int64
	last   int64
	stage  string
	emit   func(string)
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	count, err := reader.reader.Read(buffer)
	reader.bytes += int64(count)
	if reader.total > 0 {
		percent := reader.bytes * 100 / reader.total
		if percent != reader.last {
			reader.last = percent
			reader.emit(fmt.Sprintf("%s: %d%%", reader.stage, percent))
		}
	}
	return count, err
}

func extract(ctx context.Context, archive, directory, stage string, emit func(string)) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "tar", "-xjpf", "-")
	command.Dir = directory
	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return err
	}
	reader := &progressReader{reader: file, total: info.Size(), stage: stage, emit: emit}
	_, copyErr := io.CopyBuffer(stdin, reader, make([]byte, 128*1024))
	closeErr := stdin.Close()
	waitErr := command.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if waitErr != nil {
		return fmt.Errorf("extract %s: %w: %s", stage, waitErr, stderr.String())
	}
	if copyErr != nil {
		return fmt.Errorf("extract %s: %w", stage, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("extract %s: %w", stage, closeErr)
	}
	return nil
}
