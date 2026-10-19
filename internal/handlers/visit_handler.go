package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
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
	FechaVisita      string `json:"fecha_visita,omitempty"` // "2026-10-19 15:00" (slot exacto elegido en la agenda)
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

	// 🛡️ Anti-abuso: una sola solicitud pendiente por usuario y propiedad
	var pendientesPropiedad int64
	h.DB.Model(&models.Visit{}).
		Where("inquilino_id = ? AND property_id = ? AND estado = ?", userID, req.PropertyID, models.VisitStatusPendiente).
		Count(&pendientesPropiedad)
	if pendientesPropiedad > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Ya tienes una solicitud de visita pendiente para esta propiedad. Espera la respuesta del propietario o cancela la anterior desde tus visitas.",
		})
	}

	// 🛡️ Anti-abuso: tope global de solicitudes pendientes por usuario
	const maxPendientesPorUsuario = 5
	var pendientesTotales int64
	h.DB.Model(&models.Visit{}).
		Where("inquilino_id = ? AND estado = ?", userID, models.VisitStatusPendiente).
		Count(&pendientesTotales)
	if pendientesTotales >= maxPendientesPorUsuario {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Alcanzaste el límite de 5 solicitudes de visita pendientes. Espera respuestas o cancela alguna para poder agendar nuevas.",
		})
	}

	// Crear visita. Si el cliente eligió un slot de la agenda, se guarda como
	// fecha propuesta (el vendedor solo la confirma o reprograma).
	visit := models.Visit{
		PropertyID:  req.PropertyID,
		InquilinoID: userID,
		VendedorID:  prop.UserID,
		FechaVisita: time.Time{}, // Sin slot elegido: la asignará el vendedor
		Estado:      models.VisitStatusPendiente,
		Notas:       "Nombre: " + req.NombreCompleto + "\nTeléfono: " + req.Telefono,
	}

	if req.FechaVisita != "" {
		fechaPropuesta, err := time.ParseInLocation("2006-01-02 15:04", req.FechaVisita, time.Local)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fecha inválido (use: 'AAAA-MM-DD HH:MM')",
			})
		}
		if !fechaPropuesta.After(time.Now()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "El horario elegido ya pasó. Selecciona otro de la agenda",
			})
		}
		if err := validarSlotDisponible(h.DB, req.PropertyID, prop.UserID, fechaPropuesta, 0); err != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		visit.FechaVisita = fechaPropuesta
	} else {
		visit.Notas += "\nHorario preferido: " + req.HorarioPreferido
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
	mensajeSolicitud := req.NombreCompleto + " solicitó visitar \"" + prop.Titulo + "\""
	if !visit.FechaVisita.IsZero() {
		mensajeSolicitud += " para el " + visit.FechaVisita.Format("02/01/2006 a las 15:04")
	}
	notificacion := models.Notification{
		UserID:     visit.VendedorID,
		PropertyID: &visit.PropertyID,
		Tipo:       "visita_solicitada",
		Titulo:     "Nueva solicitud de visita",
		Mensaje:    mensajeSolicitud,
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	// 📧 Enviar email al vendedor
	fechaEmail := req.HorarioPreferido
	if !visit.FechaVisita.IsZero() {
		fechaEmail = visit.FechaVisita.Format("02/01/2006 15:04")
	}
	go func() {
		emailService := services.NewEmailService()
		emailService.SendVisitRequestEmail(
			prop.User.Email,
			prop.User.Nombre,
			prop.Titulo,
			req.NombreCompleto,
			req.Telefono,
			fechaEmail,
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
		req.Estado != string(models.VisitStatusConfirmada) &&
		req.Estado != string(models.VisitStatusCompletada) &&
		req.Estado != string(models.VisitStatusCancelada) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado inválido (debe ser: pendiente, confirmada, completada, cancelada)",
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

// ConfirmVisitRequest - Estructura para confirmar visita
// La fecha es OPCIONAL: si no se envía se usa la propuesta por el cliente.
type ConfirmVisitRequest struct {
	FechaVisita string `json:"fecha_visita,omitempty"` // Formato: "2024-01-20 10:00"
}

// ConfirmVisit - Confirma una visita, opcionalmente ajustando la fecha
// PATCH /api/visits/:id/confirm
func (h *VisitHandler) ConfirmVisit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	visitID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de visita inválido",
		})
	}

	// Parsear body (puede venir vacío)
	var req ConfirmVisitRequest
	if err := c.BodyParser(&req); err != nil {
		req = ConfirmVisitRequest{}
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

	// Fecha: la propuesta del cliente, o la nueva si el vendedor la ajustó
	if req.FechaVisita != "" {
		fechaVisita, err := time.ParseInLocation("2006-01-02 15:04", req.FechaVisita, time.Local)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fecha inválido (use: 'AAAA-MM-DD HH:MM')",
			})
		}
		if err := validarSlotDisponible(h.DB, visit.PropertyID, visit.VendedorID, fechaVisita, visit.ID); err != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		visit.FechaVisita = fechaVisita
	}
	if visit.FechaVisita.IsZero() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La visita no tiene fecha propuesta. Indica una fecha para confirmar",
		})
	}

	// Actualizar visita
	visit.Estado = models.VisitStatusConfirmada

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
		Mensaje:    "Tu visita a \"" + visit.Property.Titulo + "\" fue confirmada para el " + visit.FechaVisita.Format("02/01/2006 a las 15:04") + ". Revisa tu correo para más detalles.",
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

// ReprogramarVisitRequest - Nueva fecha propuesta por el vendedor
type ReprogramarVisitRequest struct {
	FechaVisita string `json:"fecha_visita" validate:"required"` // Formato: "2024-01-20 10:00"
}

// ReprogramarVisit - El vendedor propone un nuevo horario; la visita vuelve a
// quedar pendiente hasta que se confirme.
// PATCH /api/visits/:id/reprogram
func (h *VisitHandler) ReprogramarVisit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	visitID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de visita inválido",
		})
	}

	var req ReprogramarVisitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}
	if req.FechaVisita == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fecha_visita es requerida (formato: 'AAAA-MM-DD HH:MM')",
		})
	}

	nuevaFecha, err := time.ParseInLocation("2006-01-02 15:04", req.FechaVisita, time.Local)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha inválido (use: 'AAAA-MM-DD HH:MM')",
		})
	}
	if !nuevaFecha.After(time.Now()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La nueva fecha debe ser futura",
		})
	}

	var visit models.Visit
	if err := h.DB.Where("id = ? AND vendedor_id = ?", visitID, userID).First(&visit).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Visita no encontrada o no tienes permisos",
		})
	}
	if visit.Estado == models.VisitStatusCancelada || visit.Estado == models.VisitStatusCompletada {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se puede reprogramar una visita cancelada o realizada",
		})
	}

	if err := validarSlotDisponible(h.DB, visit.PropertyID, visit.VendedorID, nuevaFecha, visit.ID); err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	visit.FechaVisita = nuevaFecha
	visit.Estado = models.VisitStatusPendiente
	if err := h.DB.Save(&visit).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error reprogramando visita",
		})
	}

	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&visit, visit.ID)

	// 🔔 Notificar al cliente el nuevo horario propuesto
	notificacion := models.Notification{
		UserID:     visit.InquilinoID,
		PropertyID: &visit.PropertyID,
		Tipo:       "visita_reprogramada",
		Titulo:     "Visita reprogramada",
		Mensaje:    "El vendedor propuso reprogramar tu visita a \"" + visit.Property.Titulo + "\" para el " + nuevaFecha.Format("02/01/2006 a las 15:04"),
		IsRead:     false,
	}
	h.DB.Create(&notificacion)

	return c.JSON(fiber.Map{
		"message": "Visita reprogramada. Queda pendiente de confirmación",
		"visit":   visit,
	})
}

// DisponibilidadVisitas - Configuración de horarios de un vendedor, por día
type DisponibilidadVisitas struct {
	DuracionMin int                       `json:"duracion_min"` // Duración de cada visita en minutos
	Horarios    map[string][]RangoHorario `json:"horarios"`     // "0"=Dom .. "6"=Sáb → franjas de ese día
}

// RangoHorario - Franja horaria disponible
type RangoHorario struct {
	Desde string `json:"desde"` // "09:00"
	Hasta string `json:"hasta"` // "12:00"
}

// DisponibilidadPorDefecto - Lun-Sáb, 9-12 y 14-18, visitas de 60 min
func DisponibilidadPorDefecto() DisponibilidadVisitas {
	franjas := []RangoHorario{{Desde: "09:00", Hasta: "12:00"}, {Desde: "14:00", Hasta: "18:00"}}
	return DisponibilidadVisitas{
		DuracionMin: 60,
		Horarios: map[string][]RangoHorario{
			"1": franjas, "2": franjas, "3": franjas, "4": franjas, "5": franjas, "6": franjas,
		},
	}
}

// parseDisponibilidad interpreta el JSON guardado por el usuario.
// Formato nuevo: {"duracion_min":60,"horarios":{"1":[{"desde","hasta"}]}}.
// Formato viejo: {"dias":[1,2],"rangos":[...],"duracion_min":60} → se convierte.
func parseDisponibilidad(rawJSON []byte) DisponibilidadVisitas {
	disponibilidad := DisponibilidadPorDefecto()
	if len(rawJSON) == 0 {
		return disponibilidad
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return disponibilidad
	}

	if _, tieneNuevo := raw["horarios"]; tieneNuevo {
		// IMPORTANTE: partir de un struct limpio. Si se reutiliza el mapa del default,
		// json.Unmarshal SOLO sobrescribe las claves presentes en el JSON y conserva
		// los días del default (ej: usuario configura Lun+Mar → quedarían también Mié-Sáb).
		disponibilidad = DisponibilidadVisitas{DuracionMin: 60}
		if err := json.Unmarshal(rawJSON, &disponibilidad); err != nil {
			return DisponibilidadPorDefecto()
		}
		if disponibilidad.DuracionMin <= 0 {
			disponibilidad.DuracionMin = 60
		}
		return disponibilidad
	}

	// Formato viejo
	var vieja struct {
		Dias        []int          `json:"dias"`
		Rangos      []RangoHorario `json:"rangos"`
		DuracionMin int            `json:"duracion_min"`
	}
	if err := json.Unmarshal(rawJSON, &vieja); err != nil || len(vieja.Dias) == 0 || len(vieja.Rangos) == 0 {
		return disponibilidad
	}
	horarios := make(map[string][]RangoHorario)
	for _, d := range vieja.Dias {
		horarios[strconv.Itoa(d)] = vieja.Rangos
	}
	duracion := vieja.DuracionMin
	if duracion <= 0 {
		duracion = 60
	}
	return DisponibilidadVisitas{DuracionMin: duracion, Horarios: horarios}
}

// validarSlotDisponible verifica que una fecha/hora cumpla la agenda del
// vendedor (día atendido, dentro de una franja, sin salirse) y que no se
// solape con otra visita ya propuesta/confirmada para la misma propiedad.
// excluirVisitID permite ignorar la propia visita (al reprogramar).
func validarSlotDisponible(db *gorm.DB, propertyID, vendedorID uint, fecha time.Time, excluirVisitID uint) error {
	// 1) Agenda del vendedor: el día debe tener franjas y la visita debe caber
	var vendedor models.User
	disponibilidad := DisponibilidadPorDefecto()
	if err := db.First(&vendedor, vendedorID).Error; err == nil {
		disponibilidad = parseDisponibilidad(vendedor.HorarioVisitas)
	}

	duracion := time.Duration(disponibilidad.DuracionMin) * time.Minute
	if duracion <= 0 {
		duracion = 60 * time.Minute
	}

	franjas := disponibilidad.Horarios[strconv.Itoa(int(fecha.Weekday()))]
	inicio := fecha.Hour()*60 + fecha.Minute()
	fin := inicio + int(duracion.Minutes())

	dentroDeFranja := false
	for _, f := range franjas {
		var hDesde, mDesde, hHasta, mHasta int
		fmt.Sscanf(f.Desde, "%d:%d", &hDesde, &mDesde)
		fmt.Sscanf(f.Hasta, "%d:%d", &hHasta, &mHasta)
		if inicio >= hDesde*60+mDesde && fin <= hHasta*60+mHasta {
			dentroDeFranja = true
			break
		}
	}
	if !dentroDeFranja {
		return errors.New("el horario elegido no está dentro de la disponibilidad del vendedor")
	}

	// 2) Sin solapamiento con otras visitas activas de la misma propiedad
	var visitas []models.Visit
	db.Select("id", "fecha_visita").
		Where("property_id = ? AND id <> ? AND estado <> ? AND fecha_visita IS NOT NULL AND fecha_visita <> ?",
			propertyID, excluirVisitID, models.VisitStatusCancelada, time.Time{}).
		Find(&visitas)

	for _, v := range visitas {
		existingInicio := v.FechaVisita
		existingFin := existingInicio.Add(duracion)
		// Solape: [inicio, fin) interseca [existingInicio, existingFin)
		if fecha.Before(existingFin) && existingInicio.Before(fecha.Add(duracion)) {
			return errors.New("ese horario acaba de ser solicitado por otro interesado. Elige otro")
		}
	}
	return nil
}

// GetVisitAvailability - Devuelve la disponibilidad del vendedor de una propiedad
// y los horarios ya ocupados (visitas confirmadas/agendadas con fecha)
// GET /api/visits/availability/:propertyId
func (h *VisitHandler) GetVisitAvailability(c *fiber.Ctx) error {
	propertyID, err := strconv.Atoi(c.Params("propertyId"))
	if err != nil || propertyID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	var prop models.Property
	if err := h.DB.First(&prop, propertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// Cargar disponibilidad del vendedor (o default)
	var vendedor models.User
	disponibilidad := DisponibilidadPorDefecto()
	if err := h.DB.First(&vendedor, prop.UserID).Error; err == nil {
		disponibilidad = parseDisponibilidad(vendedor.HorarioVisitas)
	}

	// Normalizar: quitar días sin franjas
	for dia, franjas := range disponibilidad.Horarios {
		if len(franjas) == 0 {
			delete(disponibilidad.Horarios, dia)
		}
	}

	// Horarios ocupados: visitas con fecha asignada y no canceladas
	var visits []models.Visit
	h.DB.Select("fecha_visita").
		Where("property_id = ? AND estado <> ? AND fecha_visita IS NOT NULL AND fecha_visita <> ?",
			propertyID, models.VisitStatusCancelada, time.Time{}).
		Find(&visits)

	ocupadas := make([]string, 0, len(visits))
	for _, v := range visits {
		ocupadas = append(ocupadas, v.FechaVisita.Format("2006-01-02 15:04"))
	}

	return c.JSON(fiber.Map{
		"duracion_min": disponibilidad.DuracionMin,
		"horarios":     disponibilidad.Horarios,
		"ocupadas":     ocupadas,
	})
}

// UpdateVisitAvailability - El vendedor configura sus horarios de visita
// PUT /api/auth/visit-availability
func (h *VisitHandler) UpdateVisitAvailability(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var req DisponibilidadVisitas
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validaciones básicas
	if req.DuracionMin <= 0 {
		req.DuracionMin = 60
	}
	if len(req.Horarios) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Configura al menos un día con horarios",
		})
	}
	for dia, franjas := range req.Horarios {
		for _, r := range franjas {
			if r.Desde == "" || r.Hasta == "" || r.Desde >= r.Hasta {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Hay franjas horarias inválidas en " + dia + ": la hora de inicio debe ser menor a la de fin",
				})
			}
		}
	}

	// Quitar días sin franjas antes de guardar
	limpio := make(map[string][]RangoHorario)
	for dia, franjas := range req.Horarios {
		if len(franjas) > 0 {
			limpio[dia] = franjas
		}
	}
	req.Horarios = limpio

	bytes, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error procesando disponibilidad",
		})
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).
		Update("horario_visitas", datatypes.JSON(bytes)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error guardando disponibilidad",
		})
	}

	return c.JSON(fiber.Map{
		"message":        "Disponibilidad actualizada correctamente",
		"disponibilidad": req,
	})
}
