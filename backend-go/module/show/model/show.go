package showmodel

import (
	"cinema/common"
)

const EntityName = "Show"
const TableName = "shows"

type Show common.Show

func (Show) TableName() string { return TableName }

func (s *Show) Mask(isAdminOrOwner bool) {
	if auditorium := s.Auditorium; auditorium != nil {
		auditorium.Mask(isAdminOrOwner)
	}
}

type ShowCreate struct {
	ID               int          `json:"id" gorm:"column:id;primary_key" swaggerignore:"true"`
	Date             *common.Date `json:"date" gorm:"column:date" `
	StartTime        *common.Time `json:"startTime" gorm:"column:start_time"`
	EndTime          *common.Time `json:"endTime" gorm:"column:end_time"`
	AuditoriumID     int64        `json:"-" gorm:"column:auditorium_id"`
	AuditoriumFakeID string       `json:"auditoriumID" gorm:"-"`
	ImdbID           string       `json:"imdbID" gorm:"column:imdb_id"`
}

func (ShowCreate) TableName() string { return TableName }

type ShowUpdate struct {
	Date         *common.Date `gorm:"column:date" json:"date"`
	EndTime      *common.Time `gorm:"column:end_time" json:"endTime"`
	StartTime    *common.Time `gorm:"column:start_time" json:"startTime"`
	AuditoriumID int64        `gorm:"column:auditorium_id" json:"-"`
	ImdbID       string       `gorm:"column:imdb_id" json:"imdbID"`
}

func (ShowUpdate) TableName() string { return TableName }
