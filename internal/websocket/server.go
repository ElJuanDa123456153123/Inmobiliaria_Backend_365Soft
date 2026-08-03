package websocket

import (
	"log"
	"os"
	"sync"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// WSMessage - Estructura de mensaje WebSocket
type WSMessage struct {
	Type    string      `json:"type"`
	Message interface{} `json:"message"`
}

// Hub - Mantiene las conexiones activas de los clientes
type Hub struct {
	// Clientes conectados (user_id -> múltiples conexiones WebSocket)
	clients    map[uint][]*websocket.Conn
	clientsMux sync.RWMutex

	// Canal de mensajes broadcast
	broadcast chan WSMessage

	// Base de datos
	db *gorm.DB
}

// NewHub - Crea un nuevo hub
func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		clients:   make(map[uint][]*websocket.Conn),
		broadcast: make(chan WSMessage),
		db:        db,
	}
}

// Run - Ejecuta el hub en una goroutine
func (h *Hub) Run() {
	for message := range h.broadcast {
		h.clientsMux.RLock()
		conns := make([]*websocket.Conn, 0)
		for _, userConns := range h.clients {
			conns = append(conns, userConns...)
		}
		h.clientsMux.RUnlock()

		for _, conn := range conns {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("⚠️  WebSocket: Error en broadcast: %v", err)
			}
		}
	}
}

// registerClient - Registra una conexión para un usuario
func (h *Hub) registerClient(userID uint, conn *websocket.Conn) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	h.clients[userID] = append(h.clients[userID], conn)
}

// unregisterClient - Elimina una conexión específica de un usuario
func (h *Hub) unregisterClient(userID uint, conn *websocket.Conn) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()

	conns := h.clients[userID]
	for i, c := range conns {
		if c == conn {
			h.clients[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// BroadcastMessage - Envía un mensaje a todos los clientes conectados
func (h *Hub) BroadcastMessage(msgType string, message interface{}) {
	h.broadcast <- WSMessage{
		Type:    msgType,
		Message: message,
	}
}

// SendToUser - Envía un mensaje a todas las conexiones de un usuario específico
func (h *Hub) SendToUser(userID uint, msgType string, message interface{}) error {
	h.clientsMux.RLock()
	conns, ok := h.clients[userID]
	connsCopy := make([]*websocket.Conn, len(conns))
	copy(connsCopy, conns)
	h.clientsMux.RUnlock()

	if !ok || len(connsCopy) == 0 {
		return nil
	}

	wsMsg := WSMessage{
		Type:    msgType,
		Message: message,
	}

	for _, conn := range connsCopy {
		if err := conn.WriteJSON(wsMsg); err != nil {
			log.Printf("⚠️  WebSocket: Error enviando a usuario %d: %v", userID, err)
		}
	}
	return nil
}

// GetConnectedClientsCount - Retorna el número total de conexiones activas
func (h *Hub) GetConnectedClientsCount() int {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()
	count := 0
	for _, conns := range h.clients {
		count += len(conns)
	}
	return count
}

var globalHub *Hub

// InitWebSocket - Inicializa el servidor WebSocket
func InitWebSocket() error {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return err
	}

	globalHub = NewHub(dbConfig.DB)
	go globalHub.Run()
	log.Println("🔌 WebSocket Hub iniciado")
	return nil
}

// GetHub - Obtiene el hub global
func GetHub() *Hub {
	return globalHub
}

// HandleWebSocket - Maneja conexiones WebSocket usando Fiber WebSocket
// ws://localhost:3001/ws?token=xxx
func HandleWebSocket(c *fiber.Ctx) error {
	// Obtener token del query param
	token := c.Query("token")
	if token == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Token requerido",
		})
	}

	// Verificar token y obtener user_id
	userID, err := verifyTokenAndGetUserID(token)
	if err != nil {
		log.Printf("⚠️  WebSocket: Token inválido: %v", err)
		return c.Status(401).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	log.Printf("🔌 WebSocket: Usuario %d intentando conectar", userID)

	// Usar WebSocket de Fiber
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("user_id", userID)

		// Crear conexión WebSocket
		websocket.New(func(c *websocket.Conn) {
			// Registrar cliente (con mutex)
			globalHub.registerClient(userID, c)

			log.Printf("✅ WebSocket: Cliente %d conectado (conexiones activas: %d)", userID, globalHub.GetConnectedClientsCount())

			defer func() {
				globalHub.unregisterClient(userID, c)
				c.Close()
				log.Printf("🔌 WebSocket: Cliente %d desconectado", userID)
			}()

			// Loop de lectura
			for {
				messageType, msg, err := c.ReadMessage()
				if err != nil {
					log.Printf("⚠️  WebSocket: Error leyendo mensaje: %v", err)
					break
				}

				// Procesar mensajes recibidos (si es necesario)
				log.Printf("📨 WebSocket: Mensaje recibido de cliente %d, tipo: %d, len: %d", userID, messageType, len(msg))
			}
		})(c)
		return nil
	}

	return c.SendStatus(fiber.StatusUpgradeRequired)
}

// verifyTokenAndGetUserID - Verifica el token JWT y retorna el user_id
func verifyTokenAndGetUserID(tokenString string) (uint, error) {
	// Obtener JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "tu_secreto_super_seguro_cambialo_en_produccion"
	}

	// Parsear y validar token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return 0, err
	}

	// Extraer claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrSignatureInvalid
	}

	// Obtener userID desde claims
	userIDFloat, ok := (*claims)["user_id"].(float64)
	if !ok {
		return 0, jwt.ErrSignatureInvalid
	}

	userID := uint(userIDFloat)

	// Verificar que el usuario existe en la base de datos
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return 0, err
	}

	var user models.User
	if err := dbConfig.DB.First(&user, userID).Error; err != nil {
		return 0, err
	}

	return userID, nil
}

// BroadcastNewMessage - Función auxiliar para broadcast de nuevo mensaje
func BroadcastNewMessage(message models.Message) {
	if globalHub != nil {
		globalHub.BroadcastMessage("new_message", fiber.Map{
			"message": message,
		})
	}
}

// SendNotificationToUser - Envía notificación a un usuario específico
func SendNotificationToUser(userID uint, notification map[string]interface{}) error {
	if globalHub != nil {
		return globalHub.SendToUser(userID, "notification", notification)
	}
	return nil
}
