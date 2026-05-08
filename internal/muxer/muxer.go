package muxer

import (
	"context"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

// Muxer defines the interface for multiplexing media files.
type Muxer interface {
	Mux(ctx context.Context, cfg MuxConfig) error
	MergeFLV(ctx context.Context, files []string, outPath string) error
}

// MuxConfig holds all parameters needed for a muxing operation.
type MuxConfig struct {
	BVid          string
	VideoPath     string
	AudioPath     string
	AudioMaterial []entity.AudioMaterial
	OutPath       string
	Desc          string
	Title         string
	Author        string
	EpisodeID     string
	Pic           string
	Lang          string
	Subtitles     []entity.Subtitle
	AudioOnly     bool
	VideoOnly     bool
	Points        []entity.ViewPoint
	PubTime       int64
	SimplyMux     bool
	IsHevc        bool
}
