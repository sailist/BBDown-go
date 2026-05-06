package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

// ServeOption holds flags for the serve subcommand.
type ServeOption struct {
	Listen string
}

// NewServeCommand creates the serve subcommand.
func NewServeCommand() *cobra.Command {
	opt := &ServeOption{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start BBDown API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServe(cmd.Context(), opt)
		},
	}
	cmd.Flags().StringVarP(&opt.Listen, "listen", "l", "http://0.0.0.0:23333", "server listen address")
	return cmd
}

// RunServe starts the API server.
func RunServe(ctx context.Context, opt *ServeOption) error {
	slog.DebugContext(ctx, "serve command executed", "listen", opt.Listen)
	return fmt.Errorf("not implemented: %w", fmt.Errorf("serve handler not yet wired"))
}
