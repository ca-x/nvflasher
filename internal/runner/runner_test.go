package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRedact(t *testing.T) {
	if result := Redact("create -p secret password=hidden"); strings.Contains(result, "secret") || strings.Contains(result, "hidden") {
		t.Fatal(result)
	}
}
func TestCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := Run(ctx, Command{Args: []string{"sleep", "10"}}, func(string) {})
	if err == nil {
		t.Fatal("expected cancellation")
	}
}
func TestCommandEnv(t *testing.T) {
	var output string
	if err := Run(t.Context(), Command{Args: []string{"sh", "-c", "printf '%s\\n' \"$SETUP_TEST_VALUE\""}, Env: []string{"SETUP_TEST_VALUE=ready"}}, func(line string) { output += line }); err != nil {
		t.Fatal(err)
	}
	if output != "ready" {
		t.Fatalf("child env: %q", output)
	}
}
