package muxer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

// FindExecutable searches for an executable named `name` in the following order:
// 1. Current working directory
// 2. Directory containing the current executable
// 3. Directories listed in the PATH environment variable
func FindExecutable(name string) (string, error) {
	fileExt := ""
	if runtime.GOOS == "windows" {
		fileExt = ".exe"
	}

	// Current directory
	candidate := name + fileExt
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		abs, err := filepath.Abs(candidate)
		if err == nil {
			return abs, nil
		}
	}

	// Program directory
	execPath, err := os.Executable()
	if err == nil {
		programDir := filepath.Dir(execPath)
		candidate = filepath.Join(programDir, name+fileExt)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	// PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("executable %q not found", name)
}

// CheckFFmpegDOVI runs "ffmpeg -version" and checks whether libavutil
// version is at least 57.17.
func CheckFFmpegDOVI() (bool, error) {
	ffmpegPath, err := FindExecutable("ffmpeg")
	if err != nil {
		return false, fmt.Errorf("find ffmpeg: %w", err)
	}

	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ffmpeg -version: %w", err)
	}

	re := regexp.MustCompile(`libavutil\s+(\d+)\.\s*(\d+)`)
	match := re.FindStringSubmatch(string(output))
	if match == nil {
		return false, nil
	}

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])

	return (major == 57 && minor >= 17) || major > 57, nil
}
