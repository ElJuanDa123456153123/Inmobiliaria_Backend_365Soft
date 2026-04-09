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
	DB *gorm.DB
}

// NewAuthHandler - Crea un nuevo handler de autenticación
func NewAuthHandler() (*AuthHandler, error) {
	dbConfig, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &AuthHandler{
		DB: dbConfig.DB,
	}, nil
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
	result := h.DB.Where("email = ?", req.Email).First(&existingUser)
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
	if err := h.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando usuario",
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

		// Retornar respuesta de admin
		return c.JSON(AuthResponse{
			Token: token,
			User:  *adminUser,
		})
	}

	// SEGUNDO: Buscar usuario normal en la base de datos
	var user models.User
	result := h.DB.Where("email = ?", req.Email).First(&user)
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

// AdminLogin - Handler para login de administradores (credenciales harcoded)
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

	// Credenciales hardcoded para administrador
	const adminEmail = "admin@365soft.com"
	const adminPassword = "admin123"
	const adminName = "Administrador"

	// Verificar credenciales
	if req.Email != adminEmail || req.Password != adminPassword {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Crear usuario admin virtual para el token
	adminUser := &models.User{
		ID:       0,
		Nombre:   adminName,
		Email:    adminEmail,
		Rol:      "admin",
		Activo:   true,
		Verificado: true,
	}

	// Generar token JWT
	token, err := h.generateToken(adminUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando token",
		})
	}

	// Retornar respuesta
	return c.JSON(AuthResponse{
		Token: token,
		User:  *adminUser,
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
	result := h.DB.First(&user, claims.UserID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}
