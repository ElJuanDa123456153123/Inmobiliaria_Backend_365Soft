package models

import (
	"time"

	"gorm.io/gorm"
)

// Subscription - Modelo de suscripción de usuario para publicar propiedades
type Subscription struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Usuario suscrito (puede tener múltiples suscripciones)
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Plan contratado
	PlanID uint `gorm:"not null" json:"plan_id"`
	Plan   *Plan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`

	// Fechas de la suscripción
	FechaInicio time.Time  `gorm:"not null" json:"fecha_inicio"`
	FechaFin    *time.Time `json:"fecha_fin"` // Null si es recurrente

	// Contador de propiedades publicadas
	PropiedadesUsadas int `gorm:"not null;default:0" json:"propiedades_usadas"`

	// Estado: ACTIVA, VENCIDA, CANCELADA
	Estado string `gorm:"size:20;not null;default:'ACTIVA';index" json:"estado"`

	// Si está activa para renovación automática
	RenovacionAutomatica bool `gorm:"default:true" json:"renovacion_automatica"`

	// Si esta suscripción está seleccionada para publicar (solo una puede estar seleccionada por usuario)
	Seleccionada bool `gorm:"default:false;index" json:"seleccionada"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (Subscription) TableName() string {
	return "subscriptions"
}

// PropiedadesDisponibles retorna cuántas propiedades puede publicar aún
func (s *Subscription) PropiedadesDisponibles() int {
	if s.Plan == nil {
		return 0
	}
	maxPropiedades := s.Plan.LimitePropiedades
	restantes := maxPropiedades - s.PropiedadesUsadas
	if restantes < 0 {
		return 0
	}
	return restantes
}

// EstaVencida retorna true si la suscripción está vencida
func (s *Subscription) EstaVencida() bool {
	if s.FechaFin == nil {
		return false // Suscripción indefinida
	}
	return s.FechaFin.Before(time.Now())
}

// PuedePublicar retorna true si el usuario puede publicar más propiedades
func (s *Subscription) PuedePublicar() bool {
	if s.Estado != "ACTIVA" {
		return false
	}
	if s.EstaVencida() {
		return false
	}
	return s.PropiedadesDisponibles() > 0
}
