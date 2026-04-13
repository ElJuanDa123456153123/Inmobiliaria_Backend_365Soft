package models

import (
	"time"

	"gorm.io/gorm"
)

// Category - Categorías para clasificar propiedades
type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Datos de categoría
	Nombre string `gorm:"size:100;uniqueIndex;not null" json:"nombre"`
	Slug   string `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Icono  string `gorm:"size:50" json:"icono,omitempty"`
	Activo bool   `gorm:"default:true;index" json:"activo"`
	Orden  int    `gorm:"default:0" json:"orden"`

	// Relaciones
	Properties []Property `gorm:"many2many:propiedades_categorias;" json:"-"`

	// SEO
	MetaDescripcion string `gorm:"size:500" json:"meta_descripcion,omitempty"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (Category) TableName() string {
	return "categorias"
}

// PropertyCategory - Tabla pivote para relación muchos a muchos
type PropertyCategory struct {
	PropertyID uint `gorm:"primaryKey" json:"property_id"`
	CategoryID uint `gorm:"primaryKey" json:"category_id"`
	CreatedAt  time.Time `json:"created_at"`

	Property *Property `gorm:"foreignKey:PropertyID;constraint:OnDelete:CASCADE" json:"-"`
	Category *Category `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName - Especifica el nombre de la tabla pivote
func (PropertyCategory) TableName() string {
	return "propiedades_categorias"
}
