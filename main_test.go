package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthenticateSudo(t *testing.T) {
	dir := t.TempDir()
	program := `#!/bin/sh
[ "$1" = '-S' ] && [ "$2" = '-p' ] && [ -z "$3" ] && [ "$4" = '-v' ] || exit 3
IFS= read -r supplied
[ "$supplied" = 'correct password' ] || exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte(program), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := authenticateSudo("correct password"); err != nil {
		t.Fatal(err)
	}
	err := authenticateSudo("wrong password")
	if err == nil || strings.Contains(err.Error(), "wrong password") {
		t.Fatalf("authentication error exposed password or was not reported: %v", err)
	}
}
