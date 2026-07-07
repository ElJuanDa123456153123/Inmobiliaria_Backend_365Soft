package handlers

import (
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// LeadScoringHandler - Handler para Lead Scoring
type LeadScoringHandler struct {
	LeadScoringService *services.LeadScoringService
}

// NewLeadScoringHandler - Crea un nuevo handler de Lead Scoring
func NewLeadScoringHandler() (*LeadScoringHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	clientRepo := repository.NewClientRepository(db.DB)
	interactionRepo := repository.NewInteractionRepository(db.DB)
	viewHistoryRepo := repository.NewViewHistoryRepository(db.DB)
	favoriteRepo := repository.NewFavoriteRepository(db.DB)

	// Inicializar servicio
	leadScoringService := services.NewLeadScoringService(
		db.DB,
		clientRepo,
		interactionRepo,
		viewHistoryRepo,
		favoriteRepo,
	)

	return &LeadScoringHandler{
		LeadScoringService: leadScoringService,
	}, nil
}

// GetLeadScore - Obtiene el score de un lead
// GET /api/crm/clients/:id/score
func (h *LeadScoringHandler) GetLeadScore(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Obtener score
	score, err := h.LeadScoringService.GetLeadScore(uint(clientID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(score)
}

// CalculateLeadScore - Calcula o recalcula el score de un lead
// POST /api/crm/clients/:id/score/recalculate
func (h *LeadScoringHandler) CalculateLeadScore(c *fiber.Ctx) error {
	id := c.Params("id")
	clientID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Recalcular score
	score, err := h.LeadScoringService.UpdateLeadScore(uint(clientID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(score)
}

// GetAllLeadScores - Obtiene todos los scores
// GET /api/crm/leads/scores
func (h *LeadScoringHandler) GetAllLeadScores(c *fiber.Ctx) error {
	// Obtener parámetro recalculate (default: false)
	recalculate := c.QueryBool("recalculate", false)

	// Obtener todos los scores
	scores, err := h.LeadScoringService.GetAllLeadScores(recalculate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"scores": scores,
		"total":  len(scores),
	})
}

// GetTopLeads - Obtiene los mejores leads ordenados por score
// GET /api/crm/leads/top
func (h *LeadScoringHandler) GetTopLeads(c *fiber.Ctx) error {
	// Obtener advisor_id del JWT (opcional, para filtrar por asesor)
	var advisorID uint = 0

	userID := c.Locals("user_id")
	if userID != nil {
		if userIDUint, ok := userID.(uint); ok {
			advisorID = userIDUint
		} else if userIDFloat, ok := userID.(float64); ok {
			advisorID = uint(userIDFloat)
		}
	}

	// Obtener parámetro limit (default: 20)
	limit := c.QueryInt("limit", 20)

	// Obtener top leads
	scores, err := h.LeadScoringService.GetTopLeads(advisorID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"leads": scores,
		"total": len(scores),
	})
}

// GetLeadsByLevel - Obtiene leads filtrados por nivel
// GET /api/crm/leads/by-level/:level
func (h *LeadScoringHandler) GetLeadsByLevel(c *fiber.Ctx) error {
	levelStr := c.Params("level")

	// Validar nivel
	var level models.ScoreLevel
	switch levelStr {
	case "ALTO", "MEDIO", "BAJO":
		level = models.ScoreLevel(levelStr)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nivel inválido. Debe ser: ALTO, MEDIO o BAJO",
		})
	}

	// Obtener advisor_id del JWT (opcional, para filtrar por asesor)
	var advisorID uint = 0

	userID := c.Locals("user_id")
	if userID != nil {
		if userIDUint, ok := userID.(uint); ok {
			advisorID = userIDUint
		} else if userIDFloat, ok := userID.(float64); ok {
			advisorID = uint(userIDFloat)
		}
	}

	// Obtener leads por nivel
	scores, err := h.LeadScoringService.GetLeadsByLevel(level, advisorID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"leads": scores,
		"total": len(scores),
		"level": level,
	})
}

// RecalculateAllScores - Recalcula todos los scores de leads
// POST /api/crm/leads/scores/recalculate-all
func (h *LeadScoringHandler) RecalculateAllScores(c *fiber.Ctx) error {
	// Recalcular todos los scores
	if err := h.LeadScoringService.RecalculateAllScores(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Todos los lead scores han sido recalculados exitosamente",
	})
}

// GetScoreStats - Obtiene estadísticas de scoring
// GET /api/crm/leads/scores/stats
func (h *LeadScoringHandler) GetScoreStats(c *fiber.Ctx) error {
	// Obtener advisor_id del JWT (opcional, para filtrar por asesor)
	var advisorID uint = 0

	userID := c.Locals("user_id")
	if userID != nil {
		if userIDUint, ok := userID.(uint); ok {
			advisorID = userIDUint
		} else if userIDFloat, ok := userID.(float64); ok {
			advisorID = uint(userIDFloat)
		}
	}

	// Obtener estadísticas
	stats, err := h.LeadScoringService.GetScoreStats(advisorID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
