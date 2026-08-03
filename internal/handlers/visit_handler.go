package handlers

import (
	"log"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// VisitHandler - Handler para visitas agendadas
type VisitHandler struct {
	DB *gorm.DB
}

// NewVisitHandler - Crea un nuevo handler de visitas
func NewVisitHandler() (*VisitHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &VisitHandler{
		DB: dbConfig.DB,
	}, nil
}

// ScheduleVisitRequest - Estructura para agendar visita
type ScheduleVisitRequest struct {
	PropertyID       uint   `json:"property_id" validate:"required"`
	NombreCompleto   string `json:"nombre_completo" validate:"required"`
	Telefono         string `json:"telefono" validate:"required"`
	HorarioPreferido string `json:"horario_preferido" validate:"required"`
	Notas            string `json:"notas,omitempty"`
}

// ScheduleVisit - Agenda una visita a una propiedad
// POST /api/visits
func (h *VisitHandler) ScheduleVisit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Parsear body
	var req ScheduleVisitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar
	if req.PropertyID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "property_id es requerido",
		})
	}

	if req.NombreCompleto == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "nombre_completo es requerido",
		})
	}

	if req.Telefono == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "telefono es requerido",
		})
	}

	if req.HorarioPreferido == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "horario_preferido es requerido",
		})
	}

	// Verificar que la propiedad existe
	var prop models.Property
	if err := h.DB.First(&prop, req.PropertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// No agendar visita a propia propiedad
	if prop.UserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes agendar visita a tu propia propiedad",
		})
	}

	// Crear visita con fecha nula (para que el vendedor la confirme)
	visit := models.Visit{
		PropertyID:       req.PropertyID,
		InquilinoID:      userID,
		VendedorID:       prop.UserID,
		FechaVisita:      time.Time{}, // Fecha nula, será asignada por el vendedor
		Estado:           models.VisitStatusPendiente,
		Notas:            req.HorarioPreferido + "\nNombre: " + req.NombreCompleto + "\nTeléfono: " + req.Telefono,
	}

	if req.Notas != "" {
		visit.Notas += "\nNotas adicionales: " + req.Notas
	}

	if err := h.DB.Create(&visit).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error agendando visita",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&visit, visit.ID)

	// 🔔 Crear notificación para el vendedor
	notificacion := models.Notification{
		UserID:     visit.VendedorID,
		PropertyID: &visit.PropertyID,
		Tipo:       "visita_solicitada",
		Titulo:     "Nueva solicitud de visita",
		Mensaje:    req.NombreCompleto + " solicitó visitar \"" + prop.Titulo + "\"",
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	// 📧 Enviar email al vendedor
	go func() {
		emailService := services.NewEmailService()
		emailService.SendVisitRequestEmail(
			prop.User.Email,
			prop.User.Nombre,
			prop.Titulo,
			req.NombreCompleto,
			req.Telefono,
			req.HorarioPreferido,
		)
	}()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Solicitud de visita enviada correctamente",
		"visit":   visit,
	})
}

// GetPropertyVisits - Obtiene visitas de una propiedad (para el vendedor)
// GET /api/properties/:id/visits
func (h *VisitHandler) GetPropertyVisits(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener property_id del parámetro
	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	// Verificar si la propiedad existe y pertenece al usuario
	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", propertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada o no tienes permisos",
		})
	}

	// Obtener visitas
	var visits []models.Visit
	if err := h.DB.Where("property_id = ?", propertyID).
		Preload("Inquilino").
		Order("fecha_visita ASC").
		Find(&visits).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo visitas",
		})
	}

	return c.JSON(fiber.Map{
		"visitas": visits,
		"total":   len(visits),
	})
}

// GetMyVisits - Obtiene mis visitas (como interesado)
// GET /api/visits/my
func (h *VisitHandler) GetMyVisits(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de filtrado
	estado := c.Query("estado", "")

	// Construir query
	query := h.DB.Where("inquilino_id = ?", userID)

	// Aplicar filtros
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	// Obtener visitas
	var visits []models.Visit
	if err := query.Preload("Property").Preload("Vendedor").
		Order("fecha_visita ASC").
		Find(&visits).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo visitas",
		})
	}

	return c.JSON(fiber.Map{
		"visitas": visits,
		"total":   len(visits),
	})
}

// GetReceivedVisits - Obtiene visitas recibidas (como vendedor/propietario)
// GET /api/visits/received
func (h *VisitHandler) GetReceivedVisits(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de filtrado
	estado := c.Query("estado", "")

	// Construir query
	query := h.DB.Where("vendedor_id = ?", userID)

	// Aplicar filtros
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	// Obtener visitas
	var visits []models.Visit
	if err := query.Preload("Property").Preload("Inquilino").
		Order("created_at DESC").
		Find(&visits).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo visitas",
		})
	}

	return c.JSON(fiber.Map{
		"visitas": visits,
		"total":   len(visits),
	})
}

// UpdateVisitStatusRequest - Estructura para actualizar estado de visita
type UpdateVisitStatusRequest struct {
	Estado string `json:"estado" validate:"required"`
	Notas  string `json:"notas,omitempty"`
}

// UpdateVisitStatus - Actualiza el estado de una visita
// PATCH /api/visits/:id
func (h *VisitHandler) UpdateVisitStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	visitID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de visita inválido",
		})
	}

	// Parsear body
	var req UpdateVisitStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar estado
	if req.Estado != string(models.VisitStatusPendiente) &&
		req.Estado != string(models.VisitStatusCompletada) &&
		req.Estado != string(models.VisitStatusCancelada) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado inválido (debe ser: pendiente, completada, cancelada)",
		})
	}

	// Verificar que la visita existe y pertenece al usuario (como vendedor o inquilino)
	var visit models.Visit
	if err := h.DB.Where("id = ? AND (vendedor_id = ? OR inquilino_id = ?)", visitID, userID, userID).First(&visit).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Visita no encontrada",
		})
	}

	// Actualizar
	visit.Estado = models.VisitStatus(req.Estado)
	if req.Notas != "" {
		visit.Notas = req.Notas
	}

	if err := h.DB.Save(&visit).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando visita",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&visit, visit.ID)

	// 📧 Enviar email al cliente si la visita fue cancelada
	if req.Estado == string(models.VisitStatusCancelada) {
		// 🔔 Crear notificación para el cliente
		notificacion := models.Notification{
			UserID:     visit.InquilinoID,
			PropertyID: &visit.PropertyID,
			Tipo:       "visita_cancelada",
			Titulo:     "Visita cancelada",
			Mensaje:    "Tu visita para \"" + visit.Property.Titulo + "\" fue cancelada por el vendedor. Revisa tu correo.",
			IsRead:     false,
			Metadata:   `{"gmail_url":"https://mail.google.com/mail/u/` + visit.Inquilino.Email + `/#inbox"}`,
		}
		h.DB.Create(&notificacion)

		go func() {
			emailService := services.NewEmailService()
			emailService.SendVisitCancelledEmail(
				visit.Inquilino.Email,
				visit.Inquilino.Nombre,
				visit.Property.Titulo,
			)
		}()
	}

	return c.JSON(fiber.Map{
		"message": "Estado de visita actualizado",
		"visita":  visit,
	})
}

// ConfirmVisitRequest - Estructura para confirmar visita con fecha específica
type ConfirmVisitRequest struct {
	FechaVisita string `json:"fecha_visita" validate:"required"` // Formato: "2024-01-20 10:00"
}

// ConfirmVisit - Confirma una visita estableciendo fecha y hora específicas
// PATCH /api/visits/:id/confirm
func (h *VisitHandler) ConfirmVisit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	visitID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de visita inválido",
		})
	}

	// Parsear body
	var req ConfirmVisitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar
	if req.FechaVisita == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fecha_visita es requerida (formato: '2006-01-02 15:04')",
		})
	}

	// Parsear fecha
	fechaVisita, err := time.Parse("2006-01-02 15:04", req.FechaVisita)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha inválido (use: 'AAAA-MM-DD HH:MM')",
		})
	}

	// Verificar que la visita existe y pertenece al usuario (como vendedor)
	var visit models.Visit
	if err := h.DB.Where("id = ? AND vendedor_id = ?", visitID, userID).First(&visit).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Visita no encontrada o no tienes permisos",
		})
	}

	// Verificar que esté pendiente
	if visit.Estado != models.VisitStatusPendiente {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Solo se pueden confirmar visitas pendientes",
		})
	}

	// Actualizar visita
	visit.FechaVisita = fechaVisita
	visit.Estado = models.VisitStatusCompletada

	if err := h.DB.Save(&visit).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error confirmando visita",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&visit, visit.ID)

	// 🔔 Crear notificación para el cliente (con enlace a Gmail)
	notificacion := models.Notification{
		UserID:     visit.InquilinoID,
		PropertyID: &visit.PropertyID,
		Tipo:       "visita_confirmada",
		Titulo:     "Visita confirmada",
		Mensaje:    "Tu visita para \"" + visit.Property.Titulo + "\" fue confirmada. Revisa tu correo para más detalles.",
		IsRead:     false,
		Metadata:   `{"gmail_url":"https://mail.google.com/mail/u/` + visit.Inquilino.Email + `/#inbox"}`,
	}
	h.DB.Create(&notificacion)

	// 📧 Enviar email al cliente confirmando la visita
	go func() {
		log.Printf("📧 ConfirmVisit - Inquilino: %s, Email: %s, Propiedad: %s",
			visit.Inquilino.Nombre, visit.Inquilino.Email, visit.Property.Titulo)
		emailService := services.NewEmailService()
		if visit.Inquilino.Email == "" {
			log.Printf("❌ ConfirmVisit - Email del inquilino está vacío, no se puede enviar")
			return
		}
		fechaFormateada := visit.FechaVisita.Format("02/01/2006 15:04")
		err := emailService.SendVisitConfirmedEmail(
			visit.Inquilino.Email,
			visit.Inquilino.Nombre,
			visit.Property.Titulo,
			fechaFormateada,
			visit.Vendedor.Nombre,
		)
		if err != nil {
			log.Printf("❌ ConfirmVisit - Error enviando email: %v", err)
		}
	}()

	return c.JSON(fiber.Map{
		"message": "Visita confirmada correctamente",
		"visit":   visit,
	})
}
