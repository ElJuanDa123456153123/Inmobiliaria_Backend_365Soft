package services

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"gorm.io/gorm"
)

// CrmService - Servicio de lógica de negocio para el CRM
type CrmService struct {
	ClientRepo    repository.ClientRepositoryInterface
	InteractionRepo repository.InteractionRepositoryInterface
	db            *gorm.DB
}

// NewCrmService - Crea un nuevo servicio de CRM
func NewCrmService(
	clientRepo repository.ClientRepositoryInterface,
	interactionRepo repository.InteractionRepositoryInterface,
	db *gorm.DB,
) *CrmService {
	return &CrmService{
		ClientRepo:    clientRepo,
		InteractionRepo: interactionRepo,
		db:            db,
	}
}

// ==================== CLIENTES ====================

// CreateClientRequest - Request para crear un cliente
type CreateClientRequest struct {
	AdvisorID     uint                `json:"advisor_id"`
	Nombre        string              `json:"nombre"`
	Telefono      string              `json:"telefono"`
	WhatsApp      string              `json:"whatsapp,omitempty"`
	Email         string              `json:"email,omitempty"`
	Estado        models.ClientStatus `json:"estado"`
	Origen        models.ClientOrigin `json:"origen"`
	PropertyID    *uint               `json:"property_id,omitempty"`
	Notas         string              `json:"notas,omitempty"`
	Presupuesto   *float64            `json:"presupuesto,omitempty"`
	ZonaPreferida string              `json:"zona_preferida,omitempty"`
}

// CreateClient - Crea un nuevo cliente
func (s *CrmService) CreateClient(req *CreateClientRequest) (*models.Client, error) {
	// Validaciones
	if err := s.validateClientRequest(req); err != nil {
		return nil, err
	}

	// Crear cliente
	client := &models.Client{
		Nombre:        req.Nombre,
		Telefono:      req.Telefono,
		WhatsApp:      req.WhatsApp,
		Email:         req.Email,
		Estado:        req.Estado,
		Origen:        req.Origen,
		PropertyID:    req.PropertyID,
		Notas:         req.Notas,
		AdvisorID:     req.AdvisorID,
		Presupuesto:   req.Presupuesto,
		ZonaPreferida: req.ZonaPreferida,
	}

	// Auto-vincular con usuario de la plataforma si coincide email/teléfono
	s.linkClientToUser(client)

	if err := s.ClientRepo.Create(client); err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("✅ Cliente creado: %s (ID: %d)\n", client.Nombre, client.ID)
	return client, nil
}

// linkClientToUser - Intenta vincular el cliente con un usuario registrado de la
// plataforma por coincidencia de email o teléfono (últimos 8 dígitos).
// Permite que el lead scoring use la actividad real (vistas, favoritos, búsquedas).
func (s *CrmService) linkClientToUser(client *models.Client) {
	if client.UserID != nil {
		return // ya vinculado
	}

	// 1. Por email
	if client.Email != "" {
		var user models.User
		if err := s.db.Where("email = ? AND deleted_at IS NULL", client.Email).First(&user).Error; err == nil {
			client.UserID = &user.ID
			fmt.Printf("🔗 Cliente vinculado a usuario %d por email\n", user.ID)
			return
		}
	}

	// 2. Por teléfono (comparando últimos 8 dígitos, ignorando formato)
	phoneDigits := normalizePhone(client.Telefono)
	if phoneDigits == "" {
		phoneDigits = normalizePhone(client.WhatsApp)
	}
	if len(phoneDigits) >= 8 {
		suffix := phoneDigits[len(phoneDigits)-8:]
		var user models.User
		if err := s.db.Where(
			"RIGHT(REGEXP_REPLACE(telefono, '[^0-9]', '', 'g'), 8) = ? AND deleted_at IS NULL",
			suffix,
		).First(&user).Error; err == nil {
			client.UserID = &user.ID
			fmt.Printf("🔗 Cliente vinculado a usuario %d por teléfono\n", user.ID)
		}
	}
}

// normalizePhone - Deja solo dígitos del teléfono
func normalizePhone(phone string) string {
	digits := make([]rune, 0, len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	return string(digits)
}

// GetClientByID - Obtiene un cliente por ID
func (s *CrmService) GetClientByID(clientID uint) (*models.Client, error) {
	client, err := s.ClientRepo.GetWithInteractions(clientID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Cliente no encontrado")
		}
		return nil, errors.DatabaseError(err)
	}
	return client, nil
}

// UpdateClientRequest - Request para actualizar un cliente
type UpdateClientRequest struct {
	ClientID        uint                  `json:"client_id"`
	Nombre          *string               `json:"nombre,omitempty"`
	Telefono        *string               `json:"telefono,omitempty"`
	WhatsApp        *string               `json:"whatsapp,omitempty"`
	Email           *string               `json:"email,omitempty"`
	Estado          *models.ClientStatus  `json:"estado,omitempty"`
	Notas           *string               `json:"notas,omitempty"`
	ProximoContacto *time.Time            `json:"proximo_contacto,omitempty"`
	Presupuesto     *float64              `json:"presupuesto,omitempty"`
	ZonaPreferida   *string               `json:"zona_preferida,omitempty"`
}

// UpdateClient - Actualiza un cliente
func (s *CrmService) UpdateClient(req *UpdateClientRequest) (*models.Client, error) {
	// Obtener cliente
	client, err := s.ClientRepo.GetByID(req.ClientID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Cliente no encontrado")
		}
		return nil, errors.DatabaseError(err)
	}

	// Actualizar campos si se proporcionan
	if req.Nombre != nil {
		client.Nombre = *req.Nombre
	}
	if req.Telefono != nil {
		client.Telefono = *req.Telefono
	}
	if req.WhatsApp != nil {
		client.WhatsApp = *req.WhatsApp
	}
	if req.Email != nil {
		client.Email = *req.Email
	}
	if req.Estado != nil {
		client.Estado = *req.Estado
	}
	if req.Notas != nil {
		client.Notas = *req.Notas
	}
	if req.ProximoContacto != nil {
		client.ProximoContacto = req.ProximoContacto
	}
	if req.Presupuesto != nil {
		client.Presupuesto = req.Presupuesto
	}
	if req.ZonaPreferida != nil {
		client.ZonaPreferida = *req.ZonaPreferida
	}

	// Reintentar vinculación si cambiaron email o teléfono
	s.linkClientToUser(client)

	// Guardar cambios
	if err := s.ClientRepo.Update(client); err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("✅ Cliente actualizado: %s (ID: %d)\n", client.Nombre, client.ID)
	return client, nil
}

// DeleteClient - Elimina un cliente
func (s *CrmService) DeleteClient(clientID uint) error {
	if err := s.ClientRepo.Delete(clientID); err != nil {
		return errors.DatabaseError(err)
	}
	fmt.Printf("✅ Cliente eliminado (ID: %d)\n", clientID)
	return nil
}

// ListClientsRequest - Request para listar clientes
type ListClientsRequest struct {
	AdvisorID uint                    `json:"advisor_id"`
	Status    *models.ClientStatus   `json:"status,omitempty"`
	Query     string                  `json:"query,omitempty"`
	Page      int                     `json:"page"`
	Limit     int                     `json:"limit"`
}

// ListClients - Lista clientes con filtros
func (s *CrmService) ListClients(req *ListClientsRequest) ([]models.Client, int64, error) {
	// Validar paginación
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Si hay búsqueda por nombre
	if req.Query != "" {
		clients, err := s.ClientRepo.SearchByName(req.AdvisorID, req.Query)
		return clients, int64(len(clients)), err
	}

	// Si hay filtro por estado
	if req.Status != nil {
		clients, err := s.ClientRepo.GetByStatus(req.AdvisorID, *req.Status)
		return clients, int64(len(clients)), err
	}

	// Listado general paginado
	return s.ClientRepo.GetByAdvisorIDWithPagination(req.AdvisorID, req.Page, req.Limit)
}

// GetFunnelStats - Obtiene las estadísticas del embudo de ventas
func (s *CrmService) GetFunnelStats(advisorID uint) (*repository.FunnelStats, error) {
	stats, err := s.ClientRepo.GetFunnelStats(advisorID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	// Calcular tasa de conversión
	if stats.Nuevos > 0 {
		fmt.Printf("📊 Tasa de conversión: %.2f%% (%d de %d)\n",
			float64(stats.Cerrados)/float64(stats.Nuevos)*100, stats.Cerrados, stats.Nuevos)
	}

	return stats, nil
}

// GetUpcomingContacts - Obtiene próximos contactos pendientes
func (s *CrmService) GetUpcomingContacts(advisorID uint, days int) ([]models.Client, error) {
	clients, err := s.ClientRepo.GetUpcomingContacts(advisorID, days)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("📅 %d próximos contactos en los próximos %d días\n", len(clients), days)
	return clients, nil
}

// ==================== INTERACCIONES ====================

// CreateInteractionRequest - Request para crear una interacción
type CreateInteractionRequest struct {
	ClientID        uint                      `json:"client_id"`
	AdvisorID       uint                      `json:"advisor_id"`
	Tipo            models.InteractionType    `json:"tipo"`
	Notas           string                    `json:"notas"`
	ProximoContacto *time.Time                `json:"proximo_contacto,omitempty"`
	ClienteEstado   *models.ClientStatus      `json:"cliente_estado,omitempty"`
}

// CreateInteraction - Crea una nueva interacción
func (s *CrmService) CreateInteraction(req *CreateInteractionRequest) (*models.Interaction, error) {
	// Validaciones
	if req.ClientID == 0 {
		return nil, errors.ValidationError("client_id", "El ID del cliente es requerido")
	}
	if req.AdvisorID == 0 {
		return nil, errors.ValidationError("advisor_id", "El ID del asesor es requerido")
	}
	if req.Notas == "" {
		return nil, errors.ValidationError("notas", "Las notas son requeridas")
	}

	// Verificar que el cliente existe
	client, err := s.ClientRepo.GetByID(req.ClientID)
	if err != nil {
		return nil, errors.NotFound("Cliente no encontrado")
	}

	// Crear interacción
	interaction := &models.Interaction{
		ClientID:        req.ClientID,
		AdvisorID:       req.AdvisorID,
		Tipo:            req.Tipo,
		Notas:           req.Notas,
		ProximoContacto:  req.ProximoContacto,
	}

	if req.ClienteEstado != nil {
		interaction.ClienteEstado = *req.ClienteEstado
		// Actualizar estado del cliente
		client.Estado = *req.ClienteEstado
	}

	// Actualizar próximo contacto del cliente si se proporcionó
	if req.ProximoContacto != nil {
		client.ProximoContacto = req.ProximoContacto
	}

	// Iniciar transacción
	tx := s.db.Begin()

	// Guardar interacción
	if err := tx.Create(interaction).Error; err != nil {
		tx.Rollback()
		return nil, errors.DatabaseError(err)
	}

	// Actualizar cliente
	if err := tx.Save(client).Error; err != nil {
		tx.Rollback()
		return nil, errors.DatabaseError(err)
	}

	// Confirmar transacción
	if err := tx.Commit().Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("✅ Interacción creada: %s para cliente %d\n", interaction.Tipo, interaction.ClientID)
	return interaction, nil
}

// GetInteractionsByClient - Obtiene las interacciones de un cliente
func (s *CrmService) GetInteractionsByClient(clientID uint) ([]models.Interaction, error) {
	interactions, err := s.ClientRepo.GetTimeline(clientID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return interactions, nil
}

// ==================== VALIDACIONES ====================

// validateClientRequest - Valida el request de crear cliente
func (s *CrmService) validateClientRequest(req *CreateClientRequest) error {
	if req.AdvisorID == 0 {
		return errors.ValidationError("advisor_id", "El ID del asesor es requerido")
	}
	if req.Nombre == "" {
		return errors.ValidationError("nombre", "El nombre es requerido")
	}
	if len(req.Nombre) < 3 {
		return errors.ValidationError("nombre", "El nombre debe tener al menos 3 caracteres")
	}
	if req.Telefono == "" {
		return errors.ValidationError("telefono", "El teléfono es requerido")
	}
	if len(req.Telefono) < 8 {
		return errors.ValidationError("telefono", "El teléfono debe tener al menos 8 caracteres")
	}
	if req.Estado == "" {
		req.Estado = models.ClientStatusNuevo // Default
	}
	if req.Origen == "" {
		req.Origen = models.ClientOriginWeb // Default
	}

	// Validar que el estado sea válido
	validStatuses := map[models.ClientStatus]bool{
		models.ClientStatusNuevo:         true,
		models.ClientStatusContactado:    true,
		models.ClientStatusInteresado:    true,
		models.ClientStatusEnNegociacion: true,
		models.ClientStatusCerrado:       true,
		models.ClientStatusPerdido:       true,
	}
	if !validStatuses[req.Estado] {
		return errors.ValidationError("estado", "Estado inválido")
	}

	// Validar que el origen sea válido
	validOrigins := map[models.ClientOrigin]bool{
		models.ClientOriginWeb:        true,
		models.ClientOriginWhatsApp:   true,
		models.ClientOriginReferencia: true,
		models.ClientOriginInstagram:  true,
		models.ClientOriginFacebook:  true,
		models.ClientOriginTelefono:  true,
	}
	if !validOrigins[req.Origen] {
		return errors.ValidationError("origen", "Origen inválido")
	}

	return nil
}
