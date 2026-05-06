package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ConfigEntry represents a single parsed config line.
type ConfigEntry struct {
	Key    string
	Values []string
}

// ParseConfig reads a BBDown.config file and returns ordered key-value entries.
// Lines starting with # are treated as comments and ignored.
// Each config line is expected to be in --key value format.
// Values may be quoted with ".
func ParseConfig(path string) ([]ConfigEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer file.Close()

	var result []ConfigEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "-") && strings.Contains(line, " ") {
			spaceIdx := strings.Index(line, " ")
			key := strings.TrimSpace(line[:spaceIdx])
			val := strings.TrimSpace(line[spaceIdx:])
			key = strings.Trim(key, "\"")
			val = strings.Trim(val, "\"")
			if key != "" {
				result = append(result, ConfigEntry{Key: key, Values: []string{val}})
			}
		} else {
			key := strings.Trim(line, "\"")
			if key != "" {
				result = append(result, ConfigEntry{Key: key, Values: []string{}})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return result, nil
}

// HandleConfig reads the config file and prepends parsed config args to the provided args slice.
// If configFile is empty, it defaults to "BBDown.config".
// Command-line args take precedence over config file values.
func HandleConfig(args []string, configFile string) ([]string, error) {
	if configFile == "" {
		configFile = "BBDown.config"
	}

	info, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return args, nil
		}
		return nil, fmt.Errorf("checking config file: %w", err)
	}

	if info.IsDir() {
		return args, nil
	}

	slog.Info("loading config file", "path", configFile)

	entries, err := ParseConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	var result []string
	for _, entry := range entries {
		if containsArg(args, entry.Key) {
			continue
		}
		result = append(result, entry.Key)
		result = append(result, entry.Values...)
	}

	return append(result, args...), nil
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
