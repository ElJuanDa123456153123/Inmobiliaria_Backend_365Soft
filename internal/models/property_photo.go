package models

import (
	"time"

	"gorm.io/gorm"
)

// PropertyPhoto - Modelo de fotos de propiedad
type PropertyPhoto struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relación con propiedad
	PropertyID uint     `gorm:"not null;index" json:"property_id"`
	Property   *Property `gorm:"foreignKey:PropertyID;constraint:OnDelete:CASCADE" json:"-"`

	// Datos de la foto
	Orden      int    `gorm:"default:0" json:"orden"`
	URL        string `gorm:"size:500;not null" json:"url"`
	URLThumbnail string `gorm:"size:500" json:"url_thumbnail"`
	Descripcion string `gorm:"size:200" json:"descripcion,omitempty"`
	EsPortada   bool   `gorm:"default:false;index" json:"es_portada"`

	// Metadata del archivo
	FileName   string `gorm:"size:255" json:"file_name"`
	FileSize   int64  `json:"file_size"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	MimeType   string `gorm:"size:100" json:"mime_type"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (PropertyPhoto) TableName() string {
	return "fotos_propiedad"
}

// BeforeCreate - Hook antes de crear
func (p *PropertyPhoto) BeforeCreate(tx *gorm.DB) error {
	return nil
}
