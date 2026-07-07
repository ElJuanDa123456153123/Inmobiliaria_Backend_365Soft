package handlers

import (
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminHandler - Handler para administración
type AdminHandler struct {
	DB *config.Database
}

// NewAdminHandler - Crea un nuevo handler de administración
func NewAdminHandler() (*AdminHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &AdminHandler{
		DB: dbConfig,
	}, nil
}

// GetUsers - Obtiene todos los usuarios
func (h *AdminHandler) GetUsers(c *fiber.Ctx) error {
	var users []models.User

	// Crear nueva conexión por cada request (evita database closed)
	db, err := config.GetDatabase()
	if err != nil {
		fmt.Printf("❌ ERROR creando conexión: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	if err := db.DB.Find(&users).Error; err != nil {
		fmt.Printf("❌ ERROR en consulta: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo usuarios",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
	})
}

// CreateUserRequest params for creating a user
type CreateUserRequest struct {
	Nombre             string          `json:"nombre"`
	Email              string          `json:"email"`
	Password           string          `json:"password"`
	Telefono           string          `json:"telefono"`
	Rol                models.UserRole `json:"rol"`
	NombreInmobiliaria string          `json:"nombre_inmobiliaria,omitempty"`
	NIT                string          `json:"nit,omitempty"`
	Activo             bool            `json:"activo"`
}

// CreateUser - Crea un nuevo usuario (Admin access)
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validaciones básicas
	if req.Nombre == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre, email y la contraseña son requeridos",
		})
	}

	// Verificar si el email ya existe
	var existingUser models.User
	if err := db.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El email ya está registrado",
		})
	}

	// Encriptar contraseña obligatoriamente (bcrypt)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error hasheando contraseña",
		})
	}

	// Rol por defecto si no es provisto o es inválido
	if req.Rol == "" {
		req.Rol = models.RoleUsuario
	}

	user := models.User{
		Nombre:             req.Nombre,
		Email:              req.Email,
		Password:           string(hashedPassword),
		Telefono:           req.Telefono,
		Rol:                req.Rol,
		NombreInmobiliaria: req.NombreInmobiliaria,
		NIT:                req.NIT,
		Activo:             req.Activo,
		Verificado:         true, // Creado por admin se asume verificado
	}

	if err := db.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el usuario",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Usuario creado exitosamente",
		"user":    user,
	})
}

// UpdateUserRequest params for updating a user
type UpdateUserRequest struct {
	Nombre             string          `json:"nombre"`
	Email              string          `json:"email"`
	Password           string          `json:"password,omitempty"`     // Opcional
	Telefono           string          `json:"telefono"`
	Rol                models.UserRole `json:"rol"`
	NombreInmobiliaria string          `json:"nombre_inmobiliaria,omitempty"`
	NIT                string          `json:"nit,omitempty"`
	Activo             *bool           `json:"activo,omitempty"` // Usamos puntero para saber si fue provisto
}

// UpdateUser - Actualiza un usuario existente
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario es requerido",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Buscar el usuario existente
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando el usuario",
		})
	}

	// Verificar si el email es diferente y que el nuevo no esté usado
	if req.Email != "" && req.Email != user.Email {
		var count int64
		h.DB.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El email ingresado ya está en uso por otro usuario",
			})
		}
		user.Email = req.Email
	}

	// Si se ingresó una nueva contraseña, encriptarla (bcrypt)
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error encritando la nueva contraseña",
			})
		}
		user.Password = string(hashedPassword)
	}

	// Actualizar otros campos si fueron provistos
	if req.Nombre != "" {
		user.Nombre = req.Nombre
	}
	if req.Telefono != "" {
		user.Telefono = req.Telefono
	}
	if req.Rol != "" {
		user.Rol = req.Rol
	}
	if req.Activo != nil {
		user.Activo = *req.Activo
	}
	
	// Campos de inmobiliaria
	if req.Rol == models.RoleInmobiliaria {
		user.NombreInmobiliaria = req.NombreInmobiliaria
		user.NIT = req.NIT
	} else if req.Rol == models.RoleUsuario || req.Rol == models.RoleAdmin {
		// Limpiar campos inmobliaria si el rol cambió a uno que no aplica
		user.NombreInmobiliaria = ""
		user.NIT = ""
	}

	// Guardar los cambios
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando el usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario actualizado exitosamente",
		"user":    user,
	})
}

// ToggleUserActive - Activa/desactiva un usuario
func (h *AdminHandler) ToggleUserActive(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Obtener ID del usuario
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario es requerido",
		})
	}

	// Buscar usuario
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Usuario no encontrado",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando usuario",
		})
	}

	// Toggle activo
	user.Activo = !user.Activo

	// Guardar cambios
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario actualizado exitosamente",
		"user":    user,
	})
}

// DeleteUser - Elimina un usuario
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Obtener ID del usuario
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario es requerido",
		})
	}

	// Eliminar usuario
	if err := db.DB.Delete(&models.User{}, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario eliminado exitosamente",
	})
}

// MigrateData - Migra datos existentes para sincronizar usuarios y asesores,
// y actualizar propiedades para mostrar vendedor por defecto
func (h *AdminHandler) MigrateData(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	results := fiber.Map{
		"propiedades_actualizadas": 0,
		"asesores_creados":         0,
		"errores":                  []string{},
	}

	// 1. Actualizar todas las propiedades para que mostrar_vendedor = true
	fmt.Println("🔄 Iniciando migración de mostrar_vendedor en propiedades...")
	var propsUpdated int64
	if err := db.DB.Model(&models.Property{}).
		Where("mostrar_vendedor = ? OR mostrar_vendedor IS NULL", false).
		Update("mostrar_vendedor", true).Error; err != nil {
		fmt.Printf("❌ Error actualizando propiedades: %v\n", err)
		results["errores"] = append(results["errores"].([]string), fmt.Sprintf("Error actualizando propiedades: %v", err))
	} else {
		propsUpdated = db.DB.RowsAffected
		results["propiedades_actualizadas"] = int(propsUpdated)
		fmt.Printf("✅ Propiedades actualizadas: %d\n", propsUpdated)
	}

	// 2. Crear registros en asesores para usuarios con rol "agente" o "inmobiliaria" que no tengan uno
	fmt.Println("🔄 Iniciando migración de asesores para usuarios con rol 'agente' o 'inmobiliaria'...")

	// Buscar usuarios con rol "agente" o "inmobiliaria" sin asesor correspondiente
	var users []models.User
	if err := db.DB.Where("(rol = ? OR rol = ?) AND deleted_at IS NULL", models.RoleAgente, models.RoleInmobiliaria).Find(&users).Error; err != nil {
		fmt.Printf("❌ Error buscando usuarios: %v\n", err)
		results["errores"] = append(results["errores"].([]string), fmt.Sprintf("Error buscando usuarios: %v", err))
		return c.Status(fiber.StatusInternalServerError).JSON(results)
	}

	advisorsCreated := 0
	for _, user := range users {
		// Verificar si ya existe un asesor para este usuario
		var existingAdvisor models.Advisor
		if err := db.DB.Where("user_id = ?", user.ID).First(&existingAdvisor).Error; err == nil {
			// Ya existe, continuar
			continue
		}

		// Crear nuevo asesor
		advisor := models.Advisor{
			Nombre:    user.Nombre,
			Email:     user.Email,
			Telefono:  user.Telefono,
			AvatarURL: user.AvatarURL,
			Activo:    user.Activo,
			UserID:    &user.ID,
		}

		if err := db.DB.Create(&advisor).Error; err != nil {
			fmt.Printf("❌ Error creando asesor para usuario %d: %v\n", user.ID, err)
			results["errores"] = append(results["errores"].([]string), fmt.Sprintf("Error creando asesor para usuario %d: %v", user.ID, err))
		} else {
			advisorsCreated++
			fmt.Printf("✅ Asesor creado para usuario %d (%s)\n", user.ID, user.Nombre)
		}
	}

	results["asesores_creados"] = advisorsCreated

	fmt.Printf("✅ Migración completada: %d propiedades actualizadas, %d asesores creados\n", propsUpdated, advisorsCreated)

	return c.JSON(results)
}
