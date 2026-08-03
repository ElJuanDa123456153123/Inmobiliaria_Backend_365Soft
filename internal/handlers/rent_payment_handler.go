package handlers

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RentPaymentHandler - Handler para pagos de alquiler
type RentPaymentHandler struct {
	DB *gorm.DB
}

// NewRentPaymentHandler - Crea un nuevo handler de pagos de alquiler
func NewRentPaymentHandler() (*RentPaymentHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &RentPaymentHandler{
		DB: dbConfig.DB,
	}, nil
}

// RegisterPaymentRequest - Estructura para registrar pago
type RegisterPaymentRequest struct {
	MesCorrespondiente int     `json:"mes_correspondiente" validate:"required,min=1,max=12"`
	Anio               int     `json:"anio" validate:"required"`
	Monto              float64 `json:"monto" validate:"required"`
	FechaPago          string  `json:"fecha_pago" validate:"required"`
	MetodoPago         string  `json:"metodo_pago" validate:"required"`
	ComprobanteURL     string  `json:"comprobante_url,omitempty"`
	Estado             string  `json:"estado,omitempty"`
}

// RegisterPayment - Registra un pago de alquiler
// POST /api/contracts/:id/payments
func (h *RentPaymentHandler) RegisterPayment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Parsear body
	var req RegisterPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar
	if req.MesCorrespondiente < 1 || req.MesCorrespondiente > 12 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "mes_correspondiente debe estar entre 1 y 12",
		})
	}

	if req.Anio < 2000 || req.Anio > 2100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "anio inválido",
		})
	}

	if req.Monto <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "monto debe ser mayor a 0",
		})
	}

	// Parsear fecha
	fechaPago, err := time.Parse(time.RFC3339, req.FechaPago)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha_pago inválido",
		})
	}

	// Verificar que el contrato existe y pertenece al usuario (vendedor o inquilino)
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND (vendedor_id = ? OR inquilino_id = ?)", contractID, userID, userID).First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado",
		})
	}

	// Validar estado (si se proporciona)
	estado := req.Estado
	if estado == "" {
		estado = string(models.PaymentStatusPagado)
	}

	if estado != string(models.PaymentStatusPendiente) &&
		estado != string(models.PaymentStatusPagado) &&
		estado != string(models.PaymentStatusAtrasado) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado inválido (debe ser: pendiente, pagado, atrasado)",
		})
	}

	// Verificar si ya existe un pago para este mes/año de este contrato
	var existingPayment models.RentPayment
	if err := h.DB.Where("contract_id = ? AND mes_correspondiente = ? AND anio = ?", contractID, req.MesCorrespondiente, req.Anio).First(&existingPayment).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ya existe un pago registrado para este período",
		})
	}

	// Crear pago
	payment := models.RentPayment{
		ContractID:        uint(contractID),
		InquilinoID:       contract.InquilinoID,
		VendedorID:        contract.VendedorID,
		MesCorrespondiente: req.MesCorrespondiente,
		Anio:               req.Anio,
		Monto:              req.Monto,
		FechaPago:          fechaPago,
		MetodoPago:         req.MetodoPago,
		Estado:             models.PaymentStatus(estado),
		ComprobanteURL:     req.ComprobanteURL,
	}

	if err := h.DB.Create(&payment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error registrando pago",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Contract").Preload("Inquilino").Preload("Vendedor").First(&payment, payment.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pago registrado exitosamente",
		"pago":    payment,
	})
}

// GetContractPayments - Obtiene pagos de un contrato
// GET /api/contracts/:id/payments
func (h *RentPaymentHandler) GetContractPayments(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Obtener parámetros de filtrado
	estado := c.Query("estado", "")
	anio := c.Query("anio", "")

	// Verificar que el contrato existe y el usuario tiene acceso
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND (vendedor_id = ? OR inquilino_id = ?)", contractID, userID, userID).First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado",
		})
	}

	// Construir query
	query := h.DB.Where("contract_id = ?", contractID)

	// Aplicar filtros
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	if anio != "" {
		var year int
		if _, err := fmt.Sscanf(anio, "%d", &year); err == nil {
			query = query.Where("anio = ?", year)
		}
	}

	// Obtener pagos
	var payments []models.RentPayment
	if err := query.Preload("Contract").Preload("Inquilino").Preload("Vendedor").
		Order("anio DESC, mes_correspondiente DESC").
		Find(&payments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo pagos",
		})
	}

	return c.JSON(fiber.Map{
		"pagos": payments,
		"total": len(payments),
	})
}

// UpdatePaymentStatusRequest - Estructura para actualizar estado de pago
type UpdatePaymentStatusRequest struct {
	Estado         string  `json:"estado" validate:"required"`
	ComprobanteURL *string `json:"comprobante_url,omitempty"`
}

// UpdatePaymentStatus - Actualiza el estado de un pago
// PATCH /api/payments/:id
func (h *RentPaymentHandler) UpdatePaymentStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	// Parsear body
	var req UpdatePaymentStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar estado
	if req.Estado != string(models.PaymentStatusPendiente) &&
		req.Estado != string(models.PaymentStatusPagado) &&
		req.Estado != string(models.PaymentStatusAtrasado) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Estado inválido (debe ser: pendiente, pagado, atrasado)",
		})
	}

	// Verificar que el pago existe y el usuario tiene acceso (vendedor del contrato)
	var payment models.RentPayment
	if err := h.DB.Where("id = ? AND vendedor_id = ?", paymentID, userID).First(&payment).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	// Actualizar
	payment.Estado = models.PaymentStatus(req.Estado)
	if req.ComprobanteURL != nil {
		payment.ComprobanteURL = *req.ComprobanteURL
	}

	if err := h.DB.Save(&payment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando pago",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Contract").Preload("Inquilino").Preload("Vendedor").First(&payment, payment.ID)

	return c.JSON(fiber.Map{
		"message": "Estado de pago actualizado",
		"pago":    payment,
	})
}

// GetPaymentHistory - Obtiene historial de todos los pagos del usuario
// GET /api/payments/history
func (h *RentPaymentHandler) GetPaymentHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de filtrado
	estado := c.Query("estado", "")
	anio := c.Query("anio", "")
	rol := c.Query("rol", "") // "vendedor" o "inquilino"

	// Construir query
	query := h.DB.Where("vendedor_id = ? OR inquilino_id = ?", userID, userID)

	// Aplicar filtros
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	if anio != "" {
		var year int
		if _, err := fmt.Sscanf(anio, "%d", &year); err == nil {
			query = query.Where("anio = ?", year)
		}
	}

	if rol == "vendedor" {
		query = query.Where("vendedor_id = ?", userID)
	} else if rol == "inquilino" {
		query = query.Where("inquilino_id = ?", userID)
	}

	// Obtener pagos
	var payments []models.RentPayment
	if err := query.Preload("Contract").Preload("Contract.Property").Preload("Inquilino").Preload("Vendedor").
		Order("anio DESC, mes_correspondiente DESC").
		Find(&payments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo historial de pagos",
		})
	}

	return c.JSON(fiber.Map{
		"pagos": payments,
		"total": len(payments),
	})
}
