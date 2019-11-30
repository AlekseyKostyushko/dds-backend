package models

import (
	"github.com/jinzhu/gorm"
	"time"
)

type AvailableItem struct {
	gorm.Model
	ItemType string `gorm:"unique_index;not null"`
	Count    int    `gorm:"not null"`
	GameType string `gorm:"not null"`
}

type TakenItem struct {
	gorm.Model
	TakenBy        string `gorm:"not null"`
	ItemType       string `gorm:"not null"`
	AssignedToSlot string `gorm:"not null"`
	GameType       string `gorm:"not null"`
}

type HistoryItem struct {
	gorm.Model
	TakenBy  string    `gorm:"not null"`
	ItemType string    `gorm:"not null"`
	UsedTime time.Time `gorm:"not null"`
	GameType string    `gorm:"not null"`
}

// not done via reflection on purpose
// could be done via general function with generics, not yet present in go
func (a *AvailableItem) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	result["itemtype"] = a.ItemType
	result["count"] = a.Count
	result["gametype"] = a.GameType
	return result
}

func (a *TakenItem) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	result["takenby"] = a.TakenBy
	result["itemtype"] = a.ItemType
	result["assignedtoslot"] = a.AssignedToSlot
	result["gametype"] = a.GameType
	return result
}
