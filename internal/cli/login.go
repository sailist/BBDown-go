package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

// NewLoginCommand creates the login subcommand.
func NewLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login via WEB QR code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunLogin(cmd.Context())
		},
	}
	return cmd
}

// NewLoginTVCommand creates the logintv subcommand.
func NewLoginTVCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logintv",
		Short: "login via TV QR code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunLoginTV(cmd.Context())
		},
	}
	return cmd
}

// RunLogin performs WEB QR code login.
func RunLogin(ctx context.Context) error {
	slog.DebugContext(ctx, "login command executed")
	return fmt.Errorf("not implemented: %w", fmt.Errorf("login handler not yet wired"))
}

// RunLoginTV performs TV QR code login.
func RunLoginTV(ctx context.Context) error {
	slog.DebugContext(ctx, "logintv command executed")
	return fmt.Errorf("not implemented: %w", fmt.Errorf("logintv handler not yet wired"))
}
