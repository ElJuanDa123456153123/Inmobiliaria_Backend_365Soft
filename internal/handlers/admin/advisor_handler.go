package admin

import (
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdvisorHandler - Handler para gestión de asesores
type AdvisorHandler struct {
	DB *config.Database
}

// NewAdvisorHandler - Crea un nuevo handler de asesores
func NewAdvisorHandler() (*AdvisorHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &AdvisorHandler{
		DB: dbConfig,
	}, nil
}

// GetAdvisors - Obtiene todos los asesores con paginación y filtros
func (h *AdvisorHandler) GetAdvisors(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		fmt.Printf("❌ ERROR creando conexión: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Parsear query params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	activo := c.Query("activo", "")
	search := c.Query("search", "")

	offset := (page - 1) * limit

	// Construir query - Preload User para obtener avatar_url
	query := db.DB.Model(&models.Advisor{}).Where("deleted_at IS NULL").Preload("User")

	// Filtro por activo
	if activo != "" {
		if activo == "true" {
			query = query.Where("activo = ?", true)
		} else if activo == "false" {
			query = query.Where("activo = ?", false)
		}
	}

	// Búsqueda
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("nombre ILIKE ? OR email ILIKE ?", searchTerm, searchTerm)
	}

	// Contar total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fmt.Printf("❌ ERROR contando asesores: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error contando asesores",
		})
	}

	// Obtener asesores con paginación
	var advisors []models.Advisor
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&advisors).Error; err != nil {
		fmt.Printf("❌ ERROR obteniendo asesores: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesores",
		})
	}

	// Si el asesor no tiene avatar_url, usar el del usuario asociado
	for i := range advisors {
		if advisors[i].AvatarURL == "" && advisors[i].User != nil && advisors[i].User.AvatarURL != "" {
			advisors[i].AvatarURL = advisors[i].User.AvatarURL
		}
	}

	return c.JSON(fiber.Map{
		"advisors": advisors,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetAdvisor - Obtiene un asesor por ID
func (h *AdvisorHandler) GetAdvisor(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	id := c.Params("id")

	var advisor models.Advisor
	// Incluir propiedades asignadas
	if err := db.DB.Preload("Propiedades").First(&advisor, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Asesor no encontrado",
			})
		}
		fmt.Printf("❌ ERROR obteniendo asesor: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesor",
		})
	}

	return c.JSON(advisor)
}

// CreateAdvisorRequest - Estructura para crear asesor
type CreateAdvisorRequest struct {
	Nombre       string `json:"nombre"`
	Email        string `json:"email"`
	Password     string `json:"password"` // Contraseña para el login
	Telefono     string `json:"telefono"`
	Especialidad string `json:"especialidad"`
	ImagenURL    string `json:"imagen_url,omitempty"` // URL de la foto de perfil
	Activo       *bool  `json:"activo"`
}

// CreateAdvisor - Crea un nuevo asesor
func (h *AdvisorHandler) CreateAdvisor(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	var req CreateAdvisorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validaciones básicas
	if req.Nombre == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre, email y password son requeridos",
		})
	}

	// Validar longitud de contraseña
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La contraseña debe tener al menos 6 caracteres",
		})
	}

	// Verificar si el email ya existe en asesores (solo activos, no eliminados)
	var existingAdvisor models.Advisor
	if err := db.DB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&existingAdvisor).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El email ya está registrado como asesor",
		})
	}

	// Verificar si el email ya existe en usuarios (solo activos, no eliminados)
	var existingUser models.User
	if err := db.DB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El email ya está registrado como usuario",
		})
	}

	// Valor por defecto para Activo si no se proporciona
	activo := true
	if req.Activo != nil {
		activo = *req.Activo
	}

	// Hashear password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error procesando la contraseña",
		})
	}

	// Crear usuario con rol "agente"
	user := models.User{
		Nombre:     req.Nombre,
		Email:      req.Email,
		Password:   string(hashedPassword),
		Rol:        models.RoleAgente,
		Activo:     activo,
		Verificado: true, // Auto-verificar al crear desde admin
	}

	if err := db.DB.Create(&user).Error; err != nil {
		fmt.Printf("❌ ERROR creando usuario: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el usuario",
		})
	}

	// Crear asesor asociado al usuario
	advisor := models.Advisor{
		Nombre:       req.Nombre,
		Email:        req.Email,
		Telefono:     req.Telefono,
		Especialidad: req.Especialidad,
		ImagenURL:    req.ImagenURL,
		Activo:       activo,
		UserID:       &user.ID,
	}

	if err := db.DB.Create(&advisor).Error; err != nil {
		fmt.Printf("❌ ERROR creando asesor: %v\n", err)
		// Rollback: eliminar el usuario creado
		db.DB.Delete(&user)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el asesor",
		})
	}

	// Cargar el usuario en la respuesta
	advisor.User = &user

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Asesor creado exitosamente",
		"advisor": advisor,
	})
}

// UpdateAdvisorRequest - Estructura para actualizar asesor
type UpdateAdvisorRequest struct {
	Nombre       string `json:"nombre"`
	Email        string `json:"email"`
	Password     string `json:"password,omitempty"` // Contraseña para el login (opcional al actualizar)
	Telefono     string `json:"telefono"`
	Especialidad string `json:"especialidad"`
	ImagenURL    string `json:"imagen_url,omitempty"` // URL de la foto de perfil
	Activo       *bool  `json:"activo"`
}

// UpdateAdvisor - Actualiza un asesor existente
func (h *AdvisorHandler) UpdateAdvisor(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	id := c.Params("id")

	var req UpdateAdvisorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Buscar asesor existente con su usuario
	var advisor models.Advisor
	if err := db.DB.Preload("User").First(&advisor, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Asesor no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesor",
		})
	}

	// Verificar si el email ya existe (excluyendo el asesor actual)
	if req.Email != "" && req.Email != advisor.Email {
		var existingAdvisor models.Advisor
		if err := db.DB.Where("email = ? AND id != ?", req.Email, id).First(&existingAdvisor).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El email ya está registrado",
			})
		}
	}

	// Actualizar campos del asesor solo si se proporcionan
	if req.Nombre != "" {
		advisor.Nombre = req.Nombre
	}
	if req.Email != "" {
		advisor.Email = req.Email
	}
	if req.Telefono != "" {
		advisor.Telefono = req.Telefono
	}
	if req.Especialidad != "" {
		advisor.Especialidad = req.Especialidad
	}
	// ImagenURL se puede actualizar (incluso a vacío para eliminar)
	advisor.ImagenURL = req.ImagenURL
	if req.Activo != nil {
		advisor.Activo = *req.Activo
	}

	// Actualizar el usuario asociado si existe
	if advisor.User != nil {
		// Actualizar nombre y email del usuario
		if req.Nombre != "" {
			advisor.User.Nombre = req.Nombre
		}
		if req.Email != "" {
			advisor.User.Email = req.Email
		}
		// Actualizar contraseña si se proporciona
		if req.Password != "" {
			if len(req.Password) < 6 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "La contraseña debe tener al menos 6 caracteres",
				})
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error procesando la contraseña",
				})
			}
			advisor.User.Password = string(hashedPassword)
		}
		if req.Activo != nil {
			advisor.User.Activo = *req.Activo
		}

		// Guardar usuario
		if err := db.DB.Save(&advisor.User).Error; err != nil {
			fmt.Printf("❌ ERROR actualizando usuario: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error actualizando el usuario",
			})
		}
	}

	if err := db.DB.Save(&advisor).Error; err != nil {
		fmt.Printf("❌ ERROR actualizando asesor: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando el asesor",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Asesor actualizado exitosamente",
		"advisor": advisor,
	})
}

// DeleteAdvisor - Elimina un asesor (soft delete)
func (h *AdvisorHandler) DeleteAdvisor(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	id := c.Params("id")

	// Verificar que el asesor existe con su usuario
	var advisor models.Advisor
	if err := db.DB.Preload("User").First(&advisor, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Asesor no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesor",
		})
	}

	// Soft delete del asesor
	if err := db.DB.Delete(&advisor).Error; err != nil {
		fmt.Printf("❌ ERROR eliminando asesor: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando el asesor",
		})
	}

	// Soft delete del usuario asociado si existe
	if advisor.User != nil {
		if err := db.DB.Delete(&advisor.User).Error; err != nil {
			fmt.Printf("❌ ERROR eliminando usuario: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error eliminando el usuario",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Asesor eliminado exitosamente",
	})
}

// ToggleAdvisorActive - Activa/desactiva un asesor
func (h *AdvisorHandler) ToggleAdvisorActive(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	id := c.Params("id")

	// Buscar asesor con su usuario
	var advisor models.Advisor
	if err := db.DB.Preload("User").First(&advisor, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Asesor no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesor",
		})
	}

	// Toggle estado
	advisor.Activo = !advisor.Activo

	// Actualizar el estado del usuario también
	if advisor.User != nil {
		advisor.User.Activo = advisor.Activo
		if err := db.DB.Save(&advisor.User).Error; err != nil {
			fmt.Printf("❌ ERROR actualizando estado del usuario: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error actualizando estado del usuario",
			})
		}
	}

	if err := db.DB.Save(&advisor).Error; err != nil {
		fmt.Printf("❌ ERROR actualizando estado: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando estado del asesor",
		})
	}

	estado := "inactivado"
	if advisor.Activo {
		estado = "activado"
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Asesor %s exitosamente", estado),
		"advisor": advisor,
	})
}

// ========== NUEVOS ENDPOINTS PARA MÚLTIPLES ASESORES POR PROPIEDAD ==========

// GetPropertyAdvisors - Obtiene todos los asesores asignados a una propiedad
func (h *AdvisorHandler) GetPropertyAdvisors(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	propertyID := c.Params("id")

	// Obtener las relaciones PropertyAdvisor
	var propertyAdvisors []models.PropertyAdvisor
	if err := db.DB.Where("property_id = ?", propertyID).
		Preload("Advisor.User").
		Find(&propertyAdvisors).Error; err != nil {
		fmt.Printf("❌ ERROR obteniendo asesores de propiedad: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo asesores asignados",
		})
	}

	// Extraer solo los asesores
	advisors := make([]models.Advisor, 0, len(propertyAdvisors))
	for _, pa := range propertyAdvisors {
		advisor := pa.Advisor
		// Si el asesor no tiene avatar_url, usar el del usuario asociado
		if advisor.AvatarURL == "" && advisor.User != nil && advisor.User.AvatarURL != "" {
			advisor.AvatarURL = advisor.User.AvatarURL
		}
		advisors = append(advisors, advisor)
	}

	return c.JSON(fiber.Map{
		"advisors": advisors,
		"total":    len(advisors),
	})
}

// AssignAdvisorToProperty - Asigna un asesor a una propiedad (soporta múltiples)
func (h *AdvisorHandler) AssignAdvisorToProperty(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	propertyID := c.Params("id")

	var req struct {
		AdvisorID uint `json:"advisor_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if req.AdvisorID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "advisor_id es requerido",
		})
	}

	// Verificar que la propiedad existe
	var property models.Property
	if err := db.DB.First(&property, propertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada",
		})
	}

	// Verificar que el asesor existe
	var advisor models.Advisor
	if err := db.DB.First(&advisor, req.AdvisorID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Asesor no encontrado",
		})
	}

	// Verificar si ya está asignado
	var existingAssignment models.PropertyAdvisor
	err = db.DB.Where("property_id = ? AND advisor_id = ?", propertyID, req.AdvisorID).
		First(&existingAssignment).Error

	if err == nil {
		// Ya existe la asignación
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El asesor ya está asignado a esta propiedad",
		})
	}

	// Crear la asignación
	assignment := models.PropertyAdvisor{
		PropertyID: property.ID,
		AdvisorID:  advisor.ID,
	}

	if err := db.DB.Create(&assignment).Error; err != nil {
		fmt.Printf("❌ ERROR asignando asesor: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error asignando asesor a la propiedad",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Asesor asignado exitosamente",
		"advisor": advisor,
	})
}

// RemoveAdvisorFromProperty - Elimina un asesor de una propiedad
func (h *AdvisorHandler) RemoveAdvisorFromProperty(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	propertyID := c.Params("id")
	advisorID := c.Params("advisorId")

	// Buscar la asignación
	var assignment models.PropertyAdvisor
	if err := db.DB.Where("property_id = ? AND advisor_id = ?", propertyID, advisorID).
		First(&assignment).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Asignación no encontrada",
		})
	}

	// Eliminar la asignación
	if err := db.DB.Delete(&assignment).Error; err != nil {
		fmt.Printf("❌ ERROR eliminando asignación: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando asesor de la propiedad",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Asesor eliminado de la propiedad exitosamente",
	})
}

