package handlers

import (
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// CrmHandler - Handler para el CRM
type CrmHandler struct {
	CrmService *services.CrmService
}

// NewCrmHandler - Crea un nuevo handler de CRM
func NewCrmHandler() (*CrmHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	clientRepo := repository.NewClientRepository(db.DB)
	interactionRepo := repository.NewInteractionRepository(db.DB)

	// Inicializar servicio
	crmService := services.NewCrmService(clientRepo, interactionRepo, db.DB)

	return &CrmHandler{
		CrmService: crmService,
	}, nil
}

// ==================== CLIENTES ====================

// CreateClient - Crea un nuevo cliente
func (h *CrmHandler) CreateClient(c *fiber.Ctx) error {
	// Parsear request
	var req services.CreateClientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Obtener advisor_id del JWT (del middleware de autenticación)
	advisorID := c.Locals("user_id")
	if advisorID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir advisor_id a uint
	advisorIDUint, ok := advisorID.(uint)
	if !ok {
		// Intentar convertir desde float64 (JSON numbers)
		if advisorIDFloat, ok := advisorID.(float64); ok {
			advisorIDUint = uint(advisorIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo advisor_id",
			})
		}
	}

	req.AdvisorID = advisorIDUint

	// Crear cliente
	client, err := h.CrmService.CreateClient(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(client)
}

// GetClient - Obtiene un cliente por ID
func (h *CrmHandler) GetClient(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	client, err := h.CrmService.GetClientByID(uint(clientID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(client)
}

// UpdateClient - Actualiza un cliente
func (h *CrmHandler) UpdateClient(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Parsear request
	var req services.UpdateClientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	req.ClientID = uint(clientID)

	// Actualizar cliente
	client, err := h.CrmService.UpdateClient(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(client)
}

// DeleteClient - Elimina un cliente
func (h *CrmHandler) DeleteClient(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	if err := h.CrmService.DeleteClient(uint(clientID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cliente eliminado exitosamente",
	})
}

// ListClients - Lista clientes con filtros
func (h *CrmHandler) ListClients(c *fiber.Ctx) error {
	// Obtener advisor_id del JWT
	advisorID := c.Locals("user_id")
	if advisorID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir advisor_id a uint
	advisorIDUint, ok := advisorID.(uint)
	if !ok {
		if advisorIDFloat, ok := advisorID.(float64); ok {
			advisorIDUint = uint(advisorIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo advisor_id",
			})
		}
	}

	// Obtener parámetros de query
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	status := c.Query("status")
	query := c.Query("query")

	// Crear request
	req := &services.ListClientsRequest{
		AdvisorID: advisorIDUint,
		Page:      page,
		Limit:     limit,
		Query:     query,
	}

	// Parsear status si se proporciona
	if status != "" {
		var clientStatus models.ClientStatus = models.ClientStatus(status)
		req.Status = &clientStatus
	}

	// Listar clientes
	clients, total, err := h.CrmService.ListClients(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"clients": clients,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetFunnelStats - Obtiene las estadísticas del embudo de ventas
func (h *CrmHandler) GetFunnelStats(c *fiber.Ctx) error {
	// Obtener advisor_id del JWT
	advisorID := c.Locals("user_id")
	if advisorID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir advisor_id a uint
	advisorIDUint, ok := advisorID.(uint)
	if !ok {
		if advisorIDFloat, ok := advisorID.(float64); ok {
			advisorIDUint = uint(advisorIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo advisor_id",
			})
		}
	}

	// Obtener estadísticas
	stats, err := h.CrmService.GetFunnelStats(advisorIDUint)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetUpcomingContacts - Obtiene próximos contactos pendientes
func (h *CrmHandler) GetUpcomingContacts(c *fiber.Ctx) error {
	// Obtener advisor_id del JWT
	advisorID := c.Locals("user_id")
	if advisorID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir advisor_id a uint
	advisorIDUint, ok := advisorID.(uint)
	if !ok {
		if advisorIDFloat, ok := advisorID.(float64); ok {
			advisorIDUint = uint(advisorIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo advisor_id",
			})
		}
	}

	// Obtener días (default: 7)
	days := c.QueryInt("days", 7)

	// Obtener próximos contactos
	clients, err := h.CrmService.GetUpcomingContacts(advisorIDUint, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(clients)
}

// ==================== INTERACCIONES ====================

// CreateInteraction - Crea una nueva interacción
func (h *CrmHandler) CreateInteraction(c *fiber.Ctx) error {
	// Parsear request
	var req services.CreateInteractionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Obtener advisor_id del JWT
	advisorID := c.Locals("user_id")
	if advisorID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir advisor_id a uint
	advisorIDUint, ok := advisorID.(uint)
	if !ok {
		if advisorIDFloat, ok := advisorID.(float64); ok {
			advisorIDUint = uint(advisorIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo advisor_id",
			})
		}
	}

	req.AdvisorID = advisorIDUint

	// Crear interacción
	interaction, err := h.CrmService.CreateInteraction(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(interaction)
}

// GetClientInteractions - Obtiene las interacciones de un cliente
func (h *CrmHandler) GetClientInteractions(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	interactions, err := h.CrmService.GetInteractionsByClient(uint(clientID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(interactions)
}
