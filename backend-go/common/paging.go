package common

import "strings"

type Paging struct {
	Page  int   `json:"page,omitempty" form:"page"`
	Limit int   `json:"limit,omitempty" form:"limit"`
	Total int64 `json:"total,omitempty" form:"total"`

	FakeCursor string `json:"cursor,omitempty" form:"cursor"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func (p *Paging) Fulfill() {
	if p.Page <= 0 {
		p.Page = 1
	}

	if p.Limit <= 0 {
		p.Limit = 50
	}

	p.FakeCursor = strings.TrimSpace(p.FakeCursor)
}
