package flash

import (
	"context"
	"fmt"
	"strings"

	"nvflasher/internal/runner"
)

// NVIDIA scripts can ignore tar's exit status and proceed to flashing.
// Cancel the process group as soon as an incomplete APP archive is reported.
func runChecked(ctx context.Context, command runner.Command, emit func(string)) error {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	var packagingError error
	err := runner.Run(child, command, func(line string) {
		emit(line)
		if packagingError == nil && fatalPackagingLine(line) {
			packagingError = fmt.Errorf("BSP image packaging failed: %s; do not flash this package; inspect the raw image offline", line)
			cancel()
		}
	})
	if packagingError != nil {
		return packagingError
	}
	return err
}

func fatalPackagingLine(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "tar:") && (strings.Contains(line, "Cannot savedir:") || strings.Contains(line, "Exiting with failure status due to previous errors"))
}
