package service

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"gorm.io/gorm"
)

// AssignmentService - Servicio para la asignación automática de propiedades a asesores
type AssignmentService struct {
	AdvisorRepo  repository.AdvisorRepositoryInterface
	PropertyRepo repository.PropertyRepositoryInterface
	NotificationRepo repository.NotificationRepositoryInterface
	db *gorm.DB
}

// NewAssignmentService - Crea un nuevo servicio de asignación
func NewAssignmentService(
	advisorRepo repository.AdvisorRepositoryInterface,
	propertyRepo repository.PropertyRepositoryInterface,
	notificationRepo repository.NotificationRepositoryInterface,
	db *gorm.DB,
) *AssignmentService {
	return &AssignmentService{
		AdvisorRepo:     advisorRepo,
		PropertyRepo:    propertyRepo,
		NotificationRepo: notificationRepo,
		db:              db,
	}
}

// AssignToLeastLoadedAdvisor - Asigna una propiedad al asesor con menos propiedades activas
// Esta función se llama automáticamente cuando un particular publica una propiedad
func (s *AssignmentService) AssignToLeastLoadedAdvisor(property *models.Property, userRole models.UserRole) error {
	// Solo asignar si el usuario es "particular" (rol "usuario")
	if userRole != models.RoleUsuario {
		fmt.Printf("⏭️  Usuario con rol '%s' no requiere asignación de asesor\n", userRole)
		return nil
	}

	// Obtener todos los asesores activos
	asesores, err := s.AdvisorRepo.GetAllActive()
	if err != nil {
		return fmt.Errorf("error obteniendo asesores activos: %w", err)
	}

	// Si no hay asesores activos, no asignar
	if len(asesores) == 0 {
		fmt.Printf("⚠️  No hay asesores activos disponibles para asignar\n")
		return nil
	}

	// Encontrar el asesor con menos propiedades activas
	asesorSeleccionado := s.findAdvisorWithLeastLoad(asesores)

	// Asignar la propiedad al asesor
	property.AdvisorID = &asesorSeleccionado.ID
	property.EstadoPropiedad = models.PropertyStatusPendiente // Cambiar a pendiente para que el asesor la revise

	// Guardar cambios
	if err := s.PropertyRepo.Update(property); err != nil {
		return fmt.Errorf("error asignando propiedad al asesor: %w", err)
	}

	// Crear notificación para el asesor
	if err := s.createAdvisorNotification(asesorSeleccionado, property); err != nil {
		fmt.Printf("⚠️  Error creando notificación: %v\n", err)
		// No fallar si falla la notificación
	}

	fmt.Printf("✅ Propiedad ID %d asignada al asesor %s (ID: %d)\n",
		property.ID, asesorSeleccionado.Nombre, asesorSeleccionado.ID)

	return nil
}

// findAdvisorWithLeastLoad - Encuentra el asesor con menos propiedades activas
func (s *AssignmentService) findAdvisorWithLeastLoad(asesores []models.Advisor) *models.Advisor {
	var selectedAdvisor *models.Advisor
	minProperties := int64(999999)

	for i := range asesores {
		// Contar propiedades activas del asesor
		count, err := s.AdvisorRepo.CountActivePropertiesByAdvisorID(asesores[i].ID)
		if err != nil {
			fmt.Printf("⚠️  Error contando propiedades del asesor %d: %v\n", asesores[i].ID, err)
			continue
		}

		fmt.Printf("📊 Asesor %s (ID: %d) tiene %d propiedades activas\n",
			asesores[i].Nombre, asesores[i].ID, count)

		// Si este asesor tiene menos propiedades, seleccionarlo
		if count < minProperties {
			minProperties = count
			selectedAdvisor = &asesores[i]
		}
	}

	if selectedAdvisor == nil {
		// Si por alguna razón no se pudo seleccionar, tomar el primero
		selectedAdvisor = &asesores[0]
	}

	return selectedAdvisor
}

// createAdvisorNotification - Crea una notificación para el asesor
func (s *AssignmentService) createAdvisorNotification(asesor *models.Advisor, property *models.Property) error {
	// Si el asesor no tiene UserID asociado, no crear notificación
	if asesor.UserID == nil {
		fmt.Printf("⚠️  Asesor %s no tiene UserID asociado, no se crea notificación\n", asesor.Nombre)
		return nil
	}

	// Crear notificación
	notif := models.Notification{
		UserID:        *asesor.UserID, // Desreferenciar el puntero
		PropertyID:    &property.ID,
		Tipo:          "propiedad_asignada",
		Titulo:        "Nueva Propiedad Asignada",
		Mensaje:       fmt.Sprintf("Tienes una nueva propiedad asignada: %s", property.Titulo),
		IsRead:        false,
	}

	// Crear la notificación
	if err := s.NotificationRepo.Create(&notif); err != nil {
		return err
	}
	fmt.Printf("📬 Notificación creada para el asesor %s\n", asesor.Nombre)

	return nil
}

// ActivateProperty - Activa una propiedad (solo el asesor asignado puede hacerlo)
// Cambia el estado de "pendiente" a "activo"
func (s *AssignmentService) ActivateProperty(propertyID uint, advisorID uint) (*models.Property, error) {
	// Obtener propiedad
	property, err := s.PropertyRepo.GetByID(propertyID)
	if err != nil {
		return nil, errors.PropertyNotFound(propertyID)
	}

	// Verificar que la propiedad esté asignada a este asesor
	if property.AdvisorID == nil || *property.AdvisorID != advisorID {
		return nil, errors.Forbidden("No tienes permiso para activar esta propiedad")
	}

	// Verificar que la propiedad esté en estado pendiente
	if property.EstadoPropiedad != models.PropertyStatusPendiente {
		return nil, errors.ValidationError("estado", "Solo se pueden activar propiedades en estado pendiente")
	}

	// Activar propiedad
	property.EstadoPropiedad = models.PropertyStatusActivo
	now := time.Now()
	property.FechaPublicacion = &now

	if err := s.PropertyRepo.Update(property); err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("✅ Propiedad ID %d activada por el asesor %d\n", propertyID, advisorID)

	return property, nil
}

// GetAdvisorProperties - Obtiene las propiedades de un asesor filtradas por estado
func (s *AssignmentService) GetAdvisorProperties(advisorID uint, status models.PropertyStatus) ([]models.Property, error) {
	properties, err := s.AdvisorRepo.GetPropertiesByAdvisorID(advisorID, status)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return properties, nil
}

// GetAdvisorStats - Obtiene las estadísticas de un asesor
type AdvisorStats struct {
	PorGestionar int64 `json:"por_gestionar"` // Propiedades pendientes
	Activas      int64 `json:"activas"`      // Propiedades activas
	Vendidas     int64 `json:"vendidas"`     // Propiedades vendidas
	Total        int64 `json:"total"`        // Total de propiedades
}

func (s *AssignmentService) GetAdvisorStats(advisorID uint) (*AdvisorStats, error) {
	stats := &AdvisorStats{}

	// Contar propiedades por estado
	var err error

	stats.PorGestionar, err = s.AdvisorRepo.CountActivePropertiesByAdvisorID(advisorID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	// Contar activas
	s.db.Model(&models.Property{}).
		Where("advisor_id = ? AND estado_propiedad = ?", advisorID, models.PropertyStatusActivo).
		Count(&stats.Activas)

	// Contar vendidas
	s.db.Model(&models.Property{}).
		Where("advisor_id = ? AND estado_propiedad = ?", advisorID, models.PropertyStatusVendido).
		Count(&stats.Vendidas)

	// Total
	stats.Total = stats.PorGestionar + stats.Activas + stats.Vendidas

	return stats, nil
}
