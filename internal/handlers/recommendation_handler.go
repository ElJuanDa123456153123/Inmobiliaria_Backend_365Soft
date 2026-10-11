package handlers

import (
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// RecommendationHandler - Handler para Motor de Recomendaciones
type RecommendationHandler struct {
	RecommendationService *services.RecommendationService
	Redis                 *config.RedisClient
}

// NewRecommendationHandler - Crea un nuevo handler de recomendaciones
func NewRecommendationHandler() (*RecommendationHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	favoriteRepo := repository.NewFavoriteRepository(db.DB)
	viewHistoryRepo := repository.NewViewHistoryRepository(db.DB)
	savedSearchRepo := repository.NewSavedSearchRepository(db.DB)

	// Inicializar servicio
	recommendationService := services.NewRecommendationService(
		db.DB,
		favoriteRepo,
		viewHistoryRepo,
		savedSearchRepo,
	)

	// Redis opcional (caché de recomendaciones precalculadas por el job semanal)
	redisClient, err := config.NewRedis()
	if err != nil {
		redisClient = nil
	}

	return &RecommendationHandler{
		RecommendationService: recommendationService,
		Redis:                 redisClient,
	}, nil
}

// GetRecommendations - Obtiene recomendaciones personalizadas para el usuario
// GET /api/recommendations
// Prioriza las recomendaciones precalculadas por el job semanal (caché Redis, 7 días);
// si no hay caché, calcula en línea (contenido + colaborativo).
func (h *RecommendationHandler) GetRecommendations(c *fiber.Ctx) error {
	// Obtener user_id del JWT
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
	userIDUint, ok := userID.(uint)
	if !ok {
		if userIDFloat, ok := userID.(float64); ok {
			userIDUint = uint(userIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo user_id",
			})
		}
	}

	// Obtener parámetro limit (default: 10)
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// 1. Intentar caché precalculada
	if h.Redis != nil {
		var cached []services.PropertyRecommendation
		if err := h.Redis.GetCachedRecommendations(userIDUint, &cached); err == nil && len(cached) > 0 {
			if limit < len(cached) {
				cached = cached[:limit]
			}
			return c.JSON(fiber.Map{
				"recommendations": cached,
				"total":           len(cached),
				"user_id":         userIDUint,
				"cached":          true,
			})
		}
	}

	// 2. Cálculo en línea
	recommendations, err := h.RecommendationService.GetRecommendations(userIDUint, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// 3. Almacenar en caché para las próximas consultas
	if h.Redis != nil && len(recommendations) > 0 {
		h.Redis.CacheRecommendations(userIDUint, recommendations)
	}

	return c.JSON(fiber.Map{
		"recommendations": recommendations,
		"total":          len(recommendations),
		"user_id":        userIDUint,
	})
}

// GetRecommendationsById - Obtiene recomendaciones para un usuario específico (admin/asesor)
// GET /api/recommendations/user/:userId
func (h *RecommendationHandler) GetRecommendationsById(c *fiber.Ctx) error {
	// Obtener userId de la URL
	userIdStr := c.Params("userId")
	userID, err := strconv.ParseUint(userIdStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Obtener parámetro limit (default: 10)
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// Obtener recomendaciones
	recommendations, err := h.RecommendationService.GetContentBasedRecommendations(uint(userID), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"recommendations": recommendations,
		"total":          len(recommendations),
		"user_id":        uint(userID),
	})
}
