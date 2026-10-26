package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// HabitacionHandler - CRUD de habitaciones para alquiler por habitaciones
type HabitacionHandler struct {
	DB *gorm.DB
}

func NewHabitacionHandler(db *gorm.DB) *HabitacionHandler {
	return &HabitacionHandler{DB: db}
}

// HabitacionRequest - Body para crear/editar habitación
type HabitacionRequest struct {
	Nombre      string   `json:"nombre"`
	Descripcion string   `json:"descripcion"`
	Precio      *float64 `json:"precio"` // null = hereda precio de la propiedad
	Activa      *bool    `json:"activa"`
}

// contarActivos devuele el id del contrato activo de una habitación (0 = libre)
func contratoActivoDeHabitacion(db *gorm.DB, habitacionID uint) uint {
	var contrato models.RentalContract
	if err := db.Where("habitacion_id = ? AND estado = ? AND deleted_at IS NULL",
		habitacionID, models.ContractStatusActivo).First(&contrato).Error; err != nil {
		return 0
	}
	return contrato.ID
}

// marcarDisponibilidad anota en cada habitación si está libre u ocupada
func marcarDisponibilidad(db *gorm.DB, propiedadID uint, habitaciones []models.Habitacion) []models.Habitacion {
	var contratos []models.RentalContract
	db.Select("id", "habitacion_id").
		Where("property_id = ? AND estado = ? AND habitacion_id IS NOT NULL AND deleted_at IS NULL",
			propiedadID, models.ContractStatusActivo).
		Find(&contratos)

	ocupadas := make(map[uint]uint)
	for _, c := range contratos {
		if c.HabitacionID != nil {
			ocupadas[*c.HabitacionID] = c.ID
		}
	}
	for i := range habitaciones {
		if contratoID, ok := ocupadas[habitaciones[i].ID]; ok {
			habitaciones[i].ContratoActivoID = &contratoID
			habitaciones[i].Disponible = false
		} else {
			habitaciones[i].Disponible = habitaciones[i].Activa
		}
	}
	return habitaciones
}

// ListHabitaciones - Lista habitaciones de una propiedad (dueño, con ocupación)
// GET /api/panel/properties/:id/habitaciones
func (h *HabitacionHandler) ListHabitaciones(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de propiedad inválido"})
	}

	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", propertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Propiedad no encontrada o no tienes permisos"})
	}

	var habitaciones []models.Habitacion
	h.DB.Where("property_id = ?", propertyID).Order("id ASC").Find(&habitaciones)
	habitaciones = marcarDisponibilidad(h.DB, uint(propertyID), habitaciones)

	return c.JSON(fiber.Map{
		"habitaciones": habitaciones,
		"total":        len(habitaciones),
		"disponibles":  countDisponibles(habitaciones),
	})
}

func countDisponibles(habitaciones []models.Habitacion) int {
	n := 0
	for _, h := range habitaciones {
		if h.Disponible {
			n++
		}
	}
	return n
}

// CreateHabitacion - Crea una habitación
// POST /api/panel/properties/:id/habitaciones
func (h *HabitacionHandler) CreateHabitacion(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID de propiedad inválido"})
	}

	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", propertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Propiedad no encontrada o no tienes permisos"})
	}

	var req HabitacionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error parseando request"})
	}
	if req.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El nombre de la habitación es requerido"})
	}

	// Límite razonable
	var count int64
	h.DB.Model(&models.Habitacion{}).Where("property_id = ?", propertyID).Count(&count)
	if count >= 30 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Máximo 30 habitaciones por propiedad"})
	}

	activa := true
	if req.Activa != nil {
		activa = *req.Activa
	}

	hab := models.Habitacion{
		PropertyID:  uint(propertyID),
		Nombre:      req.Nombre,
		Descripcion: req.Descripcion,
		Precio:      req.Precio,
		Activa:      activa,
	}
	if err := h.DB.Create(&hab).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error creando habitación"})
	}

	hab.Disponible = activa
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Habitación creada", "habitacion": hab})
}

// UpdateHabitacion - Edita una habitación
// PUT /api/panel/properties/:id/habitaciones/:hid
func (h *HabitacionHandler) UpdateHabitacion(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID, err := c.ParamsInt("id")
	habitacionID, err2 := c.ParamsInt("hid")
	if err != nil || err2 != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "IDs inválidos"})
	}

	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", propertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Propiedad no encontrada o no tienes permisos"})
	}

	var hab models.Habitacion
	if err := h.DB.Where("id = ? AND property_id = ?", habitacionID, propertyID).First(&hab).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Habitación no encontrada"})
	}

	var req HabitacionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error parseando request"})
	}
	if req.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El nombre de la habitación es requerido"})
	}

	// No se puede desactivar una habitación ocupada
	if req.Activa != nil && !*req.Activa && hab.Activa {
		if contratoActivoDeHabitacion(h.DB, hab.ID) != 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "No se puede desactivar: la habitación tiene un contrato activo. Finaliza el contrato primero",
			})
		}
	}

	hab.Nombre = req.Nombre
	hab.Descripcion = req.Descripcion
	hab.Precio = req.Precio
	if req.Activa != nil {
		hab.Activa = *req.Activa
	}
	if err := h.DB.Save(&hab).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error guardando habitación"})
	}

	habitaciones := marcarDisponibilidad(h.DB, uint(propertyID), []models.Habitacion{hab})
	return c.JSON(fiber.Map{"message": "Habitación actualizada", "habitacion": habitaciones[0]})
}

// DeleteHabitacion - Elimina una habitación (solo si no tiene contrato activo)
// DELETE /api/panel/properties/:id/habitaciones/:hid
func (h *HabitacionHandler) DeleteHabitacion(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	propertyID, err := c.ParamsInt("id")
	habitacionID, err2 := c.ParamsInt("hid")
	if err != nil || err2 != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "IDs inválidos"})
	}

	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", propertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Propiedad no encontrada o no tienes permisos"})
	}

	var hab models.Habitacion
	if err := h.DB.Where("id = ? AND property_id = ?", habitacionID, propertyID).First(&hab).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Habitación no encontrada"})
	}

	if contratoActivoDeHabitacion(h.DB, hab.ID) != 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "No se puede eliminar: la habitación tiene un contrato activo. Finaliza el contrato primero",
		})
	}

	if err := h.DB.Delete(&hab).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error eliminando habitación"})
	}

	return c.JSON(fiber.Map{"message": "Habitación eliminada"})
}
