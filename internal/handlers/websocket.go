package handlers

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/gofiber/websocket/v2"
)

// WebSocketHandler - Handler para conexiones WebSocket en tiempo real
type WebSocketHandler struct {
	Redis      *config.RedisClient
	clients    map[*websocket.Conn]bool
	clientsMux sync.Mutex
}

// NewWebSocketHandler - Crea un nuevo handler WebSocket
func NewWebSocketHandler(redis *config.RedisClient) *WebSocketHandler {
	return &WebSocketHandler{
		Redis:   redis,
		clients: make(map[*websocket.Conn]bool),
	}
}

// PropertyUpdateMessage - Mensaje de actualización de propiedad
type PropertyUpdateMessage struct {
	Action      string `json:"action"`       // created, updated, deleted, status_changed
	PropertyID  uint   `json:"property_id"`
	Data        interface{} `json:"data"`
	Timestamp   int64  `json:"timestamp"`
}

// HandlePropertyUpdates - Maneja conexiones WebSocket para actualizaciones de propiedades
func (h *WebSocketHandler) HandlePropertyUpdates(c *websocket.Conn) {
	// Verificar que Redis esté disponible
	if h.Redis == nil {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"Redis no disponible"}`))
		c.Close()
		return
	}

	// Agregar cliente a la lista
	h.clientsMux.Lock()
	h.clients[c] = true
	h.clientsMux.Unlock()

	// Limpiar al desconectar
	defer func() {
		h.clientsMux.Lock()
		delete(h.clients, c)
		h.clientsMux.Unlock()
		c.Close()
	}()

	// Suscribirse al canal de Redis
	pubsub, err := h.Redis.SubscribeToPropertyUpdates()
	if err != nil {
		log.Printf("Error suscribiéndose a Redis: %v", err)
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"Error suscribiéndose a actualizaciones"}`))
		return
	}
	defer pubsub.Close()

	// Canal para mensajes
	ch := pubsub.Channel()

	// Enviar mensaje de bienvenida
	welcomeMsg := map[string]interface{}{
		"type":    "connected",
		"message": "Conectado a actualizaciones de propiedades en tiempo real",
	}
	welcomeBytes, _ := json.Marshal(welcomeMsg)
	c.WriteMessage(websocket.TextMessage, welcomeBytes)

	// Escuchar mensajes
	for {
		select {
		case msg := <-ch:
			// Parsear mensaje de Redis
			var updateMsg PropertyUpdateMessage
			if err := json.Unmarshal([]byte(msg.Payload), &updateMsg); err != nil {
				log.Printf("Error parseando mensaje de Redis: %v", err)
				continue
			}

			// Enviar a todos los clientes conectados
			h.clientsMux.Lock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("Error enviando mensaje a cliente: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.clientsMux.Unlock()

		default:
			// Leer mensajes del cliente (ping/pong, etc.)
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}
}

// BroadcastToClients - Envía un mensaje a todos los clientes conectados (útil sin Redis)
func (h *WebSocketHandler) BroadcastToClients(message interface{}) error {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for client := range h.clients {
		if err := client.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("Error enviando mensaje a cliente: %v", err)
			client.Close()
			delete(h.clients, client)
		}
	}

	return nil
}

// GetConnectedClientsCount - Retorna el número de clientes conectados
func (h *WebSocketHandler) GetConnectedClientsCount() int {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	return len(h.clients)
}
