package handlers

import (
	"fmt"
	"log"
	"os"
	stdpath "path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/websocket"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MessageHandler - Handler para mensajería entre usuarios
type MessageHandler struct {
	DB *gorm.DB
}

// NewMessageHandler - Crea un nuevo handler de mensajes
func NewMessageHandler() (*MessageHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &MessageHandler{
		DB: dbConfig.DB,
	}, nil
}

// SendMessageRequest - Estructura para enviar mensaje
type SendMessageRequest struct {
	ParaUserID     *uint  `json:"para_user_id" validate:"required"`
	PropertyID     *uint  `json:"property_id,omitempty"`
	Mensaje        string `json:"mensaje" validate:"required"`
	TipoMensaje    string `json:"tipo_mensaje,omitempty"`    // "texto", "imagen", "documento"
	ArchivoURL     string `json:"archivo_url,omitempty"`     // URL del archivo adjunto
	ArchivoNombre  string `json:"archivo_nombre,omitempty"`  // Nombre original del archivo
}

// SendMessage - Envía un mensaje a otro usuario
// POST /api/messages/send
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Parsear body
	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar
	if req.ParaUserID == nil || *req.ParaUserID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "para_user_id es requerido",
		})
	}

	if req.Mensaje == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "mensaje es requerido",
		})
	}

	// No enviarse mensaje a sí mismo
	if *req.ParaUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes enviarte mensajes a ti mismo",
		})
	}

	// Verificar que el destinatario existe
	var destUser models.User
	if err := h.DB.Where("id = ?", req.ParaUserID).First(&destUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario destinatario no encontrado",
		})
	}

	// Si hay PropertyID, verificar que existe
	if req.PropertyID != nil {
		var prop models.Property
		if err := h.DB.First(&prop, *req.PropertyID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Propiedad no encontrada",
			})
		}
	}

	// Crear mensaje
	message := models.Message{
		DeUserID:   userID,
		ParaUserID: *req.ParaUserID,
		PropertyID: req.PropertyID,
		Mensaje:    req.Mensaje,
		Leido:      false,
	}

	// Procesar archivo adjunto
	if req.TipoMensaje != "" && (req.TipoMensaje == "imagen" || req.TipoMensaje == "documento") {
		message.TipoMensaje = req.TipoMensaje
		if req.ArchivoURL != "" {
			message.ArchivoURL = &req.ArchivoURL
		}
		if req.ArchivoNombre != "" {
			message.ArchivoNombre = &req.ArchivoNombre
		}
	} else {
		message.TipoMensaje = "texto"
	}

	if err := h.DB.Create(&message).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error enviando mensaje",
		})
	}

	// Cargar relaciones
	h.DB.Preload("DeUser").Preload("ParaUser").Preload("Property").First(&message, message.ID)

	// 🔌 Broadcast del nuevo mensaje por WebSocket
	log.Printf("📨 Enviando mensaje WebSocket: de %d para %d", message.DeUserID, message.ParaUserID)
	wsHub := websocket.GetHub()
	if wsHub != nil {
		// Enviar al destinatario
		wsHub.SendToUser(message.ParaUserID, "new_message", fiber.Map{
			"message": message,
		})
		// Enviar confirmación al emisor
		wsHub.SendToUser(message.DeUserID, "message_sent", fiber.Map{
			"message": message,
		})
	} else {
		log.Println("⚠️  WebSocket Hub no disponible")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Mensaje enviado",
		"mensaje":  message,
	})
}

// ConversationData - Estructura de conversación para el frontend
type ConversationData struct {
	OtherUserID      uint    `json:"other_user_id"`
	OtherUserName    string  `json:"other_user_name"`
	OtherUserEmail   string  `json:"other_user_email"`
	PropertyID       *uint   `json:"property_id,omitempty"`
	PropertyTitulo   *string `json:"property_titulo,omitempty"`
	PropertyImagen   *string `json:"property_imagen,omitempty"`
	PropertyEstado   *string `json:"property_estado,omitempty"`
	PropertyExpirada *bool   `json:"property_expirada,omitempty"`
	PropertyTipoOperacion *string `json:"property_tipo_operacion,omitempty"`
	UnreadCount      int     `json:"unread_count"`
	LastMessage      *string `json:"last_message,omitempty"`
	LastMessageTime  *string `json:"last_message_time,omitempty"`
}

// GetConversations - Obtiene lista de conversaciones del usuario agrupadas por
// usuario y propiedad (cada consulta sobre una propiedad es un hilo independiente)
// GET /api/messages/conversations
func (h *MessageHandler) GetConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener todos los mensajes donde el usuario es emisor o receptor.
	// Property se precarga con Unscoped: una publicación expirada se elimina
	// (soft-delete) pero el chat debe seguir mostrando su estado/vigencia.
	var messages []models.Message
	if err := h.DB.Where("de_user_id = ? OR para_user_id = ?", userID, userID).
		Preload("DeUser").
		Preload("ParaUser").
		Preload("Property", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at DESC").
		Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo conversaciones",
		})
	}

	// Agrupar por (usuario + propiedad): cada propiedad es un hilo separado
	type convKey struct {
		other    uint
		property uint // 0 = chat general sin propiedad
	}
	conversations := make(map[convKey]*ConversationData)

	for _, msg := range messages {
		var otherUserID uint
		var isFromMe bool

		if msg.DeUserID == userID {
			otherUserID = msg.ParaUserID
			isFromMe = true
		} else {
			otherUserID = msg.DeUserID
			isFromMe = false
		}

		key := convKey{other: otherUserID}
		var propertyID *uint
		if msg.PropertyID != nil {
			pid := *msg.PropertyID
			key.property = pid
			propertyID = &pid
		}

		existing, ok := conversations[key]

		// Contar no leído de este mensaje
		unread := 0
		if !isFromMe && !msg.Leido {
			unread = 1
		}

		if !ok {
			// Los mensajes vienen DESC: el primero visto de cada hilo es el más reciente
			conv := &ConversationData{
				OtherUserID:    otherUserID,
				UnreadCount:    unread,
				LastMessage:    &msg.Mensaje,
				PropertyID:     propertyID,
			}
			timeStr := msg.CreatedAt.Format("2006-01-02 15:04")
			conv.LastMessageTime = &timeStr

			// Datos del otro usuario
			var otherUser models.User
			if isFromMe {
				otherUser = msg.ParaUser
			} else {
				otherUser = msg.DeUser
			}
			conv.OtherUserName = strings.TrimSpace(otherUser.Nombre + " " + otherUser.Apellido)
			conv.OtherUserEmail = otherUser.Email

			// Datos de la propiedad (título + imagen de portada + vigencia)
			if msg.Property != nil {
				titulo := msg.Property.Titulo
				conv.PropertyTitulo = &titulo
				estadoProp := string(msg.Property.EstadoPropiedad)
				conv.PropertyEstado = &estadoProp
				tipoOp := string(msg.Property.TipoOperacion)
				conv.PropertyTipoOperacion = &tipoOp
				if msg.Property.FechaExpiracion != nil {
					expirada := msg.Property.FechaExpiracion.Before(time.Now())
					conv.PropertyExpirada = &expirada
				} else {
					sinExpiracion := false
					conv.PropertyExpirada = &sinExpiracion
				}
				if len(msg.Property.PropertyImages) > 0 {
					img := msg.Property.PropertyImages[0]
					for _, im := range msg.Property.PropertyImages {
						if im.Principal {
							img = im
							break
						}
					}
					conv.PropertyImagen = &img.ImageURL
				}
			}

			conversations[key] = conv
		} else if unread > 0 {
			existing.UnreadCount += unread
		}
	}

	// Convertir a slice ordenado por último mensaje (desc)
	result := make([]*ConversationData, 0, len(conversations))
	for _, conv := range conversations {
		result = append(result, conv)
	}
	sort.Slice(result, func(i, j int) bool {
		a := ""
		b := ""
		if result[i].LastMessageTime != nil {
			a = *result[i].LastMessageTime
		}
		if result[j].LastMessageTime != nil {
			b = *result[j].LastMessageTime
		}
		return a > b
	})

	return c.JSON(fiber.Map{
		"conversations": result,
		"total":         len(result),
	})
}

// ConsultaData - Estructura de una consulta (conversación sobre una propiedad)
type ConsultaData struct {
	PropertyID        uint    `json:"property_id"`
	PropertyTitulo    string  `json:"property_titulo"`
	PropertyZona      string  `json:"property_zona"`
	PropertyPrecio    float64 `json:"property_precio"`
	PropertyMoneda    string  `json:"property_moneda"`
	PropertyOperacion string  `json:"property_operacion"`
	PropertyImagen    *string `json:"property_imagen,omitempty"`

	OtherUserID  uint   `json:"other_user_id"`
	OtherUserName string `json:"other_user_name"`
	OtherUserEmail string `json:"other_user_email"`

	TotalMensajes   int    `json:"total_mensajes"`
	UnreadCount     int    `json:"unread_count"`
	LastMessage     string `json:"last_message"`
	LastMessageDeMi bool   `json:"last_message_de_mi"`
	LastMessageTime string `json:"last_message_time"`
	FirstContactAt  string `json:"first_contact_at"`
}

// GetMyConsultas - Obtiene las consultas del usuario (conversaciones con propiedad asociada),
// agrupadas por propiedad y destinatario
// GET /api/panel/messages/consultas
func (h *MessageHandler) GetMyConsultas(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Mensajes del usuario que tengan una propiedad asociada (consultas)
	var messages []models.Message
	if err := h.DB.Where("(de_user_id = ? OR para_user_id = ?) AND property_id IS NOT NULL", userID, userID).
		Preload("DeUser").
		Preload("ParaUser").
		Preload("Property").
		Preload("Property.PropertyImages").
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo consultas",
		})
	}

	type threadKey struct {
		propertyID uint
		otherUser  uint
	}
	threads := make(map[threadKey]*ConsultaData)

	for _, msg := range messages {
		if msg.PropertyID == nil || msg.Property == nil {
			// Propiedad eliminada: se omite la consulta
			continue
		}

		var otherUserID uint
		var deMi bool
		if msg.DeUserID == userID {
			otherUserID = msg.ParaUserID
			deMi = true
		} else {
			otherUserID = msg.DeUserID
			deMi = false
		}

		key := threadKey{propertyID: *msg.PropertyID, otherUser: otherUserID}
		thread, ok := threads[key]
		if !ok {
			var otherUser models.User
			if deMi {
				otherUser = msg.ParaUser
			} else {
				otherUser = msg.DeUser
			}

			thread = &ConsultaData{
				PropertyID:        *msg.PropertyID,
				PropertyTitulo:    msg.Property.Titulo,
				PropertyZona:      msg.Property.Zona,
				PropertyPrecio:    msg.Property.Precio,
				PropertyMoneda:    msg.Property.Moneda,
				PropertyOperacion: string(msg.Property.TipoOperacion),
				OtherUserID:       otherUser.ID,
				OtherUserName:     strings.TrimSpace(otherUser.Nombre + " " + otherUser.Apellido),
				OtherUserEmail:    otherUser.Email,
				FirstContactAt:    msg.CreatedAt.Format("2006-01-02 15:04"),
			}

			// Imagen principal de la propiedad (o la primera)
			if len(msg.Property.PropertyImages) > 0 {
				img := msg.Property.PropertyImages[0]
				for _, im := range msg.Property.PropertyImages {
					if im.Principal {
						img = im
						break
					}
				}
				thread.PropertyImagen = &img.ImageURL
			}

			threads[key] = thread
		}

		thread.TotalMensajes++
		if !deMi && !msg.Leido {
			thread.UnreadCount++
		}

		// Los mensajes vienen ordenados ASC: el último sobrescribe
		thread.LastMessage = msg.Mensaje
		thread.LastMessageDeMi = deMi
		thread.LastMessageTime = msg.CreatedAt.Format("2006-01-02 15:04")
	}

	// Convertir a slice ordenado por último mensaje (desc)
	result := make([]*ConsultaData, 0, len(threads))
	for _, t := range threads {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastMessageTime > result[j].LastMessageTime
	})

	return c.JSON(fiber.Map{
		"consultas": result,
		"total":     len(result),
	})
}

// GetConversation - Obtiene conversación con un usuario específico
// GET /api/messages/thread/:user_id o GET /api/messages/thread/:property_id/:user_id
func (h *MessageHandler) GetConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener user_id
	otherUserID, err := c.ParamsInt("user_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario inválido",
		})
	}

	// Obtener todos los mensajes entre ambos usuarios
	var messages []models.Message
	query := h.DB.Where("(de_user_id = ? AND para_user_id = ?) OR (de_user_id = ? AND para_user_id = ?)",
		userID, otherUserID, otherUserID, userID)

	// Si hay property_id, filtrar también por propiedad
	propertyID := c.QueryInt("property_id")
	if propertyID > 0 {
		query = query.Where("property_id = ?", propertyID)
	}

	if err := query.
		Preload("DeUser").
		Preload("ParaUser").
		Preload("Property").
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo conversación",
		})
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"total":    len(messages),
	})
}

// MarkMessagesAsRead - Marca mensajes de una conversación como leídos
// POST /api/messages/mark-read/:user_id(?property_id=)
func (h *MessageHandler) MarkMessagesAsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	otherUserID, err := c.ParamsInt("user_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario inválido",
		})
	}

	// Scope opcional por propiedad (hilo específico)
	query := h.DB.Model(&models.Message{}).
		Where("de_user_id = ? AND para_user_id = ? AND leido = ?", otherUserID, userID, false)

	propertyID := c.QueryInt("property_id")
	if propertyID > 0 {
		query = query.Where("property_id = ?", propertyID)
	}

	if err := query.Update("leido", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error marcando mensajes como leídos",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Mensajes marcados como leídos",
	})
}

// DeleteConversation - Elimina (soft delete) los mensajes entre dos usuarios,
// opcionalmente limitado a un hilo de una propiedad
// DELETE /api/messages/conversation/:user_id(?property_id=)
func (h *MessageHandler) DeleteConversation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	otherUserID, err := c.ParamsInt("user_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de usuario inválido",
		})
	}

	// Soft delete de todos los mensajes entre ambos usuarios
	query := h.DB.Where(
		"(de_user_id = ? AND para_user_id = ?) OR (de_user_id = ? AND para_user_id = ?)",
		userID, otherUserID, otherUserID, userID,
	)

	// Scope opcional por propiedad (solo elimina el hilo de esa propiedad)
	propertyID := c.QueryInt("property_id")
	if propertyID > 0 {
		query = query.Where("property_id = ?", propertyID)
	}

	result := query.Delete(&models.Message{})

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error eliminando conversación",
		})
	}

	log.Printf("🗑️  Conversación eliminada: %d <-> %d (property: %d) (%d mensajes)", userID, otherUserID, propertyID, result.RowsAffected)

	return c.JSON(fiber.Map{
		"message":       "Conversación eliminada",
		"deleted_count": result.RowsAffected,
	})
}

// UploadMessageFile - Sube un archivo (imagen o documento) para usar en mensajes
// POST /api/messages/upload
func (h *MessageHandler) UploadMessageFile(c *fiber.Ctx) error {
	// Obtener archivo del formulario
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No se proporcionó ningún archivo",
		})
	}

	// Validar tamaño máximo (10MB)
	const maxFileSize = 10 * 1024 * 1024
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El archivo es demasiado grande. Máximo 10MB",
		})
	}

	// Determinar si es imagen o documento
	contentType := file.Header.Get("Content-Type")
	isImage := false
	if strings.HasPrefix(contentType, "image/") {
		isImage = true
	}

	// Validar tipos permitidos
	allowedExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".txt": true, ".zip": true,
	}
	ext := strings.ToLower(stdpath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tipo de archivo no permitido",
		})
	}

	// Crear directorio de uploads si no existe
	uploadDir := "uploads/mensajes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al crear directorio de uploads",
		})
	}

	// Generar nombre único
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("mensaje-%d-%d%s", timestamp, file.Size, ext)
	filepath := stdpath.Join(uploadDir, filename)

	// Guardar archivo
	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al guardar el archivo",
		})
	}

	// Generar URL
	fileURL := fmt.Sprintf("/api/mensajes/files/%s", filename)

	tipoMensaje := "documento"
	if isImage {
		tipoMensaje = "imagen"
	}

	log.Printf("📎 Archivo de mensaje subido: %s (%s)", filepath, tipoMensaje)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":      "Archivo subido exitosamente",
		"url":          fileURL,
		"filename":     filename,
		"original":     file.Filename,
		"tipo_mensaje": tipoMensaje,
		"size":         file.Size,
	})
}

// ServeMessageFile - Sirve un archivo de mensaje desde el sistema de archivos
// GET /api/mensajes/files/:filename
func (h *MessageHandler) ServeMessageFile(c *fiber.Ctx) error {
	filename := c.Params("filename")

	if strings.Contains(filename, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nombre de archivo inválido",
		})
	}

	filepath := stdpath.Join("uploads/mensajes", filename)

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Archivo no encontrado",
		})
	}

	return c.SendFile(filepath)
}
