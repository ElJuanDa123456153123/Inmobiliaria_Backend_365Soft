package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// GoogleUserInfo - Estructura para respuesta de Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// GoogleAuthHandler - Handler para autenticación con Google
type GoogleAuthHandler struct {
	DB        *gorm.DB
	OAuthConf *oauth2.Config
}

// NewGoogleAuthHandler - Crea un nuevo handler de Google Auth
func NewGoogleAuthHandler() (*GoogleAuthHandler, error) {
	dbConfig, err := NewAuthHandler()
	if err != nil {
		return nil, fmt.Errorf("error conectando a database: %w", err)
	}

	// Obtener variables de entorno
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	callbackURL := os.Getenv("GOOGLE_CALLBACK_URL")

	if googleClientID == "" || googleClientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID y GOOGLE_CLIENT_SECRET son requeridos")
	}

	// Si no hay callback URL, usar la por defecto
	if callbackURL == "" {
		callbackURL = "http://localhost:3001/api/auth/google/callback"
	}

	// Configurar OAuth
	conf := &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  callbackURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleAuthHandler{
		DB:        dbConfig.DB,
		OAuthConf: conf,
	}, nil
}

// GoogleLogin - Inicia el flujo de OAuth con Google
func (h *GoogleAuthHandler) GoogleLogin(c *fiber.Ctx) error {
	// Generar URL de OAuth
	url := h.OAuthConf.AuthCodeURL("state", oauth2.AccessTypeOffline)

	// Redirigir a Google
	return c.Redirect(url)
}

// GoogleCallback - Maneja el callback de Google
func (h *GoogleAuthHandler) GoogleCallback(c *fiber.Ctx) error {
	// Obtener el código de autorización
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Código de autorización no encontrado",
		})
	}

	// Intercambiar el código por un token
	token, err := h.OAuthConf.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Error intercambiando código por token: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error intercambiando código por token",
		})
	}

	// Obtener información del usuario
	client := h.OAuthConf.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Error obteniendo info de usuario: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error obteniendo información del usuario",
		})
	}
	defer resp.Body.Close()

	// Parsear respuesta
	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		log.Printf("Error parseando respuesta de Google: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error procesando información del usuario",
		})
	}

	// Buscar o crear usuario
	var user models.User
	result := h.DB.Where("email = ?", googleUser.Email).First(&user)

	if result.Error != nil {
		// Usuario no existe, crear nuevo
		user = models.User{
			Nombre:     googleUser.Name,
			Email:      googleUser.Email,
			Telefono:   "", // Se completará después
			Rol:        models.RoleUsuario,
			Activo:     true,
			Verificado: googleUser.VerifiedEmail,
			Password:   "", // No tiene password (solo Google)
		}

		if err := h.DB.Create(&user).Error; err != nil {
			log.Printf("Error creando usuario: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error creando usuario",
			})
		}
	} else {
		// Usuario existe, actualizar información
		user.Nombre = googleUser.Name
		user.Verificado = googleUser.VerifiedEmail
		if err := h.DB.Save(&user).Error; err != nil {
			log.Printf("Error actualizando usuario: %v", err)
		}
	}

	// Generar token JWT
	authHandler := &AuthHandler{DB: h.DB}
	jwtToken, err := authHandler.generateToken(&user)
	if err != nil {
		log.Printf("Error generando token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando token",
		})
	}

	// Verificar si el perfil está completo (tiene teléfono)
	perfilCompleto := user.Telefono != ""

	// Redirigir al frontend con el token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	var redirectURL string
	if perfilCompleto {
		// Perfil completo, redirigir a página de éxito
		redirectURL = fmt.Sprintf("%s/auth/success?token=%s&nombre=%s",
			frontendURL, jwtToken, user.Nombre)
	} else {
		// Perfil incompleto, redirigir a completar perfil
		redirectURL = fmt.Sprintf("%s/auth/complete-profile?token=%s&nombre=%s&email=%s",
			frontendURL, jwtToken, user.Nombre, user.Email)
	}

	return c.Redirect(redirectURL)
}

// CompleteGoogleProfile - Completa el perfil de usuario de Google
func (h *GoogleAuthHandler) CompleteGoogleProfile(c *fiber.Ctx) error {
	// Obtener token del header
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token no proporcionado",
		})
	}

	// Eliminar "Bearer " del token
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Parsear request body
	var req struct {
		Telefono string `json:"telefono"`
		Rol      string `json:"rol"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar
	if req.Telefono == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Teléfono es requerido",
		})
	}

	// Validar token y obtener usuario
	authHandler := &AuthHandler{DB: h.DB}
	user, err := authHandler.ValidateTokenAndGetUser(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	// Actualizar usuario
	user.Telefono = req.Telefono
	if req.Rol != "" {
		switch req.Rol {
		case "usuario":
			user.Rol = models.RoleUsuario
		case "agente":
			user.Rol = models.RoleAgente
		case "inmobiliaria":
			user.Rol = models.RoleInmobiliaria
		}
	}

	if err := h.DB.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando usuario",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Perfil completado exitosamente",
		"user":    user,
	})
}
