package entity

// VInfo holds metadata information about a video / bangumi.
type VInfo struct {
	Title      string `json:"title"`
	Desc       string `json:"desc"`
	Pic        string `json:"pic"`
	PubTime    int64  `json:"pubTime"`
	IsBangumi  bool   `json:"isBangumi"`
	IsCheese   bool   `json:"isCheese"`
	IsBangumiEnd bool `json:"isBangumiEnd"`
	Index      string `json:"index"`
	PagesInfo  []Page `json:"pagesInfo"`
	IsSteinGate bool  `json:"isSteinGate"`
}
