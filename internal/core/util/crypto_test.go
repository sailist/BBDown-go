package util

import (
	"strconv"
	"testing"
)

func TestWbiSign(t *testing.T) {
	got := WbiSign("foo=bar", "testkey")
	want := "foo=bar&w_rid=8d1a0e0e6dce58e83f6fa70eaa6a4a60"
	if got != want {
		t.Errorf("WbiSign(\"foo=bar\", \"testkey\") = %q, want %q", got, want)
	}
}

func TestGetMixinKey(t *testing.T) {
	orig := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!?"
	got := GetMixinKey(orig)
	want := "KLi2R8nwfOavW3JzrH5Nx9GjtseDcCFd"
	if got != want {
		t.Errorf("GetMixinKey(...) = %q, want %q", got, want)
	}
}

func TestGetSign(t *testing.T) {
	tests := []struct {
		parms      string
		isBiliPlus bool
		want       string
	}{
		{"hello", false, "187efac2968f6c908267699283a9e969"},
		{"hello", true, "c6ace8546bd45114aafb88034d5a545c"},
	}

	for _, tt := range tests {
		got := GetSign(tt.parms, tt.isBiliPlus)
		if got != tt.want {
			t.Errorf("GetSign(%q, %v) = %q, want %q", tt.parms, tt.isBiliPlus, got, tt.want)
		}
	}
}

func TestGetTimestamp(t *testing.T) {
	// Seconds should be ~10 digits
	s := GetTimestamp(true)
	if _, err := strconv.ParseInt(s, 10, 64); err != nil {
		t.Errorf("GetTimestamp(true) = %q, not a valid integer", s)
	}
	if len(s) < 9 || len(s) > 11 {
		t.Errorf("GetTimestamp(true) = %q, unexpected length %d", s, len(s))
	}

	// Milliseconds should be ~13 digits
	ms := GetTimestamp(false)
	if _, err := strconv.ParseInt(ms, 10, 64); err != nil {
		t.Errorf("GetTimestamp(false) = %q, not a valid integer", ms)
	}
	if len(ms) < 12 || len(ms) > 14 {
		t.Errorf("GetTimestamp(false) = %q, unexpected length %d", ms, len(ms))
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size float64
		want string
	}{
		{-1, "invalid size"},
		{0, "0 bytes"},
		{512, "512 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1.5 * 1024 * 1024, "1.50 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{2.5 * 1024 * 1024 * 1024, "2.50 GB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.size)
		if got != tt.want {
			t.Errorf("FormatFileSize(%v) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		seconds  int
		absolute bool
		want     string
	}{
		{0, false, "00m00s"},
		{0, true, "00:00:00"},
		{65, false, "01m05s"},
		{65, true, "00:01:05"},
		{3665, false, "1h01m05s"},
		{3665, true, "01:01:05"},
		{3600, false, "1h00m00s"},
		{3600, true, "01:00:00"},
		{7200, false, "2h00m00s"},
		{7200, true, "02:00:00"},
	}

	for _, tt := range tests {
		got := FormatTime(tt.seconds, tt.absolute)
		if got != tt.want {
			t.Errorf("FormatTime(%d, %v) = %q, want %q", tt.seconds, tt.absolute, got, tt.want)
		}
	}
}

func TestGetValidFileName(t *testing.T) {
	tests := []struct {
		input       string
		replacement string
		filterSlash bool
		want        string
	}{
		{"hello", "_", false, "hello"},
		{"file:name?", "_", false, "file_name_"},
		{"a/b\\c", "_", true, "a_b_c"},
		{"quote\"test", "-", false, "quote-test"},
		{"pipe|star*", "_", false, "pipe_star_"},
		{"slash/included", "_", false, "slash_included"},
	}

	for _, tt := range tests {
		got := GetValidFileName(tt.input, tt.replacement, tt.filterSlash)
		if got != tt.want {
			t.Errorf("GetValidFileName(%q, %q, %v) = %q, want %q", tt.input, tt.replacement, tt.filterSlash, got, tt.want)
		}
	}
}

func TestGetValidFileNameWithSlashInReplacement(t *testing.T) {
	// When replacement itself contains '/', the filterSlash second pass
	// will replace the newly introduced '/' characters again (matching C# behavior).
	input := "a/b"
	replacement := "_/_"
	got := GetValidFileName(input, replacement, true)
	// First pass: "a" + "_/_" + "b" = "a_/_b"
	// Second pass (filterSlash): replace '/' with "_/_" -> "a__/__b"
	want := "a__/__b"
	if got != want {
		t.Errorf("GetValidFileName(%q, %q, true) = %q, want %q", input, replacement, got, want)
	}
}
