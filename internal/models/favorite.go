package models

import (
	"time"

	"gorm.io/gorm"
)

// Favorite - Modelo de favoritos
type Favorite struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID     uint     `gorm:"not null;uniqueIndex:idx_user_property" json:"user_id"`
	User       User     `gorm:"foreignKey:UserID" json:"-"`
	PropertyID uint     `gorm:"not null;uniqueIndex:idx_user_property" json:"property_id"`
	Property   Property `gorm:"foreignKey:PropertyID" json:"property"`
}

func (Favorite) TableName() string {
	return "favoritos"
}
