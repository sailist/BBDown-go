package util

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// GetRandomString generates a random alphanumeric string of the given length.
func GetRandomString(length int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rnd.Intn(len(chars))]
	}
	return string(b)
}

// WbiSign generates a WBI signature by appending an MD5 hash of api+wbi.
func WbiSign(api, wbi string) string {
	h := md5.Sum([]byte(api + wbi))
	return fmt.Sprintf("%s&w_rid=%x", api, h)
}

// GetMixinKey derives a 32-character mixin key from orig using a fixed index table.
func GetMixinKey(orig string) string {
	mixinKeyEncTab := []int{
		46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
		27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
	}

	var b strings.Builder
	b.Grow(32)
	for _, idx := range mixinKeyEncTab {
		b.WriteByte(orig[idx])
	}
	return b.String()
}

// GetSign returns an MD5 sign for the given parameters using Bilibili's salts.
func GetSign(parms string, isBiliPlus bool) string {
	var toEncode string
	if isBiliPlus {
		toEncode = parms + "acd495b248ec528c2eed1e862d393126"
	} else {
		toEncode = parms + "59b43e04ad6965f34319062b478f83dd"
	}
	h := md5.Sum([]byte(toEncode))
	return fmt.Sprintf("%x", h)
}

// GetTimestamp returns the current Unix timestamp as a string.
// If seconds is true, it returns Unix seconds; otherwise, milliseconds.
func GetTimestamp(seconds bool) string {
	now := time.Now()
	if seconds {
		return strconv.FormatInt(now.Unix(), 10)
	}
	return strconv.FormatInt(now.UnixMilli(), 10)
}

// FormatFileSize formats a file size in bytes to a human-readable string.
func FormatFileSize(size float64) string {
	switch {
	case size < 0:
		return "invalid size"
	case size >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB", size/(1024*1024*1024))
	case size >= 1024*1024:
		return fmt.Sprintf("%.2f MB", size/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.2f KB", size/1024)
	default:
		return fmt.Sprintf("%v bytes", size)
	}
}

// FormatTime formats a duration in seconds to a human-readable string.
// If absolute is true, it returns "HH:MM:SS"; otherwise, it returns "XhXXmXXs" or "XXmXXs".
func FormatTime(seconds int, absolute bool) string {
	ts := time.Duration(seconds) * time.Second
	totalHours := int(ts.Hours())
	minutes := int(ts.Minutes()) % 60
	sec := int(ts.Seconds()) % 60

	if absolute {
		return fmt.Sprintf("%02d:%02d:%02d", totalHours, minutes, sec)
	}

	if totalHours == 0 {
		return fmt.Sprintf("%02dm%02ds", minutes, sec)
	}
	return fmt.Sprintf("%dh%02dm%02ds", totalHours, minutes, sec)
}

// GetValidFileName replaces invalid filename characters with replacement.
// If filterSlash is true, it also explicitly replaces '/' and '\\'.
func GetValidFileName(input, replacement string, filterSlash bool) string {
	invalidChars := []rune{
		34, 60, 62, 124, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 58, 42, 63, 92, 47,
	}

	invalidSet := make(map[rune]struct{}, len(invalidChars))
	for _, c := range invalidChars {
		invalidSet[c] = struct{}{}
	}

	var b strings.Builder
	for _, c := range input {
		if _, ok := invalidSet[c]; ok {
			b.WriteString(replacement)
		} else {
			b.WriteRune(c)
		}
	}

	title := b.String()
	if filterSlash {
		title = strings.ReplaceAll(title, "/", replacement)
		title = strings.ReplaceAll(title, "\\", replacement)
	}
	return title
}
