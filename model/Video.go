package model

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Video VIDEOテーブル
type Video struct {
	gorm.Model
	VideoID     string
	ChannelID   string
	Title       string
	URL         string
	Length      string
	PublishedAt time.Time `gorm:"default:NULL"`
}
