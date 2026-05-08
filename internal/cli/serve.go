package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/server"
)

// ServeOption holds flags for the serve subcommand.
type ServeOption struct {
	Listen string
}

// NewServeCommand creates the serve subcommand.
func NewServeCommand(rootOpt *Option) *cobra.Command {
	opt := &ServeOption{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start BBDown API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServe(cmd.Context(), rootOpt, opt.Listen)
		},
	}
	cmd.Flags().StringVarP(&opt.Listen, "listen", "l", "http://0.0.0.0:23333", "server listen address")
	return cmd
}

// RunServe starts the API server.
func RunServe(ctx context.Context, opt *Option, listen string) error {
	slog.DebugContext(ctx, "serve command executed", "listen", listen)

	u, err := url.Parse(listen)
	if err != nil || u.Scheme != "http" {
		return fmt.Errorf("%s is not a valid http URL, url example: http://0.0.0.0:23333. If you need https, please configure a reverse proxy", listen)
	}

	cfg := config.NewConfig()
	cfg.Cookie = opt.Cookie
	cfg.Token = opt.AccessToken
	cfg.DebugLog = opt.Debug
	cfg.Host = opt.Host
	cfg.EpHost = opt.EpHost
	cfg.TvHost = opt.TvHost
	cfg.Area = opt.Area

	srv := server.NewServer(cfg)
	return srv.Run(u.Host)
}
