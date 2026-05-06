package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigWithComments(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "BBDown.config")

	content := `
# This is a comment
--cookie test123

# Another comment
--debug-log
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	entries, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].Key != "--cookie" || len(entries[0].Values) != 1 || entries[0].Values[0] != "test123" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Key != "--debug-log" || len(entries[1].Values) != 0 {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseConfigKeyValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "BBDown.config")

	content := `--cookie SESSDATA=abc
--host api.example.com`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	entries, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].Key != "--cookie" || len(entries[0].Values) != 1 || entries[0].Values[0] != "SESSDATA=abc" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Key != "--host" || len(entries[1].Values) != 1 || entries[1].Values[0] != "api.example.com" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseConfigQuotedValues(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "BBDown.config")

	content := `--cookie "SESSDATA=abc; bili_jct=xyz"
--work-dir "/path/to/dir"`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	entries, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].Key != "--cookie" || len(entries[0].Values) != 1 || entries[0].Values[0] != "SESSDATA=abc; bili_jct=xyz" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Key != "--work-dir" || len(entries[1].Values) != 1 || entries[1].Values[0] != "/path/to/dir" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestHandleConfigCLIPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "BBDown.config")

	content := `--cookie config_cookie
--debug-log
--host config_host`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	args := []string{"--cookie", "cli_cookie", "some_url"}
	result, err := HandleConfig(args, configPath)
	if err != nil {
		t.Fatalf("HandleConfig failed: %v", err)
	}

	expected := []string{"--debug-log", "--host", "config_host", "--cookie", "cli_cookie", "some_url"}
	if !sliceEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestHandleConfigDefaultPath(t *testing.T) {
	configPath := "BBDown.config"
	content := `--debug-log`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(configPath)
	})

	args := []string{"some_url"}
	result, err := HandleConfig(args, "")
	if err != nil {
		t.Fatalf("HandleConfig failed: %v", err)
	}

	expected := []string{"--debug-log", "some_url"}
	if !sliceEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestHandleConfigMissingFile(t *testing.T) {
	args := []string{"some_url"}
	result, err := HandleConfig(args, filepath.Join(t.TempDir(), "nonexistent.config"))
	if err != nil {
		t.Fatalf("HandleConfig failed: %v", err)
	}

	if !sliceEqual(result, args) {
		t.Errorf("expected %v, got %v", args, result)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
