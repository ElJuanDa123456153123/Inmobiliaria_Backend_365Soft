package handlers

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthHandler - Handler para autenticación
type AuthHandler struct {
	DB    *config.Database
	Redis *config.RedisClient
}

// NewAuthHandler - Crea un nuevo handler de autenticación
func NewAuthHandler() (*AuthHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar Redis (opcional, si falla continúa sin rate limit reset)
	redis, err := config.NewRedis()
	if err != nil {
		// Redis no está disponible, continuar sin rate limit reset
		fmt.Println("⚠️  Redis no disponible, ejecutando sin reset de rate limit")
		redis = nil
	}

	return &AuthHandler{
		DB:    dbConfig,
		Redis: redis,
	}, nil
}

// clearRateLimitKeys - Limpia los contadores de rate limit para una IP después de login exitoso
func (h *AuthHandler) clearRateLimitKeys(clientIP string) {
	// Si Redis no está disponible, no hacer nada
	if h.Redis == nil {
		return
	}

	now := time.Now().Unix()
	minuteKey := fmt.Sprintf("ratelimit:%s:%d", clientIP, now/60)
	hourKey := fmt.Sprintf("ratelimit:%s:%d", clientIP, now/3600)

	// Eliminar las claves de rate limit para el minuto y hora actuales
	h.Redis.Client.Del(h.Redis.Ctx, minuteKey)
	h.Redis.Client.Del(h.Redis.Ctx, hourKey)
}

// RegistroRequest - Estructura para registro
type RegistroRequest struct {
	Nombre             string `json:"nombre"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	Telefono           string `json:"telefono"`
	Rol                string `json:"rol"` // "usuario", "agente", "inmobiliaria"
	NombreInmobiliaria string `json:"nombre_inmobiliaria,omitempty"`
	NIT                string `json:"nit,omitempty"`
}

// LoginRequest - Estructura para login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse - Respuesta de autenticación
type AuthResponse struct {
	Token string       `json:"token"`
	User  models.User  `json:"user"`
}

// Claims - Estructura de claims JWT
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
	jwt.RegisteredClaims
}

// Register - Handler para registro de usuarios
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	// Parsear request body
	var req RegistroRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar campos requeridos
	if req.Nombre == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre, email y password son requeridos",
		})
	}

	// Validar rol
	rol := models.RoleUsuario
	if req.Rol != "" {
		switch req.Rol {
		case "usuario":
			rol = models.RoleUsuario
		case "agente":
			rol = models.RoleAgente
		case "inmobiliaria":
			rol = models.RoleInmobiliaria
			if req.NombreInmobiliaria == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Nombre de inmobiliaria es requerido para rol inmobiliaria",
				})
			}
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Rol inválido",
			})
		}
	}

	// Verificar si el email ya existe
	var existingUser models.User
	result := h.DB.DB.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El email ya está registrado",
		})
	}

	// Hashear password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error hasheando password",
		})
	}

	// Crear usuario
	user := models.User{
		Nombre:             req.Nombre,
		Email:              req.Email,
		Password:           string(hashedPassword),
		Telefono:           req.Telefono,
		Rol:                rol,
		NombreInmobiliaria: req.NombreInmobiliaria,
		NIT:                req.NIT,
		Activo:             true,
		Verificado:         false,
	}

	// Guardar en base de datos
	if err := h.DB.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando usuario",
		})
	}

	// Si el rol es agente o inmobiliaria, crear perfil de asesor automáticamente
	if rol == models.RoleAgente || rol == models.RoleInmobiliaria {
		advisor := models.Advisor{
			Nombre:    user.Nombre,
			Email:     user.Email,
			Telefono:  user.Telefono,
			AvatarURL: user.AvatarURL,
			Activo:    true,
			UserID:    &user.ID,
		}
		if err := h.DB.DB.Create(&advisor).Error; err != nil {
			// Log error pero continuar (el usuario ya fue creado)
			fmt.Printf("⚠️  Error creando perfil de asesor para usuario %d: %v\n", user.ID, err)
		} else {
			fmt.Printf("✅ Perfil de asesor creado para usuario %d (%s)\n", user.ID, user.Nombre)
		}
	}

	// Generar token JWT
	token, err := h.generateToken(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando token",
		})
	}

	// Retornar respuesta
	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login - Handler para login de usuarios
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	// Parsear request body
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar campos
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email y password son requeridos",
		})
	}

	// PRIMERO: Verificar si es el administrador harcoded
	const adminEmail = "admin@365soft.com"
	const adminPassword = "admin123"
	const adminName = "Administrador"

	if req.Email == adminEmail && req.Password == adminPassword {
		// Crear usuario admin virtual
		adminUser := &models.User{
			ID:         0,
			Nombre:     adminName,
			Email:      adminEmail,
			Rol:        "admin",
			Activo:     true,
			Verificado: true,
		}

		// Generar token JWT
		token, err := h.generateToken(adminUser)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error generando token",
			})
		}

		// Limpiar rate limit después de login exitoso
		h.clearRateLimitKeys(c.IP())

		// Retornar respuesta de admin
		return c.JSON(AuthResponse{
			Token: token,
			User:  *adminUser,
		})
	}

	// SEGUNDO: Buscar usuario normal en la base de datos
	var user models.User
	result := h.DB.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Credenciales inválidas",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando usuario",
		})
	}

	// Verificar password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Verificar si el usuario está activo
	if !user.Activo {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Usuario inactivo",
		})
	}

	// Generar token JWT
	token, err := h.generateToken(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando token",
		})
	}

	// Limpiar rate limit después de login exitoso
	h.clearRateLimitKeys(c.IP())

	// Retornar respuesta
	return c.JSON(AuthResponse{
		Token: token,
		User:  user,
	})
}

// GetMe - Obtiene el usuario actual del token
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	// Obtener usuario del contexto (inyectado por middleware)
	user := c.Locals("user").(*models.User)

	return c.JSON(user)
}

// UpdateProfileRequest - Estructura para actualizar perfil
type UpdateProfileRequest struct {
	Nombre           string     `json:"nombre"`
	Apellido         string     `json:"apellido"`
	Email            string     `json:"email"`
	Telefono         string     `json:"telefono"`
	FechaNacimiento  *time.Time `json:"fecha_nacimiento,omitempty"`
	Sexo             string     `json:"sexo,omitempty"`
	Direccion        string     `json:"direccion,omitempty"`
	CI               string     `json:"ci,omitempty"`
	AvatarURL        string     `json:"avatar_url,omitempty"`
	NuevaPassword    string     `json:"nueva_password,omitempty"`
}

// DatosPublicosRequest - Estructura para actualizar datos públicos del vendedor
type DatosPublicosRequest struct {
	NombreContacto  string `json:"nombre_contacto"`
	TelefonoContacto string `json:"telefono_contacto"`
	HorarioContacto string `json:"horario_contacto"`
}

// UpdateProfile - Actualiza el perfil del usuario actual
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	// Obtener usuario del contexto (inyectado por middleware)
	user := c.Locals("user").(*models.User)

	// Parsear request body
	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar campos requeridos
	if req.Nombre == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El nombre es requerido",
		})
	}

	// Si se quiere cambiar el email, verificar que no exista
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		result := h.DB.DB.Where("email = ?", req.Email).First(&existingUser)
		if result.Error == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "El email ya está en uso",
			})
		}
		user.Email = req.Email
	}

	// Actualizar campos básicos
	user.Nombre = req.Nombre
	user.Apellido = req.Apellido
	user.Telefono = req.Telefono
	user.FechaNacimiento = req.FechaNacimiento
	user.Sexo = req.Sexo
	user.Direccion = req.Direccion
	user.CI = req.CI
	user.AvatarURL = req.AvatarURL

	// Si se proporciona nueva contraseña, hashearla
	if req.NuevaPassword != "" {
		if len(req.NuevaPassword) < 6 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "La contraseña debe tener al menos 6 caracteres",
			})
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NuevaPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error procesando la contraseña",
			})
		}
		user.Password = string(hashedPassword)
	}

	// Guardar cambios en la base de datos
	if err := h.DB.DB.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando el perfil",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Perfil actualizado exitosamente",
		"user":    user,
	})
}

// UpdateDatosPublicos - Actualiza los datos públicos del vendedor
func (h *AuthHandler) UpdateDatosPublicos(c *fiber.Ctx) error {
	// Obtener usuario del contexto (inyectado por middleware)
	user := c.Locals("user").(*models.User)

	// Parsear request body
	var req DatosPublicosRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar campos requeridos
	if req.NombreContacto == "" || req.TelefonoContacto == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre de contacto y teléfono son requeridos",
		})
	}

	// Actualizar campos de datos públicos
	user.NombreContacto = req.NombreContacto
	user.TelefonoContacto = req.TelefonoContacto
	user.HorarioContacto = req.HorarioContacto

	// Guardar cambios en la base de datos
	if err := h.DB.DB.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando datos públicos",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Datos públicos actualizados exitosamente",
		"user":    user,
	})
}

// AdminLogin - Handler para login de administradores (verificado contra la BD)
func (h *AuthHandler) AdminLogin(c *fiber.Ctx) error {
	// Parsear request body
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar campos
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email y password son requeridos",
		})
	}

	// Buscar usuario en la base de datos
	var user models.User
	result := h.DB.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Credenciales inválidas",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando usuario",
		})
	}

	// Verificar que sea administrador
	if user.Rol != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Acceso denegado: no es administrador",
		})
	}

	// Verificar password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Verificar si el usuario está activo
	if !user.Activo {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Usuario inactivo",
		})
	}

	// Generar token JWT
	token, err := h.generateToken(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando token",
		})
	}

	// Retornar respuesta
	return c.JSON(AuthResponse{
		Token: token,
		User:  user,
	})
}

// generateToken - Genera un token JWT para un usuario
func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	// Obtener JWT secret desde variables de entorno
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "tu_secreto_super_seguro_cambialo_en_produccion"
	}

	// Crear claims
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Rol:    string(user.Rol),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 horas
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Crear token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar token
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("error firmando token: %w", err)
	}

	return tokenString, nil
}

// ValidateTokenAndGetUser - Valida un token JWT y retorna el usuario
func (h *AuthHandler) ValidateTokenAndGetUser(tokenString string) (*models.User, error) {
	// Obtener JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "tu_secreto_super_seguro_cambialo_en_produccion"
	}

	// Parsear token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validar método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Extraer claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token inválido")
	}

	// Buscar usuario
	var user models.User
	result := h.DB.DB.First(&user, claims.UserID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}
