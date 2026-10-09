package flash

import (
	"context"
	"nvflasher/internal/runner"
	"nvflasher/internal/tegra"
)

func Run(ctx context.Context, dir string, opts tegra.Options, emit func(string)) error {
	args, err := tegra.Command(opts)
	if err != nil {
		return err
	}
	return runner.Run(ctx, runner.Command{Dir: dir, Args: args}, emit)
}
