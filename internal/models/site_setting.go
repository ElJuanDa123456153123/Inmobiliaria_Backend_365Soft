package models

import "time"

// SiteSetting - Configuración global del sitio (clave-valor)
// Ej: accent_color para el color de acento configurado por el admin
type SiteSetting struct {
	Key       string    `gorm:"primaryKey;size:100;column:setting_key" json:"key"`
	Value     string    `gorm:"size:255;column:setting_value" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (SiteSetting) TableName() string {
	return "site_settings"
}
