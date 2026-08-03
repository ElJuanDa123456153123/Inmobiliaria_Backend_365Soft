package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PropertyContactHandler - Handler para contactos de propiedades
type PropertyContactHandler struct {
	DB *gorm.DB
}

// NewPropertyContactHandler - Crea un nuevo handler de contactos
func NewPropertyContactHandler() (*PropertyContactHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &PropertyContactHandler{
		DB: dbConfig.DB,
	}, nil
}

// CreateContactRequest - Estructura para crear contacto
type CreateContactRequest struct {
	PropertyID uint   `json:"property_id" validate:"required"`
	Mensaje    string `json:"mensaje" validate:"required"`
}

// CreateContact - Crea un nuevo contacto de interés
// POST /api/properties/:id/contacts
func (h *PropertyContactHandler) CreateContact(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener property_id del parámetro
	propertyID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de propiedad inválido",
		})
	}

	// Parsear body
	var req CreateContactRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Usar el ID de la propiedad del parámetro
	req.PropertyID = uint(propertyID)

	// Verificar si la propiedad existe
	var prop models.Property
	if err := h.DB.First(&prop, req.PropertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// No permitir contactarse con propia propiedad
	if prop.UserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes contactarte con tu propia propiedad",
		})
	}

	// Verificar si ya existe un contacto previo (para no duplicar)
	var existingContact models.PropertyContact
	if err := h.DB.Where("user_id = ? AND property_id = ?", userID, req.PropertyID).First(&existingContact).Error; err == nil {
		// Ya existe un contacto, retornarlo
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":  "Contacto ya existente",
			"contacto": existingContact,
		})
	}

	// Crear contacto
	contact := models.PropertyContact{
		PropertyID: req.PropertyID,
		UserID:     userID,
		VendedorID: prop.UserID,
		Mensaje:    req.Mensaje,
		Leido:      false,
		Respondido: false,
	}

	if err := h.DB.Create(&contact).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando contacto",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("User").Preload("Vendedor").First(&contact, contact.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Contacto creado exitosamente",
		"contacto": contact,
	})
}

// GetPropertyContacts - Obtiene contactos de una propiedad
// GET /api/properties/:id/contacts
func (h *PropertyContactHandler) GetPropertyContacts(c *fiber.Ctx) error {
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

	// Obtener contactos
	var contacts []models.PropertyContact
	if err := h.DB.Where("property_id = ?", propertyID).
		Preload("User").
		Order("created_at DESC").
		Find(&contacts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo contactos",
		})
	}

	return c.JSON(fiber.Map{
		"contactos": contacts,
		"total":     len(contacts),
	})
}

// GetMyContacts - Obtiene todos los contactos del vendedor (usuario autenticado)
// GET /api/contacts/my
func (h *PropertyContactHandler) GetMyContacts(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de filtrado
	leido := c.Query("leido", "")
	respondido := c.Query("respondido", "")
	propertyID := c.Query("property_id", "")

	// Construir query
	query := h.DB.Where("vendedor_id = ?", userID)

	// Aplicar filtros
	if leido != "" {
		if leido == "true" {
			query = query.Where("leido = ?", true)
		} else if leido == "false" {
			query = query.Where("leido = ?", false)
		}
	}

	if respondido != "" {
		if respondido == "true" {
			query = query.Where("respondido = ?", true)
		} else if respondido == "false" {
			query = query.Where("respondido = ?", false)
		}
	}

	if propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	// Obtener contactos
	var contacts []models.PropertyContact
	if err := query.Preload("Property").Preload("User").
		Order("created_at DESC").
		Find(&contacts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo contactos",
		})
	}

	// Contar no leídos
	var unreadCount int64
	h.DB.Model(&models.PropertyContact{}).
		Where("vendedor_id = ? AND leido = ?", userID, false).
		Count(&unreadCount)

	return c.JSON(fiber.Map{
		"contactos":    contacts,
		"total":        len(contacts),
		"no_leidos":    unreadCount,
	})
}

// MarkAsRead - Marca un contacto como leído
// PATCH /api/contacts/:id/read
func (h *PropertyContactHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contactID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contacto inválido",
		})
	}

	// Verificar que el contacto pertenezca a una propiedad del vendedor
	var contact models.PropertyContact
	if err := h.DB.Where("id = ? AND vendedor_id = ?", contactID, userID).First(&contact).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contacto no encontrado",
		})
	}

	// Actualizar
	contact.Leido = true
	if err := h.DB.Save(&contact).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando contacto",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Contacto marcado como leído",
		"contacto": contact,
	})
}

// MarkAsResponded - Marca un contacto como respondido
// PATCH /api/contacts/:id/respond
func (h *PropertyContactHandler) MarkAsResponded(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contactID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contacto inválido",
		})
	}

	// Verificar que el contacto pertenezca a una propiedad del vendedor
	var contact models.PropertyContact
	if err := h.DB.Where("id = ? AND vendedor_id = ?", contactID, userID).First(&contact).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contacto no encontrado",
		})
	}

	// Actualizar
	contact.Respondido = true
	contact.Leido = true // Si respondió, está leído
	if err := h.DB.Save(&contact).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando contacto",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Contacto marcado como respondido",
		"contacto": contact,
	})
}
