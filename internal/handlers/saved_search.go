package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SavedSearchHandler - Handler para búsquedas guardadas y notificaciones
type SavedSearchHandler struct {
	DB *gorm.DB
}

// NewSavedSearchHandler - Crea un nuevo handler de búsquedas guardadas
func NewSavedSearchHandler() (*SavedSearchHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &SavedSearchHandler{
		DB: dbConfig.DB,
	}, nil
}

// CreateSavedSearch - Guarda una nueva búsqueda de filtros
func (h *SavedSearchHandler) CreateSavedSearch(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type Request struct {
		Nombre  string                 `json:"nombre"`
		Filtros map[string]interface{} `json:"filtros"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nombre es requerido"})
	}
	if len(req.Filtros) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Filtros son requeridos"})
	}

	filtrosJSON, err := json.Marshal(req.Filtros)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error marshalling filtros"})
	}

	savedSearch := models.SavedSearch{
		UserID:   userID,
		Nombre:   req.Nombre,
		Filtros:  filtrosJSON,
		IsActive: true,
	}

	if err := h.DB.Create(&savedSearch).Error; err != nil {
		fmt.Printf("Error creando búsqueda guardada: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error guardando la búsqueda"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":      "Búsqueda guardada exitosamente",
		"saved_search": savedSearch,
	})
}

// GetSavedSearches - Obtiene todas las búsquedas guardadas del usuario
func (h *SavedSearchHandler) GetSavedSearches(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var searches []models.SavedSearch
	if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&searches).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo búsquedas guardadas"})
	}

	// Para cada búsqueda guardada, también calculamos dinámicamente la cantidad actual de coincidencias
	type SearchWithCount struct {
		models.SavedSearch
		MatchesCount int64 `json:"matches_count"`
	}

	var response []SearchWithCount = make([]SearchWithCount, 0)

	for _, search := range searches {
		var filters map[string]interface{}
		if err := json.Unmarshal(search.Filtros, &filters); err == nil {
			var count int64
			query := h.BuildMatchQuery(filters)
			query.Count(&count)
			response = append(response, SearchWithCount{
				SavedSearch:  search,
				MatchesCount: count,
			})
		} else {
			response = append(response, SearchWithCount{
				SavedSearch:  search,
				MatchesCount: 0,
			})
		}
	}

	return c.JSON(fiber.Map{"searches": response})
}

// DeleteSavedSearch - Elimina una búsqueda guardada
func (h *SavedSearchHandler) DeleteSavedSearch(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	searchID := c.Params("id")

	result := h.DB.Where("id = ? AND user_id = ?", searchID, userID).Delete(&models.SavedSearch{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error eliminando búsqueda guardada"})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Búsqueda no encontrada"})
	}

	// También eliminar las notificaciones asociadas a esta búsqueda guardada
	h.DB.Where("saved_search_id = ?", searchID).Delete(&models.Notification{})

	return c.JSON(fiber.Map{"message": "Búsqueda guardada eliminada"})
}

// ToggleSavedSearch - Activa/desactiva las alertas de una búsqueda
func (h *SavedSearchHandler) ToggleSavedSearch(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	searchID := c.Params("id")

	var search models.SavedSearch
	if err := h.DB.Where("id = ? AND user_id = ?", searchID, userID).First(&search).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Búsqueda no encontrada"})
	}

	search.IsActive = !search.IsActive
	if err := h.DB.Save(&search).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error actualizando estado de alertas"})
	}

	return c.JSON(fiber.Map{
		"message":      "Estado de alertas actualizado",
		"saved_search": search,
	})
}

// GetSavedSearchMatches - Obtiene las propiedades que actualmente coinciden con la búsqueda guardada
func (h *SavedSearchHandler) GetSavedSearchMatches(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	searchID := c.Params("id")

	var search models.SavedSearch
	if err := h.DB.Where("id = ? AND user_id = ?", searchID, userID).First(&search).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Búsqueda no encontrada"})
	}

	var filters map[string]interface{}
	if err := json.Unmarshal(search.Filtros, &filters); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Filtros inválidos"})
	}

	var properties []models.Property
	query := h.BuildMatchQuery(filters)
	err := query.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("PropertyAdvisors.Advisor").
		Order("created_at DESC").
		Limit(50). // Limitar a las 50 coincidencias más recientes
		Find(&properties).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo propiedades coincidentes"})
	}

	return c.JSON(fiber.Map{
		"search":     search,
		"properties": properties,
	})
}

// GetNotifications - Obtiene notificaciones del usuario
func (h *SavedSearchHandler) GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var notifications []models.Notification
	err := h.DB.
		Preload("Property.PropertyImages").
		Preload("SavedSearch").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20). // Traer las últimas 20 notificaciones
		Find(&notifications).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error obteniendo notificaciones"})
	}

	// Obtener cantidad de no leídas
	var unreadCount int64
	h.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&unreadCount)

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

// MarkNotificationsAsRead - Marca todas las notificaciones del usuario como leídas
func (h *SavedSearchHandler) MarkNotificationsAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	err := h.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error marcando notificaciones como leídas"})
	}

	return c.JSON(fiber.Map{"message": "Notificaciones marcadas como leídas"})
}

// BuildMatchQuery - Helper para construir la consulta de propiedades basadas en filtros guardados
func (h *SavedSearchHandler) BuildMatchQuery(filters map[string]interface{}) *gorm.DB {
	query := h.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "activo")

	// Operación
	operation := getStringField(filters, "tipo_operacion", "operation")
	if operation != "" && operation != "todas" && operation != "todos" {
		query = query.Where("tipo_operacion = ?", operation)
	}

	// Tipo de propiedad
	propType := getStringField(filters, "tipo_propiedad", "type")
	if propType != "" && propType != "todas" && propType != "todos" {
		query = query.Where("tipo_propiedad = ?", propType)
	}

	// Zona / Dirección / Título
	zona := getStringField(filters, "zona", "search")
	if zona != "" {
		searchTerm := "%" + zona + "%"
		query = query.Where("(zona ILIKE ? OR direccion ILIKE ? OR titulo ILIKE ? OR descripcion ILIKE ?)", searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// Moneda
	moneda := getStringField(filters, "moneda", "")
	if moneda != "" && moneda != "todas" && moneda != "todos" {
		query = query.Where("moneda = ?", moneda)
	}

	// Precios
	if priceMin, err := getFloatField(filters, "precio_min", "priceMin"); err == nil && priceMin > 0 {
		query = query.Where("precio >= ?", priceMin)
	}
	if priceMax, err := getFloatField(filters, "precio_max", "priceMax"); err == nil && priceMax > 0 {
		query = query.Where("precio <= ?", priceMax)
	}

	// Dormitorios (habitaciones)
	dormitorios := getStringField(filters, "dormitorios", "bedrooms")
	if dormitorios != "" && dormitorios != "todas" && dormitorios != "todos" {
		query = query.Where("dormitorios = ?", dormitorios)
	}

	// Baños
	banos := getStringField(filters, "banos", "bathrooms")
	if banos != "" && banos != "todas" && banos != "todos" {
		query = query.Where("banos = ?", banos)
	}

	// Garaje
	garaje := getStringField(filters, "garaje", "")
	if garaje != "" && garaje != "todas" && garaje != "todos" {
		query = query.Where("garaje = ?", garaje)
	}

	return query
}

// Helpers locales para parseo seguro de datos de JSON maps
func getStringField(filters map[string]interface{}, key1, key2 string) string {
	if v, ok := filters[key1]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if key2 != "" {
		if v, ok := filters[key2]; ok && v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getFloatField(filters map[string]interface{}, key1, key2 string) (float64, error) {
	var val interface{}
	var ok bool
	if val, ok = filters[key1]; !ok || val == nil {
		if key2 != "" {
			if val, ok = filters[key2]; !ok || val == nil {
				return 0, fmt.Errorf("key not found")
			}
		} else {
			return 0, fmt.Errorf("key not found")
		}
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if v == "" {
			return 0, nil
		}
		return strconv.ParseFloat(v, 64)
	}
	return 0, fmt.Errorf("unknown type")
}
