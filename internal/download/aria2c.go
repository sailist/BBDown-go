package download

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sailist/BBDown-go/internal/config"
)

var execCommandContext = exec.CommandContext

// DownloadWithAria2c downloads a file using the external aria2c executable.
func DownloadWithAria2c(ctx context.Context, url, path, args string, cfg *config.Config) error {
	aria2cPath, err := findAria2c()
	if err != nil {
		return fmt.Errorf("find aria2c: %w", err)
	}

	dir := filepath.Dir(path)
	filename := filepath.Base(path)

	var cmdArgs []string
	cmdArgs = append(cmdArgs,
		"--auto-file-renaming=false",
		"--download-result=hide",
		"--allow-overwrite=true",
		"--console-log-level=warn",
		"-x16",
		"-s16",
		"-j16",
		"-k5M",
	)

	if !strings.Contains(url, "platform=android_tv_yst") && !strings.Contains(url, "platform=android") {
		cmdArgs = append(cmdArgs, "--header=Referer: https://www.bilibili.com")
	}
	cmdArgs = append(cmdArgs, "--header=User-Agent: Mozilla/5.0")
	if cfg != nil && cfg.Cookie != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--header=Cookie: %s", cfg.Cookie))
	}

	if args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	cmdArgs = append(cmdArgs, url)
	cmdArgs = append(cmdArgs, "-d", dir, "-o", filename)

	cmd := execCommandContext(ctx, aria2cPath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aria2c: %w", err)
	}
	return nil
}

// IsAria2cAvailable reports whether an aria2c executable can be found.
func IsAria2cAvailable() bool {
	_, err := findAria2c()
	return err == nil
}

func findAria2c() (string, error) {
	// current dir
	if _, err := os.Stat("./aria2c"); err == nil {
		abs, err := filepath.Abs("./aria2c")
		if err == nil {
			return abs, nil
		}
	}
	if _, err := os.Stat("./aria2c.exe"); err == nil {
		abs, err := filepath.Abs("./aria2c.exe")
		if err == nil {
			return abs, nil
		}
	}

	// program dir
	execPath, err := os.Executable()
	if err == nil {
		programDir := filepath.Dir(execPath)
		candidate := filepath.Join(programDir, "aria2c")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		candidate = filepath.Join(programDir, "aria2c.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// PATH
	return exec.LookPath("aria2c")
}
