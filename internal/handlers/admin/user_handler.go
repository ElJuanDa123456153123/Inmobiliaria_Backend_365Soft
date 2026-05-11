package admin

import (
	"fmt"
	"os"
	stdpath "path/filepath"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler - Handler para administración de usuarios
type UserHandler struct {
	DB *config.Database
}

// NewUserHandler - Crea un nuevo handler de usuarios
func NewUserHandler() (*UserHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &UserHandler{
		DB: dbConfig,
	}, nil
}

// GetUsers - Obtiene todos los usuarios
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
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
	AvatarURL          string          `json:"avatar_url,omitempty"` // URL del avatar
	Activo             bool            `json:"activo"`
}

// CreateUser - Crea un nuevo usuario (Admin access)
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
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
		AvatarURL:          req.AvatarURL, // Guardar avatar si se proporciona
		Activo:             req.Activo,
		Verificado:         true, // Creado por admin se asume verificado
	}

	// Debug log
	fmt.Printf("📝 Creando usuario con avatar_url: '%s'\n", req.AvatarURL)

	if err := db.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando el usuario",
		})
	}

	// Debug log
	fmt.Printf("✅ Usuario creado con ID: %d, avatar_url: '%s'\n", user.ID, user.AvatarURL)

	// Si el rol es "agente", crear automáticamente un registro en la tabla advisors
	// Esto asegura compatibilidad con el sistema de asignación de propiedades
	if user.Rol == models.RoleAgente {
		advisor := models.Advisor{
			Nombre:    user.Nombre,
			Email:     user.Email,
			Telefono:  user.Telefono,
			AvatarURL: user.AvatarURL,
			Activo:    user.Activo,
			UserID:    &user.ID,
		}

		if err := db.DB.Create(&advisor).Error; err != nil {
			fmt.Printf("⚠️  Usuario creado pero error al crear registro de advisor: %v\n", err)
			// No retornamos error porque el usuario ya se creó exitosamente
		} else {
			fmt.Printf("✅ Registro de advisor creado automáticamente para el usuario %d\n", user.ID)
		}
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
	Password           string          `json:"password,omitempty"` // Opcional
	Telefono           string          `json:"telefono"`
	Rol                models.UserRole `json:"rol"`
	NombreInmobiliaria string          `json:"nombre_inmobiliaria,omitempty"`
	NIT                string          `json:"nit,omitempty"`
	AvatarURL          string          `json:"avatar_url,omitempty"` // URL del avatar
	Activo             *bool           `json:"activo,omitempty"` // Usamos puntero para saber si fue provisto
}

// UpdateUser - Actualiza un usuario existente
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
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

	// Actualizar avatar si se proporcionó
	if req.AvatarURL != "" {
		// Si es diferente al actual, eliminar la imagen anterior
		if req.AvatarURL != user.AvatarURL {
			// Eliminar la imagen anterior si existe
			if user.AvatarURL != "" {
				parts := strings.Split(user.AvatarURL, "/")
				if len(parts) > 0 {
					oldFilename := parts[len(parts)-1]
					oldFilepath := stdpath.Join("uploads/avatars", oldFilename)

					// Verificar que el archivo existe antes de eliminar
					if _, err := os.Stat(oldFilepath); err == nil {
						// Eliminar el archivo
						if removeErr := os.Remove(oldFilepath); removeErr != nil {
							// Log del error pero continuar con la actualización
							fmt.Printf("⚠️  Error eliminando imagen anterior '%s': %v\n", oldFilepath, removeErr)
						} else {
							fmt.Printf("✅ Imagen anterior eliminada: %s\n", oldFilepath)
						}
					}
				}
			}

			// Actualizar con la nueva URL
			user.AvatarURL = req.AvatarURL
		}
		// Si es igual, no hacer nada (mantener la actual)
	}
	// Si req.AvatarURL está vacío, mantener el avatar actual (no eliminar)

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

	// Si el rol es "agente", actualizar también el registro en la tabla advisors
	if user.Rol == models.RoleAgente {
		var advisor models.Advisor
		if err := db.DB.Where("user_id = ?", user.ID).First(&advisor).Error; err == nil {
			// Actualizar campos del advisor
			advisor.Nombre = user.Nombre
			advisor.Email = user.Email
			advisor.Telefono = user.Telefono
			advisor.AvatarURL = user.AvatarURL
			advisor.Activo = user.Activo

			if err := db.DB.Save(&advisor).Error; err != nil {
				fmt.Printf("⚠️  Usuario actualizado pero error al actualizar registro de advisor: %v\n", err)
			} else {
				fmt.Printf("✅ Registro de advisor actualizado automáticamente para el usuario %d\n", user.ID)
			}
		} else {
			// Si no existe advisor, crearlo
			advisor := models.Advisor{
				Nombre:    user.Nombre,
				Email:     user.Email,
				Telefono:  user.Telefono,
				AvatarURL: user.AvatarURL,
				Activo:    user.Activo,
				UserID:    &user.ID,
			}

			if err := db.DB.Create(&advisor).Error; err != nil {
				fmt.Printf("⚠️  Usuario actualizado pero error al crear registro de advisor: %v\n", err)
			} else {
				fmt.Printf("✅ Registro de advisor creado automáticamente para el usuario %d\n", user.ID)
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "Usuario actualizado exitosamente",
		"user":    user,
	})
}

// ToggleUserActive - Activa/desactiva un usuario
func (h *UserHandler) ToggleUserActive(c *fiber.Ctx) error {
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
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
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

	// Primero buscar el usuario para obtener su avatar_url
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

	// Eliminar la imagen del avatar si existe
	if user.AvatarURL != "" {
		// Extraer el nombre del archivo de la URL
		// Formato esperado: http://localhost:3001/api/avatars/avatar-123456.png
		parts := strings.Split(user.AvatarURL, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			filepath := stdpath.Join("uploads/avatars", filename)

			// Verificar que el archivo existe antes de eliminar
			if _, err := os.Stat(filepath); err == nil {
				// Eliminar el archivo
				if removeErr := os.Remove(filepath); removeErr != nil {
					// Log del error pero continuar con la eliminación del usuario
					fmt.Printf("⚠️  Error eliminando imagen de avatar '%s': %v\n", filepath, removeErr)
				} else {
					fmt.Printf("✅ Imagen de avatar eliminada: %s\n", filepath)
				}
			}
		}
	}

	// Eliminar usuario de la base de datos
	if err := db.DB.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando usuario",
		})
	}

	// Si el rol era "agente", eliminar también el registro en la tabla advisors
	if user.Rol == models.RoleAgente {
		var advisor models.Advisor
		if err := db.DB.Where("user_id = ?", user.ID).First(&advisor).Error; err == nil {
			if err := db.DB.Delete(&advisor).Error; err != nil {
				fmt.Printf("⚠️  Usuario eliminado pero error al eliminar registro de advisor: %v\n", err)
			} else {
				fmt.Printf("✅ Registro de advisor eliminado automáticamente para el usuario %d\n", user.ID)
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "Usuario eliminado exitosamente",
	})
}
