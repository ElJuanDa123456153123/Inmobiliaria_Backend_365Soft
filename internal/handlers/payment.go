package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PaymentHandler - Handler para pagos QR
type PaymentHandler struct {
	QrPagoService *services.QrPagoService
}

// NewPaymentHandler - Crea un nuevo handler de pagos
func NewPaymentHandler() (*PaymentHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	cfg := config.LoadConfig()

	qrPagoService := services.NewQrPagoService(db.DB, cfg)

	return &PaymentHandler{
		QrPagoService: qrPagoService,
	}, nil
}

// CreatePaymentRequest - Request para crear pago
type CreatePaymentRequest struct {
	PropertyID uint `json:"property_id"` // Opcional, puede ser 0
	PlanID     uint `json:"plan_id"`
}

// CreatePayment - Genera un QR para pagar la publicación de una propiedad
func (h *PaymentHandler) CreatePayment(c *fiber.Ctx) error {
	// Obtener user_id del JWT (del middleware de autenticación)
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Parsear request
	var req CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar que se haya enviado plan_id
	if req.PlanID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "plan_id es requerido",
		})
	}

	// Convertir user_id a uint
	userIDUint, ok := userID.(uint)
	if !ok {
		// Intentar convertir desde float64 (JSON numbers)
		if userIDFloat, ok := userID.(float64); ok {
			userIDUint = uint(userIDFloat)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error obteniendo user_id",
			})
		}
	}

	// Generar QR (property_id puede ser 0 si el usuario aún no tiene la propiedad)
	var propertyIDPtr *uint
	if req.PropertyID > 0 {
		propertyIDPtr = &req.PropertyID
	}

	params := services.GenerarQrParams{
		PropertyID: propertyIDPtr, // Puede ser nil
		UserID:     userIDUint,
		PlanID:     req.PlanID,
	}

	qrPago, err := h.QrPagoService.GenerarQr(c.Context(), params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(qrPago)
}

// GetPaymentStatus - Obtiene el estado de un pago QR
func (h *PaymentHandler) GetPaymentStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	qrPagoID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar estado
	status, err := h.QrPagoService.VerificarEstado(c.Context(), uint(qrPagoID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

// WebhookPayment - Webhook para recibir notificaciones del banco
func (h *PaymentHandler) WebhookPayment(c *fiber.Ctx) error {
	var payload services.BancoWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing webhook payload",
		})
	}

	// Procesar webhook
	respuesta, err := h.QrPagoService.ProcesarWebhook(c.Context(), payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(respuesta)
}

// CancelPayment - Cancela un pago QR no pagado
func (h *PaymentHandler) CancelPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	qrPagoID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar que el QR pertenezca al usuario autenticado
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Cancelar QR
	if err := h.QrPagoService.CancelarQr(uint(qrPagoID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "QR cancelado exitosamente",
	})
}

// GetMyPayments - Obtiene los pagos del usuario autenticado (pagos QR de planes de publicación)
func (h *PaymentHandler) GetMyPayments(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
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

	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a la base de datos",
		})
	}

	// Filtro opcional por estado (PENDIENTE, PAGADO, CANCELADO, EXPIRADO)
	estado := c.Query("estado", "")

	query := db.DB.Where("user_id = ?", userIDUint)
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	var pagos []models.QrPago
	if err := query.Preload("Plan").Preload("Property").Order("created_at DESC").Find(&pagos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo pagos",
		})
	}

	return c.JSON(fiber.Map{
		"payments": pagos,
		"total":    len(pagos),
	})
}

// GetMySubscription - Obtiene la suscripción del usuario autenticado
func (h *PaymentHandler) GetMySubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
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

	// Buscar suscripción del usuario con el plan cargado (la que está seleccionada)
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a la base de datos",
		})
	}

	var subscription struct {
		models.Subscription
		Plan *models.Plan `json:"plan"`
	}

	err = db.DB.
		Where("user_id = ? AND seleccionada = ?", userIDUint, true).
		Preload("Plan").
		First(&subscription).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No tiene suscripción - devolver 404
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No tienes una suscripción activa",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo suscripción",
		})
	}

	// Crear respuesta con el plan anidado dentro de la suscripción
	type SubscriptionResponse struct {
		models.Subscription
		Plan *models.Plan `json:"plan"`
	}

	response := SubscriptionResponse{
		Subscription: subscription.Subscription,
		Plan:         subscription.Plan,
	}

	return c.JSON(response)
}

// SimulatePaymentSuccess - SIMULA un pago exitoso (SOLO PARA DESARROLLO/PRUEBAS)
// Este endpoint simula que el banco envió el webhook confirmando el pago
func (h *PaymentHandler) SimulatePaymentSuccess(c *fiber.Ctx) error {
	// Obtener ID del pago
	id := c.Params("id")
	qrPagoID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Obtener usuario autenticado
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
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

	// Verificar que el QR pertenezca al usuario
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a la base de datos",
		})
	}

	var qrPago models.QrPago
	err = db.DB.First(&qrPago, uint(qrPagoID)).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	if qrPago.UserID != userIDUint {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Este pago no te pertenece",
		})
	}

	// Simular payload del webhook
	payload := services.BancoWebhookPayload{
		Alias:                qrPago.Alias,
		NumeroOrdenOriginante: "TEST-" + strconv.FormatUint(uint64(time.Now().Unix()), 10),
		NombreCliente:        "CLIENTE PRUEBA",
		DocumentoCliente:     "123456789",
		CuentaCliente:        "1234567890",
		Fechaproceso:         time.Now().Format("2006-01-02T15:04:05"),
	}

	// Procesar el pago usando el servicio
	respuesta, err := h.QrPagoService.ProcesarWebhook(c.Context(), payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error procesando pago: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Pago simulado exitosamente",
		"response": respuesta,
		"qr_pago_id": qrPagoID,
	})
}

// GetMySubscriptions - Obtiene TODAS las suscripciones del usuario autenticado
func (h *PaymentHandler) GetMySubscriptions(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
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

	// Buscar todas las suscripciones del usuario con el plan cargado
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a la base de datos",
		})
	}

	type SubscriptionWithPlan struct {
		models.Subscription
		Plan *models.Plan `json:"plan"`
	}

	var subscriptions []SubscriptionWithPlan

	err = db.DB.
		Where("user_id = ?", userIDUint).
		Preload("Plan").
		Order("seleccionada DESC, created_at DESC").
		Find(&subscriptions).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo suscripciones",
		})
	}

	return c.JSON(subscriptions)
}

// SelectSubscription - Selecciona una suscripción específica como activa
func (h *PaymentHandler) SelectSubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No autorizado",
		})
	}

	// Convertir user_id a uint
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

	// Obtener ID de la suscripción a seleccionar
	subscriptionID := c.Params("id")
	subscriptionIDUint, err := strconv.ParseUint(subscriptionID, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de suscripción inválido",
		})
	}

	// Conectar a la base de datos
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a la base de datos",
		})
	}

	// Verificar que la suscripción pertenezca al usuario
	var subscription models.Subscription
	err = db.DB.
		Where("id = ? AND user_id = ?", subscriptionIDUint, userIDUint).
		First(&subscription).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Suscripción no encontrada",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error buscando suscripción",
		})
	}

	// Verificar que la suscripción esté activa
	if subscription.Estado != "ACTIVA" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Solo puedes seleccionar suscripciones activas",
		})
	}

	// Verificar que la suscripción no esté vencida
	if subscription.EstaVencida() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes seleccionar una suscripción vencida",
		})
	}

	// Iniciar transacción
	tx := db.DB.Begin()

	// Desmarcar todas las suscripciones del usuario
	if err := tx.Model(&models.Subscription{}).
		Where("user_id = ?", userIDUint).
		Update("seleccionada", false).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error desmarcando suscripciones",
		})
	}

	// Marcar la suscripción seleccionada
	if err := tx.Model(&subscription).
		Update("seleccionada", true).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error seleccionando suscripción",
		})
	}

	// Confirmar transacción
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error confirmando cambios",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Suscripción seleccionada exitosamente",
		"subscription_id": subscriptionIDUint,
	})
}
