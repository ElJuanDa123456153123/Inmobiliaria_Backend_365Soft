package models

import (
	"time"
)

// ViewHistory - Modelo de historial de vistas de propiedades
type ViewHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID     uint     `gorm:"not null;index:idx_vh_user_date" json:"user_id"`
	User       User     `gorm:"foreignKey:UserID" json:"-"`
	PropertyID uint     `gorm:"not null;index:idx_vh_user_property" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property"`
	ViewedAt   time.Time `gorm:"not null;index:idx_vh_user_date" json:"viewed_at"`
}

func (ViewHistory) TableName() string {
	return "view_histories"
}
