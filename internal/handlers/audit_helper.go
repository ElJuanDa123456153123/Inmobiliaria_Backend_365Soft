package handlers

import (
	"encoding/json"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// LogAudit - Registra una acción de auditoría
func (h *PropertyHandler) LogAudit(c *fiber.Ctx, action, entity string, entityID *uint, userID *uint, details interface{}) {
	auditLog := models.AuditLog{
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}

	// Asignar userID si existe
	if userID != nil {
		auditLog.UserID = userID
	}

	// Serializar detalles a JSON si existen
	if details != nil {
		detailsBytes, err := json.Marshal(details)
		if err == nil {
			auditLog.Details = string(detailsBytes)
		}
	}

	// Guardar en BD a través del servicio (sin error handling, es solo log)
	h.PropertyService.AuditRepo.CreateAsync(&auditLog)
}
