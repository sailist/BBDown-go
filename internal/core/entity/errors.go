package entity

import "errors"

var (
	ErrLoginRequired = errors.New("login required")
	ErrRegionBlocked = errors.New("content blocked in current region")
	ErrParseFailed   = errors.New("failed to parse video streams")
	ErrQRExpired     = errors.New("QR code expired")
	ErrKeyNotFound   = errors.New("key not found")
)
