package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// AuditService - Interface para servicio de auditoría
type AuditService interface {
	LogAction(ctx interface{}, entry *AuditLogEntry) error
}

// AuditMiddleware - Middleware para logging de auditoría
type AuditMiddleware struct {
	auditService AuditService
}

// AuditLogEntry - Entrada de log
type AuditLogEntry struct {
	UserID    *uint
	IPAddress string
	UserAgent  string
	Accion    models.AuditAction
	Entidad   models.AuditEntity
	EntityID  *uint
	Before    interface{}
	After     interface{}
	Metadata  map[string]interface{}
	RequestID string
}

// NewAuditMiddleware - Crea una nueva instancia del middleware
func NewAuditMiddleware(auditService AuditService) *AuditMiddleware {
	return &AuditMiddleware{
		auditService: auditService,
	}
}

// Audit - Middleware que loggea acciones automáticamente
func (m *AuditMiddleware) Audit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Capturar request body para auditoría
		var body []byte
		if c.Method() != "GET" && c.Method() != "DELETE" {
			body = c.Body()
			// Nota: En Fiber v2, el body ya está disponible, no necesitamos restaurarlo
		}

		// 2. Continuar con el handler
		err := c.Next()

		// 3. Loggear acción (después de procesar)
		m.logAction(c, c.Response().StatusCode(), body)

		return err
	}
}

// logAction - Loggea la acción en el request actual
func (m *AuditMiddleware) logAction(c *fiber.Ctx, statusCode int, body []byte) {
	// Obtener información del request
	method := c.Method()
	path := c.Path()
	userID := c.Locals("user_id")
	requestID := c.Locals("request_id")

	// Determinar acción y entidad basado en método y path
	accion := m.getAction(method, path, statusCode)
	entidad, entityID := m.getEntity(path)

	if accion == "" {
		return // No loggear esta acción
	}

	// Crear entrada de log
	entry := &AuditLogEntry{
		UserID:    userID.(*uint),
		IPAddress: c.IP(),
		UserAgent:  c.Get("User-Agent"),
		Accion:    accion,
		Entidad:   entidad,
		EntityID:  entityID,
		Metadata: map[string]interface{}{
			"method":      method,
			"path":        path,
			"status_code": statusCode,
			"request_id":  requestID,
		},
		RequestID: requestID.(string),
	}

	// Añadir body si existe
	if len(body) > 0 && len(body) < 10000 { // Limitar a 10KB
		entry.Metadata["body"] = string(body)
	}

	// Loggear de forma asíncrona para no bloquear
	go func() {
		_ = m.auditService.LogAction(context.Background(), entry)
	}()
}

// getAction - Determina la acción de auditoría
func (m *AuditMiddleware) getAction(method, path string, statusCode int) models.AuditAction {
	// Solo loggear requests exitosos
	if statusCode >= 400 {
		return ""
	}

	switch method {
	case "POST":
		return models.ActionCreate
	case "PUT", "PATCH":
		return models.ActionUpdate
	case "DELETE":
		return models.ActionDelete
	case "GET":
		// Solo loggear GETs de ciertos recursos
		if contains(path, "/admin") || contains(path, "/dashboard") {
			return models.ActionView
		}
	}

	return ""
}

// getEntity - Determina la entidad basado en el path
func (m *AuditMiddleware) getEntity(path string) (models.AuditEntity, *uint) {
	// Extraer entity ID del path
	parts := splitPath(path)

	if len(parts) == 0 {
		return "", nil
	}

	// Determinar entidad basado en el primer segmento
	switch parts[0] {
	case "propiedades", "api":
		if len(parts) > 1 {
			// Intentar parsear ID
			if id := parseID(parts[1]); id != nil {
				return models.EntityProperty, id
			}
		}
		return models.EntityProperty, nil

	case "categorias":
		if len(parts) > 1 {
			if id := parseID(parts[1]); id != nil {
				return models.EntityCategory, id
			}
		}
		return models.EntityCategory, nil

	case "usuarios":
		if len(parts) > 1 {
			if id := parseID(parts[1]); id != nil {
				return models.EntityUser, id
			}
		}
		return models.EntityUser, nil

	case "admin":
		return models.EntityAdmin, nil

	default:
		return "", nil
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s[len(s)-1:] == substr[len(substr)-1:]
}

func splitPath(path string) []string {
	// Eliminar query params
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Split por /
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

func parseID(s string) *uint {
	var id uint
	if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
		return &id
	}
	return nil
}
