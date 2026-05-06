package entity

import (
	"strconv"

	"github.com/nilaonai/bbdown-go/pkg/bvconv"
)

// Page represents a video page (episode/part).
type Page struct {
	Index     int         `json:"index"`
	Aid       string      `json:"aid"`
	Cid       string      `json:"cid"`
	Epid      string      `json:"epid"`
	Title     string      `json:"title"`
	Dur       int         `json:"dur"`
	Res       string      `json:"res"`
	PubTime   int64       `json:"pubTime"`
	Cover     string      `json:"cover"`
	Desc      string      `json:"desc"`
	OwnerName string      `json:"ownerName"`
	OwnerMid  string      `json:"ownerMid"`
	Points    []ViewPoint `json:"points"`
}

// BVid returns the BVid for this page by encoding its Aid.
func (p Page) BVid() string {
	aid, _ := strconv.ParseInt(p.Aid, 10, 64)
	return bvconv.Encode(aid)
}

// Equal reports whether p and other represent the same page.
func (p Page) Equal(other Page) bool {
	return p.Aid == other.Aid && p.Cid == other.Cid && p.Epid == other.Epid
}

// ViewPoint represents a single viewpoint / chapter marker.
type ViewPoint struct {
	Title string `json:"title"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
