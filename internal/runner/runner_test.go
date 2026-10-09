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
