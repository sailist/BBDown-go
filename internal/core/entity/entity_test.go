package entity

import (
	"testing"

	"github.com/nilaonai/bbdown-go/pkg/bvconv"
)

func TestPageEqual(t *testing.T) {
	p1 := Page{Aid: "123", Cid: "456", Epid: "789"}
	p2 := Page{Aid: "123", Cid: "456", Epid: "789"}
	p3 := Page{Aid: "123", Cid: "999", Epid: "789"}

	if !p1.Equal(p2) {
		t.Error("expected equal pages")
	}
	if p1.Equal(p3) {
		t.Error("expected unequal pages")
	}
}

func TestPageBVid(t *testing.T) {
	p := Page{Aid: "170001"}
	got := p.BVid()
	want := bvconv.Encode(170001)
	if got != want {
		t.Errorf("BVid() = %s, want %s", got, want)
	}
}

func TestVideoEqual(t *testing.T) {
	v1 := Video{ID: "1", Dfn: "1080P", Res: "1920x1080", Fps: "60", Codecs: "avc1", Bandwith: 1000, Dur: 120}
	v2 := Video{ID: "1", Dfn: "1080P", Res: "1920x1080", Fps: "60", Codecs: "avc1", Bandwith: 1000, Dur: 120}
	v3 := Video{ID: "1", Dfn: "720P", Res: "1920x1080", Fps: "60", Codecs: "avc1", Bandwith: 1000, Dur: 120}

	if !v1.Equal(v2) {
		t.Error("expected equal videos")
	}
	if v1.Equal(v3) {
		t.Error("expected unequal videos")
	}
}

func TestAudioEqual(t *testing.T) {
	a1 := Audio{ID: "1", Dfn: "192K", Codecs: "mp4a", Bandwith: 128, Dur: 120}
	a2 := Audio{ID: "1", Dfn: "192K", Codecs: "mp4a", Bandwith: 128, Dur: 120}
	a3 := Audio{ID: "1", Dfn: "192K", Codecs: "mp4a", Bandwith: 128, Dur: 999}

	if !a1.Equal(a2) {
		t.Error("expected equal audios")
	}
	if a1.Equal(a3) {
		t.Error("expected unequal audios")
	}
}

func TestAudioShortCodecs(t *testing.T) {
	a := Audio{Codecs: "mp4a.40.2"}
	got := a.ShortCodecs()
	want := "MP4A.40.2"
	if got != want {
		t.Errorf("ShortCodecs() = %s, want %s", got, want)
	}
}
