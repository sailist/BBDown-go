package cli

import (
	"context"
	"strings"
	"testing"
)

func TestNewOptionDefaults(t *testing.T) {
	opt := NewOption()
	if opt.MultiThread != true {
		t.Errorf("expected MultiThread = true, got %v", opt.MultiThread)
	}
	if opt.ForceHttp != true {
		t.Errorf("expected ForceHttp = true, got %v", opt.ForceHttp)
	}
	if opt.SkipAi != true {
		t.Errorf("expected SkipAi = true, got %v", opt.SkipAi)
	}
	if opt.ForceReplaceHost != true {
		t.Errorf("expected ForceReplaceHost = true, got %v", opt.ForceReplaceHost)
	}
	if opt.DelayPerPage != "0" {
		t.Errorf("expected DelayPerPage = 0, got %s", opt.DelayPerPage)
	}
	if opt.Host != "api.bilibili.com" {
		t.Errorf("expected Host = api.bilibili.com, got %s", opt.Host)
	}
	if opt.EpHost != "api.bilibili.com" {
		t.Errorf("expected EpHost = api.bilibili.com, got %s", opt.EpHost)
	}
	if opt.TvHost != "api.snm0516.aisee.tv" {
		t.Errorf("expected TvHost = api.snm0516.aisee.tv, got %s", opt.TvHost)
	}
}

func TestNewRootCommandFlags(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		check    func(opt *Option) bool
		expected bool
	}{
		{
			name:     "use-tv-api long",
			args:     []string{"--use-tv-api"},
			check:    func(opt *Option) bool { return opt.UseTvApi },
			expected: true,
		},
		{
			name:     "use-app-api long",
			args:     []string{"--use-app-api"},
			check:    func(opt *Option) bool { return opt.UseAppApi },
			expected: true,
		},
		{
			name:     "use-intl-api long",
			args:     []string{"--use-intl-api"},
			check:    func(opt *Option) bool { return opt.UseIntlApi },
			expected: true,
		},
		{
			name:     "encoding-priority short",
			args:     []string{"-e", "hevc,avc"},
			check:    func(opt *Option) bool { return opt.EncodingPriority == "hevc,avc" },
			expected: true,
		},
		{
			name:     "dfn-priority short",
			args:     []string{"-q", "120,112"},
			check:    func(opt *Option) bool { return opt.DfnPriority == "120,112" },
			expected: true,
		},
		{
			name:     "only-show-info long",
			args:     []string{"--only-show-info"},
			check:    func(opt *Option) bool { return opt.OnlyShowInfo },
			expected: true,
		},
		{
			name:     "hide-streams long",
			args:     []string{"--hide-streams"},
			check:    func(opt *Option) bool { return opt.HideStreams },
			expected: true,
		},
		{
			name:     "interactive long",
			args:     []string{"--interactive"},
			check:    func(opt *Option) bool { return opt.Interactive },
			expected: true,
		},
		{
			name:     "use-aria2c long",
			args:     []string{"--use-aria2c"},
			check:    func(opt *Option) bool { return opt.UseAria2c },
			expected: true,
		},
		{
			name:     "multi-thread default true",
			args:     []string{},
			check:    func(opt *Option) bool { return opt.MultiThread },
			expected: true,
		},
		{
			name:     "multi-thread disabled",
			args:     []string{"--multi-thread=false"},
			check:    func(opt *Option) bool { return !opt.MultiThread },
			expected: true,
		},
		{
			name:     "select-page short",
			args:     []string{"-p", "1,3-5"},
			check:    func(opt *Option) bool { return opt.SelectPage == "1,3-5" },
			expected: true,
		},
		{
			name:     "force-http default true",
			args:     []string{},
			check:    func(opt *Option) bool { return opt.ForceHttp },
			expected: true,
		},
		{
			name:     "download-danmaku long",
			args:     []string{"--download-danmaku"},
			check:    func(opt *Option) bool { return opt.DownloadDanmaku },
			expected: true,
		},
		{
			name:     "download-danmaku-formats long",
			args:     []string{"--download-danmaku-formats", "ass,xml"},
			check:    func(opt *Option) bool { return opt.DownloadDanmakuFormats == "ass,xml" },
			expected: true,
		},
		{
			name:     "user-agent long",
			args:     []string{"--user-agent", "test-ua"},
			check:    func(opt *Option) bool { return opt.UserAgent == "test-ua" },
			expected: true,
		},
		{
			name:     "cookie short",
			args:     []string{"-c", "SESSDATA=abc"},
			check:    func(opt *Option) bool { return opt.Cookie == "SESSDATA=abc" },
			expected: true,
		},
		{
			name:     "access-token long",
			args:     []string{"--access-token", "tok"},
			check:    func(opt *Option) bool { return opt.AccessToken == "tok" },
			expected: true,
		},
		{
			name:     "file-pattern short",
			args:     []string{"-F", "<videoTitle>"},
			check:    func(opt *Option) bool { return opt.FilePattern == "<videoTitle>" },
			expected: true,
		},
		{
			name:     "multi-file-pattern short",
			args:     []string{"-M", MultiPageDefaultSavePath},
			check:    func(opt *Option) bool { return opt.MultiFilePattern == MultiPageDefaultSavePath },
			expected: true,
		},
		{
			name:     "skip-ai default true",
			args:     []string{},
			check:    func(opt *Option) bool { return opt.SkipAi },
			expected: true,
		},
		{
			name:     "host default",
			args:     []string{},
			check:    func(opt *Option) bool { return opt.Host == "api.bilibili.com" },
			expected: true,
		},
		{
			name:     "tv-host default",
			args:     []string{},
			check:    func(opt *Option) bool { return opt.TvHost == "api.snm0516.aisee.tv" },
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opt := NewOption()
			cmd := NewRootCommand(opt)
			cmd.SetArgs(c.args)
			// Prevent cobra from printing usage or exiting.
			cmd.SetOut(&cobraNoOpWriter{})
			cmd.SetErr(&cobraNoOpWriter{})
			_ = cmd.Execute()
			if c.check(opt) != c.expected {
				t.Errorf("flag parse check failed for args %v", c.args)
			}
		})
	}
}

func TestNewRootCommandPositionalArg(t *testing.T) {
	opt := NewOption()
	cmd := NewRootCommand(opt)
	cmd.SetArgs([]string{"https://www.bilibili.com/video/BV1xx411c7mD"})
	cmd.SetOut(&cobraNoOpWriter{})
	cmd.SetErr(&cobraNoOpWriter{})

	// RunE returns an error because handler is not wired, but URL should be parsed.
	_ = cmd.Execute()
	if opt.URL != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Errorf("expected URL to be parsed, got %s", opt.URL)
	}
}

func TestNewRootCommandHiddenFlags(t *testing.T) {
	opt := NewOption()
	cmd := NewRootCommand(opt)

	hidden := []string{
		"aria2c-proxy",
		"only-hevc",
		"only-avc",
		"only-av1",
		"add-dfn-subfix",
		"no-padding-page-num",
		"bandwith-ascending",
	}

	for _, name := range hidden {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("expected hidden flag %s to exist", name)
			continue
		}
		if !f.Hidden {
			t.Errorf("expected flag %s to be hidden", name)
		}
	}
}

func TestNewLoginCommand(t *testing.T) {
	cmd := NewLoginCommand()
	if cmd.Use != "login" {
		t.Errorf("expected Use = login, got %s", cmd.Use)
	}
	if cmd.Short != "login via WEB QR code" {
		t.Errorf("unexpected short description: %s", cmd.Short)
	}
}

func TestNewLoginTVCommand(t *testing.T) {
	cmd := NewLoginTVCommand()
	if cmd.Use != "logintv" {
		t.Errorf("expected Use = logintv, got %s", cmd.Use)
	}
	if cmd.Short != "login via TV QR code" {
		t.Errorf("unexpected short description: %s", cmd.Short)
	}
}

func TestNewServeCommand(t *testing.T) {
	cmd := NewServeCommand()
	if cmd.Use != "serve" {
		t.Errorf("expected Use = serve, got %s", cmd.Use)
	}
	if cmd.Short != "start BBDown API server" {
		t.Errorf("unexpected short description: %s", cmd.Short)
	}

	listen, err := cmd.Flags().GetString("listen")
	if err != nil {
		t.Fatalf("failed to get listen flag: %v", err)
	}
	if listen != "http://0.0.0.0:23333" {
		t.Errorf("expected listen default = http://0.0.0.0:23333, got %s", listen)
	}
}

func TestRunRootNotImplemented(t *testing.T) {
	opt := NewOption()
	err := RunRoot(context.Background(), opt)
	if err == nil {
		t.Error("expected error from unimplemented root handler")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error to contain 'not implemented', got %v", err)
	}
}

func TestRunLoginNotImplemented(t *testing.T) {
	err := RunLogin(context.Background())
	if err == nil {
		t.Error("expected error from unimplemented login handler")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error to contain 'not implemented', got %v", err)
	}
}

func TestRunLoginTVNotImplemented(t *testing.T) {
	err := RunLoginTV(context.Background())
	if err == nil {
		t.Error("expected error from unimplemented logintv handler")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error to contain 'not implemented', got %v", err)
	}
}

func TestRunServeNotImplemented(t *testing.T) {
	opt := &ServeOption{Listen: "http://127.0.0.1:8080"}
	err := RunServe(context.Background(), opt)
	if err == nil {
		t.Error("expected error from unimplemented serve handler")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error to contain 'not implemented', got %v", err)
	}
}

func TestDefaultSavePathConstants(t *testing.T) {
	if SinglePageDefaultSavePath != "<videoTitle>" {
		t.Errorf("unexpected SinglePageDefaultSavePath: %s", SinglePageDefaultSavePath)
	}
	if MultiPageDefaultSavePath != "<videoTitle>/[P<pageNumberWithZero>]<pageTitle>" {
		t.Errorf("unexpected MultiPageDefaultSavePath: %s", MultiPageDefaultSavePath)
	}
}

func TestSubcommandRegistration(t *testing.T) {
	opt := NewOption()
	root := NewRootCommand(opt)
	root.AddCommand(NewLoginCommand())
	root.AddCommand(NewLoginTVCommand())
	root.AddCommand(NewServeCommand())

	found := map[string]bool{}
	for _, c := range root.Commands() {
		found[c.Use] = true
	}

	if !found["login"] {
		t.Error("expected login subcommand to be registered")
	}
	if !found["logintv"] {
		t.Error("expected logintv subcommand to be registered")
	}
	if !found["serve"] {
		t.Error("expected serve subcommand to be registered")
	}
}

// cobraNoOpWriter discards all writes to silence cobra output in tests.
type cobraNoOpWriter struct{}

func (w *cobraNoOpWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
