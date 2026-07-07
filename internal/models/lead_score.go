package models

import (
	"time"

	"gorm.io/gorm"
)

// ScoreLevel - Nivel de calidad del lead
type ScoreLevel string

const (
	ScoreLevelAlto   ScoreLevel = "ALTO"   // 🔥 Lead caliente, alta probabilidad de conversión
	ScoreLevelMedio  ScoreLevel = "MEDIO"  // ⚡ Lead moderado, requiere seguimiento
	ScoreLevelBajo   ScoreLevel = "BAJO"   // ❄️ Lead frío, baja prioridad
)

// LeadScore - Modelo de scoring de clientes/leads
// Calcula automáticamente la calidad de un lead basándose en múltiples factores
type LeadScore struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// === RELACIÓN CON EL CLIENTE ===
	ClientID uint    `gorm:"not null;uniqueIndex" json:"client_id"`
	Client   *Client `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// === PUNTUACIÓN TOTAL (0-100) ===
	// Se calcula automáticamente basándose en los factores
	Score int `gorm:"not null;default:0" json:"score"`

	// === NIVEL DEL LEAD ===
	Level ScoreLevel `gorm:"size:10;not null" json:"level"`

	// === FACTORES DE SCORING (DETALLE) ===
	// Presupuesto definido (+20 puntos)
	HasBudget bool `gorm:"default:false" json:"has_budget"`

	// Zona específica definida (+15 puntos)
	HasZone bool `gorm:"default:false" json:"has_zone"`

	// Interacciones en última semana (+20 puntos max, 5pts c/u)
	RecentInteractionsCount int `gorm:"default:0" json:"recent_interactions_count"`

	// Visto propiedades (+10 puntos)
	HasViewedProperties bool `gorm:"default:false" json:"has_viewed_properties"`

	// Favoritos marcados (+10 puntos)
	HasFavorites bool `gorm:"default:false" json:"has_favorites"`

	// Propiedad de interés específico (+10 puntos)
	HasSpecificProperty bool `gorm:"default:false" json:"has_specific_property"`

	// Origen del lead (Web/WhatsApp = +5, Referencia = +10)
	OriginBonus int `gorm:"default:0" json:"origin_bonus"`

	// Estado del pipeline (Contactado+ = +5, Interesado+ = +10)
	StateBonus int `gorm:"default:0" json:"state_bonus"`

	// === METADATOS ADICIONALES ===
	// Última fecha de cálculo
	LastCalculatedAt *time.Time `json:"last_calculated_at"`

	// Última fecha de interacción
	LastInteractionAt *time.Time `json:"last_interaction_at"`

	// Días desde la última interacción
	DaysSinceLastInteraction int `json:"days_since_last_interaction"`
}

// TableName - Especifica el nombre de la tabla en PostgreSQL
func (LeadScore) TableName() string {
	return "lead_scores"
}

// GetLevelColor - Retorna el color del nivel para el frontend
func (ls *LeadScore) GetLevelColor() string {
	switch ls.Level {
	case ScoreLevelAlto:
		return "green"
	case ScoreLevelMedio:
		return "yellow"
	case ScoreLevelBajo:
		return "red"
	default:
		return "gray"
	}
}

// GetLevelIcon - Retorna el icono del nivel para el frontend
func (ls *LeadScore) GetLevelIcon() string {
	switch ls.Level {
	case ScoreLevelAlto:
		return "🔥"
	case ScoreLevelMedio:
		return "⚡"
	case ScoreLevelBajo:
		return "❄️"
	default:
		return "❓"
	}
}

// CalculateLevel - Calcula el nivel basado en el score
func (ls *LeadScore) CalculateLevel() ScoreLevel {
	if ls.Score >= 60 {
		return ScoreLevelAlto
	} else if ls.Score >= 30 {
		return ScoreLevelMedio
	}
	return ScoreLevelBajo
}

// IsStale - Retorna true si el scoring necesita recálculo
// (más de 7 días desde el último cálculo)
func (ls *LeadScore) IsStale() bool {
	if ls.LastCalculatedAt == nil {
		return true
	}
	return time.Since(*ls.LastCalculatedAt) > 7*24*time.Hour
}

// BeforeCreate - Hook de GORM antes de crear
func (ls *LeadScore) BeforeCreate(tx *gorm.DB) error {
	ls.Level = ls.CalculateLevel()
	now := time.Now()
	ls.LastCalculatedAt = &now
	return nil
}

// BeforeUpdate - Hook de GORM antes de actualizar
func (ls *LeadScore) BeforeUpdate(tx *gorm.DB) error {
	ls.Level = ls.CalculateLevel()
	now := time.Now()
	ls.LastCalculatedAt = &now
	return nil
}
