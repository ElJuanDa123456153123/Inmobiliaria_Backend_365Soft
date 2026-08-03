package handlers

import (
	"encoding/base64"
	"fmt"
	"os"
	stdpath "path/filepath"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProofHandler - Handler para subir comprobantes de pago
type ProofHandler struct {
	DB *config.Database
}

// NewProofHandler - Crea un nuevo handler de comprobantes
func NewProofHandler() (*ProofHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &ProofHandler{DB: db}, nil
}

// UploadProofRequest - Solicitud para subir comprobante
type UploadProofRequest struct {
	QrPagoID uint   `json:"qr_pago_id"`
	Image    string `json:"image"` // Base64
}

// GetPendingProofsResponse - Respuesta con lista de comprobantes pendientes
type GetPendingProofsResponse struct {
	ID              uint        `json:"id"`
	CreatedAt       string      `json:"created_at"`
	Monto           float64     `json:"monto"`
	Estado          string      `json:"estado"`
	ComprobanteURL  string      `json:"comprobante_url"`
	User            UserInfo    `json:"user"`
	Plan            PlanInfo    `json:"plan"`
	Property        *PropertyInfo `json:"property,omitempty"`
}

type UserInfo struct {
	ID       uint   `json:"id"`
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Telefono string `json:"telefono"`
}

type PlanInfo struct {
	ID     uint    `json:"id"`
	Nombre string  `json:"nombre"`
	Precio float64 `json:"precio"`
}

type PropertyInfo struct {
	ID     uint   `json:"id"`
	Titulo string `json:"titulo"`
}

// GetPendingProofs - Lista todos los comprobantes pendientes (admin only)
// GET /api/admin/payments/pending
func (h *ProofHandler) GetPendingProofs(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Verificar que es admin
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil || user.Rol != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo administradores pueden ver comprobantes",
		})
	}

	// Buscar todos los pagos en estado EN_PROCESO
	var qrPagos []models.QrPago
	if err := db.DB.Preload("Plan").Preload("User").Where("estado = ?", "EN_PROCESO").Order("created_at DESC").Find(&qrPagos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo comprobantes",
		})
	}

	// Convertir a respuesta con información detallada
	response := make([]GetPendingProofsResponse, 0, len(qrPagos))
	for _, qr := range qrPagos {
		proof := GetPendingProofsResponse{
			ID:             qr.ID,
			CreatedAt:      qr.CreatedAt.Format("2006-01-02 15:04:05"),
			Monto:          qr.Monto,
			Estado:         qr.Estado,
			ComprobanteURL: qr.DetalleGlosa, // Guardamos la ruta aquí
			User: UserInfo{
				ID:       qr.User.ID,
				Nombre:   qr.User.Nombre,
				Email:    qr.User.Email,
				Telefono: qr.User.Telefono,
			},
		}

		// Info del plan
		if qr.Plan.ID > 0 {
			proof.Plan = PlanInfo{
				ID:     qr.Plan.ID,
				Nombre: qr.Plan.Nombre,
				Precio: qr.Plan.Precio,
			}
		}

		// Info de la propiedad (si existe)
		if qr.PropertyID != nil && *qr.PropertyID > 0 {
			var property models.Property
			if err := db.DB.Select("id, titulo").First(&property, *qr.PropertyID).Error; err == nil {
				proof.Property = &PropertyInfo{
					ID:     property.ID,
					Titulo: property.Titulo,
				}
			}
		}

		response = append(response, proof)
	}

	return c.JSON(response)
}

// GetPendingProofsCount - Retorna el contador de comprobantes pendientes (admin only)
// GET /api/admin/payments/pending/count
func (h *ProofHandler) GetPendingProofsCount(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Verificar que es admin
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil || user.Rol != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo administradores pueden ver el contador",
		})
	}

	// Contar pagos en estado EN_PROCESO
	var count int64
	db.DB.Model(&models.QrPago{}).Where("estado = ?", "EN_PROCESO").Count(&count)

	return c.JSON(fiber.Map{
		"count": count,
	})
}

// UploadProof - Sube un comprobante de pago (para métodos manuales)
// POST /api/payments/proof
func (h *ProofHandler) UploadProof(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Obtener user_id del JWT
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		if userIDFloat, ok := userID.(float64); ok {
			userIDUint = uint(userIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo user_id",
			})
		}
	}

	// Parsear request
	var req UploadProofRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar que se haya enviado qr_pago_id
	if req.QrPagoID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "qr_pago_id es requerido",
		})
	}

	// Validar que se haya enviado la imagen
	if req.Image == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "La imagen es requerida",
		})
	}

	// Verificar que el QR existe y pertenece al usuario
	var qrPago models.QrPago
	if err := db.DB.Where("id = ? AND user_id = ?", req.QrPagoID, userIDUint).First(&qrPago).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	// Verificar que el pago esté pendiente
	if qrPago.Estado != "PENDIENTE" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Este pago ya fue procesado",
		})
	}

	// Verificar que no haya expirado
	if time.Now().After(qrPago.FechaVencimiento) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Este pago ha expirado",
		})
	}

	// Decodificar imagen base64
	imageData, err := parseBase64Image(req.Image)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Error decodificando imagen: %v", err),
		})
	}

	// Generar nombre único para el archivo
	extParts := strings.Split(imageData.Mime, "/")
	ext := "jpg" // Default
	if len(extParts) == 2 {
		switch extParts[1] {
		case "png":
			ext = "png"
		case "webp":
			ext = "webp"
		case "jpeg", "jpg":
			ext = "jpg"
		}
	}
	filename := fmt.Sprintf("proof-%s-%d.%s", uuid.New().String()[:8], qrPago.ID, ext)

	// Directorio para guardar comprobantes
	uploadDir := "uploads/comprobantes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando directorio de uploads",
		})
	}

	// Ruta completa del archivo
	filepath := stdpath.Join(uploadDir, filename)

	// Guardar archivo
	if err := os.WriteFile(filepath, imageData.Data, 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error guardando archivo",
		})
	}

	fmt.Printf("✅ Comprobante guardado: %s\n", filepath)

	// Actualizar QR Pago con la imagen del comprobante
	qrPago.DetalleGlosa = filepath // Guardamos la ruta en detalle_glosa por ahora
	// También podríamos agregar un campo nuevo como "comprobante_url"

	// Cambiar estado a EN_PROCESO (manual review)
	qrPago.Estado = "EN_PROCESO"

	if err := db.DB.Save(&qrPago).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando pago",
		})
	}

	fmt.Printf("✅ Pago %d actualizado a estado EN_PROCESO (esperando aprobación admin)\n", qrPago.ID)

	// Crear notificación de comprobante recibido
	notification := models.Notification{
		UserID:      userIDUint,
		Tipo:        "pago_proceso",
		Titulo:      "Comprobante recibido",
		Mensaje:     fmt.Sprintf("Hemos recibido tu comprobante de pago de Bs. %.2f. Tu pago está siendo revisado. Te notificaremos cuando sea aprobado.", qrPago.Monto),
		IsRead:      false,
	}
	if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
		notification.PropertyID = qrPago.PropertyID
	}
	if err := db.DB.Create(&notification).Error; err != nil {
		fmt.Printf("⚠️  Error creando notificación: %v\n", err)
	} else {
		fmt.Printf("🔔 Notificación creada para usuario %d - Comprobante recibido\n", userIDUint)
	}

	return c.JSON(fiber.Map{
		"message": "Comprobante subido exitosamente",
		"qr_pago_id": qrPago.ID,
		"estado": qrPago.Estado,
		"comprobante_url": filepath,
	})
}

// ImageData - Datos de imagen decodificada
type ImageData struct {
	Data []byte
	Mime string
}

// parseBase64Image - Decodifica una imagen en base64
func parseBase64Image(data string) (*ImageData, error) {
	var mime string

	// Remover prefix si existe (ej: "data:image/png;base64,")
	parts := strings.SplitN(data, ",", 2)
	if len(parts) == 2 {
		// Extraer el mime type del prefix
		mimeParts := strings.Split(strings.TrimPrefix(parts[0], "data:"), ";")
		if len(mimeParts) > 0 {
			mime = mimeParts[0]
		}
		data = parts[1]
	} else {
		// Si no hay prefix, asumir jpeg por defecto
		mime = "image/jpeg"
	}

	// Decodificar base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("error decodificando base64: %w", err)
	}

	// Validar que tenga datos
	if len(decoded) == 0 {
		return nil, fmt.Errorf("la imagen decodificada está vacía")
	}

	return &ImageData{
		Data: decoded,
		Mime: mime,
	}, nil
}

// ReviewProofRequest - Solicitud para revisar comprobante (admin)
type ReviewProofRequest struct {
	Accion string `json:"accion"` // "aprobar" o "rechazar"
	Motivo  string `json:"motivo"` // Obligatorio si rechaza
}

// ReviewProof - Aprueba o rechaza un comprobante de pago (admin only)
// POST /api/admin/payments/:id/review
func (h *ProofHandler) ReviewProof(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Verificar que es admin
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil || user.Rol != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo administradores pueden revisar comprobantes",
		})
	}

	// Obtener qr_pago_id de la URL
	qrPagoID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Parsear request
	var req ReviewProofRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	// Validar acción
	if req.Accion != "aprobar" && req.Accion != "rechazar" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Acción inválida. Debe ser 'aprobar' o 'rechazar'",
		})
	}

	// Si rechaza, requiere motivo
	if req.Accion == "rechazar" && req.Motivo == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "El motivo es obligatorio al rechazar un comprobante",
		})
	}

	// Obtener QR Pago
	var qrPago models.QrPago
	if err := db.DB.Preload("Plan").First(&qrPago, qrPagoID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	// Verificar que esté en proceso
	if qrPago.Estado != "EN_PROCESO" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Este pago no está en estado de revisión",
		})
	}

	// Procesar según acción
	if req.Accion == "aprobar" {
		// Cambiar estado a PAGADO
		qrPago.Estado = "PAGADO"
		now := time.Now()
		qrPago.FechaProceso = &now

		if err := db.DB.Save(&qrPago).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error actualizando pago",
			})
		}

		fmt.Printf("✅ Pago %d aprobado por admin\n", qrPago.ID)

		// Procesar el pago: crear suscripción y activar propiedad
		// Usar transacción para mantener consistencia
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			// 1. Buscar si ya existe una suscripción activa del mismo plan para este usuario
			var existingSubscription models.Subscription
			err := tx.Where("user_id = ? AND plan_id = ? AND estado = ?",
				qrPago.UserID, qrPago.PlanID, "ACTIVA").First(&existingSubscription).Error

			if err == gorm.ErrRecordNotFound {
				// No existe suscripción, crear una nueva
				newSubscription := models.Subscription{
					UserID:    qrPago.UserID,
					PlanID:    qrPago.PlanID,
					Estado:    "ACTIVA",
					FechaInicio: time.Now(),
					PropiedadesUsadas: 1,
				}
				if err := tx.Create(&newSubscription).Error; err != nil {
					return fmt.Errorf("error creando suscripción: %w", err)
				}
				fmt.Printf("✅ Nueva suscripción creada para usuario %d\n", qrPago.UserID)
			} else {
				// Ya existe, acumular una propiedad más
				if err := tx.Model(&existingSubscription).Update("propiedades_usadas", gorm.Expr("propiedades_usadas + ?", 1)).Error; err != nil {
					return fmt.Errorf("error actualizando suscripción: %w", err)
				}
				fmt.Printf("✅ Suscripción actualizada para usuario %d (propiedades: %d)\n",
					qrPago.UserID, existingSubscription.PropiedadesUsadas+1)
			}

			// 2. Activar la propiedad si existe
			if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
				if err := tx.Model(&models.Property{}).Where("id = ?", *qrPago.PropertyID).
					Update("estado_propiedad", "activo").Error; err != nil {
					return fmt.Errorf("error activando propiedad: %w", err)
				}
				fmt.Printf("✅ Propiedad ID %d activada\n", *qrPago.PropertyID)
			}

			return nil
		})

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error procesando el pago: " + err.Error(),
			})
		}

		// Crear notificación de aprobación
		notification := models.Notification{
			UserID:      qrPago.UserID,
			Tipo:        "pago_confirmado",
			Titulo:      "¡Comprobante aprobado!",
			Mensaje:     fmt.Sprintf("Tu comprobante de pago de Bs. %.2f ha sido aprobado. Tu pago ha sido procesado exitosamente.", qrPago.Monto),
			IsRead:      false,
		}
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			notification.PropertyID = qrPago.PropertyID
		}
		db.DB.Create(&notification)
		fmt.Printf("🔔 Notificación de aprobación creada para usuario %d\n", qrPago.UserID)

		return c.JSON(fiber.Map{
			"message": "Comprobante aprobado exitosamente",
			"estado": "PAGADO",
		})

	} else {
		// Rechazar
		qrPago.Estado = "RECHAZADO"

		if err := db.DB.Save(&qrPago).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error actualizando pago",
			})
		}

		fmt.Printf("❌ Pago %d rechazado por admin. Motivo: %s\n", qrPago.ID, req.Motivo)

		// Crear notificación de rechazo
		notification := models.Notification{
			UserID:      qrPago.UserID,
			Tipo:        "pago_error",
			Titulo:      "Comprobante rechazado",
			Mensaje:     fmt.Sprintf("Tu comprobante de pago ha sido rechazado. Motivo: %s. Por favor, sube un nuevo comprobante o genera un nuevo QR.", req.Motivo),
			IsRead:      false,
		}
		if qrPago.PropertyID != nil && *qrPago.PropertyID > 0 {
			notification.PropertyID = qrPago.PropertyID
		}
		db.DB.Create(&notification)
		fmt.Printf("🔔 Notificación de rechazo creada para usuario %d\n", qrPago.UserID)

		return c.JSON(fiber.Map{
			"message": "Comprobante rechazado",
			"estado": "RECHAZADO",
		})
	}
}

// GetAllProofs - Lista todos los comprobantes con filtros (admin only)
// GET /api/admin/payments/all?estado={estado}&search={texto}&page={page}&limit={limit}
func (h *ProofHandler) GetAllProofs(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Verificar que es admin
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil || user.Rol != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Solo administradores pueden ver comprobantes",
		})
	}

	// Obtener parámetros de query
	estado := c.Query("estado", "")
	search := c.Query("search", "")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Construir query
	query := db.DB.Model(&models.QrPago{})

	// Filtro por estado
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	// Búsqueda por nombre de usuario, email, o título de propiedad
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Joins("LEFT JOIN users ON qr_pagos.user_id = users.id").
			Where("users.nombre ILIKE ? OR users.email ILIKE ?", searchTerm, searchTerm)
	}

	// Contar total
	var total int64
	query.Count(&total)

	// Paginar
	offset := (page - 1) * limit
	var qrPagos []models.QrPago

	if err := query.Preload("Plan").
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&qrPagos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo comprobantes",
		})
	}

	// Convertir a respuesta con información detallada
	response := make([]GetPendingProofsResponse, 0, len(qrPagos))
	for _, qr := range qrPagos {
		proof := GetPendingProofsResponse{
			ID:             qr.ID,
			CreatedAt:      qr.CreatedAt.Format("2006-01-02 15:04:05"),
			Monto:          qr.Monto,
			Estado:         qr.Estado,
			ComprobanteURL: qr.DetalleGlosa,
			User: UserInfo{
				ID:       qr.User.ID,
				Nombre:   qr.User.Nombre,
				Email:    qr.User.Email,
				Telefono: qr.User.Telefono,
			},
		}

		// Info del plan
		if qr.Plan.ID > 0 {
			proof.Plan = PlanInfo{
				ID:     qr.Plan.ID,
				Nombre: qr.Plan.Nombre,
				Precio: qr.Plan.Precio,
			}
		}

		// Info de la propiedad (si existe)
		if qr.PropertyID != nil && *qr.PropertyID > 0 {
			var property models.Property
			if err := db.DB.Select("id, titulo").First(&property, *qr.PropertyID).Error; err == nil {
				proof.Property = &PropertyInfo{
					ID:     property.ID,
					Titulo: property.Titulo,
				}
			}
		}

		response = append(response, proof)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"data":        response,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}
