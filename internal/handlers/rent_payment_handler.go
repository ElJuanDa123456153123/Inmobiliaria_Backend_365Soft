package handlers

import (
	"fmt"
	"strings"
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

	// Verificar si ya existe un pago (cuota) para este mes/año de este contrato
	var existingPayment models.RentPayment
	if err := h.DB.Where("contract_id = ? AND mes_correspondiente = ? AND anio = ? AND tipo = ?",
		contractID, req.MesCorrespondiente, req.Anio, models.PaymentTypeCuota).First(&existingPayment).Error; err == nil {
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

	// 🔔 Notificar al inquilino según el nuevo estado
	tipoNotif := "pago_confirmado"
	tituloNotif := "Pago confirmado"
	mensajeNotif := "El propietario confirmó tu pago"
	if req.Estado == string(models.PaymentStatusAtrasado) {
		tipoNotif = "pago_marco_atrasado"
		tituloNotif = "Pago marcado como atrasado"
		mensajeNotif = "El propietario marcó tu pago como atrasado"
	} else if req.Estado == string(models.PaymentStatusPendiente) {
		tipoNotif = "pago_rechazado"
		tituloNotif = "Comprobante rechazado"
		mensajeNotif = "El propietario rechazó tu comprobante. Vuelve a subirlo o coordina el pago"
	}
	var contract models.RentalContract
	if err := h.DB.Select("property_id").First(&contract, payment.ContractID).Error; err == nil {
		notif := models.Notification{
			UserID:     payment.InquilinoID,
			PropertyID: &contract.PropertyID,
			Tipo:       tipoNotif,
			Titulo:     tituloNotif,
			Mensaje:    mensajeNotif,
			IsRead:     false,
		}
		h.DB.Create(&notif)
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

// SubmitComprobanteRequest - Body para subir comprobante de una cuota
type SubmitComprobanteRequest struct {
	MetodoPago     string `json:"metodo_pago" validate:"required"` // qr, efectivo, transferencia
	ComprobanteURL string `json:"comprobante_url" validate:"required"`
}

// SubmitComprobante - El inquilino sube el comprobante de una cuota (queda pendiente de confirmación)
// POST /api/rent-payments/:id/comprobante
func (h *RentPaymentHandler) SubmitComprobante(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	var req SubmitComprobanteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parseando request",
		})
	}

	if strings.TrimSpace(req.MetodoPago) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "metodo_pago es requerido (qr, efectivo o transferencia)",
		})
	}
	if strings.TrimSpace(req.ComprobanteURL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "comprobante_url es requerido",
		})
	}

	// El pago debe pertenecer al inquilino autenticado
	var payment models.RentPayment
	if err := h.DB.Where("id = ? AND inquilino_id = ?", paymentID, userID).First(&payment).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pago no encontrado",
		})
	}

	if payment.Estado == models.PaymentStatusPagado {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Este pago ya fue confirmado",
		})
	}

	// Registrar el intento de pago: queda pendiente hasta que el dueño confirme
	payment.MetodoPago = strings.TrimSpace(req.MetodoPago)
	payment.ComprobanteURL = strings.TrimSpace(req.ComprobanteURL)
	payment.FechaPago = time.Now()

	if err := h.DB.Save(&payment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error guardando comprobante",
		})
	}

	// 🔔 Notificar al propietario que debe confirmar
	var contract models.RentalContract
	if err := h.DB.Select("property_id").First(&contract, payment.ContractID).Error; err == nil {
		notif := models.Notification{
			UserID:     payment.VendedorID,
			PropertyID: &contract.PropertyID,
			Tipo:       "comprobante_subido",
			Titulo:     "Nuevo comprobante de pago",
			Mensaje:    "Tu inquilino subió el comprobante de una cuota. Revísalo y confírmalo",
			IsRead:     false,
		}
		h.DB.Create(&notif)
	}

	return c.JSON(fiber.Map{
		"message": "Comprobante enviado. El propietario lo confirmará",
		"pago":    payment,
	})
}

// GetPanelTenants - Contratos del propietario con sus inquilinos y resumen de cuotas
// GET /api/panel/tenants
func (h *RentPaymentHandler) GetPanelTenants(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var contracts []models.RentalContract
	if err := h.DB.Where("vendedor_id = ? AND deleted_at IS NULL", userID).
		Preload("Property").
		Preload("Property.PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("Inquilino").
		Order("created_at DESC").
		Find(&contracts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo contratos",
		})
	}

	// Resumen de cuotas por contrato
	type TenantSummary struct {
		models.RentalContract
		CuotasPagadas    int64                 `json:"cuotas_pagadas"`
		CuotasPendientes int64                 `json:"cuotas_pendientes"`
		CuotasAtrasadas  int64                 `json:"cuotas_atrasadas"`
		TotalPagado      float64               `json:"total_pagado"`
		Pagos            []models.RentPayment  `json:"pagos"`
	}

	result := make([]TenantSummary, 0, len(contracts))
	for _, contract := range contracts {
		summary := TenantSummary{RentalContract: contract}

		h.DB.Model(&models.RentPayment{}).Where("contract_id = ? AND tipo = ? AND estado = ? AND deleted_at IS NULL",
			contract.ID, models.PaymentTypeCuota, models.PaymentStatusPagado).Count(&summary.CuotasPagadas)
		h.DB.Model(&models.RentPayment{}).Where("contract_id = ? AND tipo = ? AND estado = ? AND deleted_at IS NULL",
			contract.ID, models.PaymentTypeCuota, models.PaymentStatusPendiente).Count(&summary.CuotasPendientes)
		h.DB.Model(&models.RentPayment{}).Where("contract_id = ? AND tipo = ? AND estado = ? AND deleted_at IS NULL",
			contract.ID, models.PaymentTypeCuota, models.PaymentStatusAtrasado).Count(&summary.CuotasAtrasadas)

		type totalResult struct {
			Sum float64
		}
		var tr totalResult
		// Suma todo lo efectivamente pagado: cuotas, depósito y anticrés
		h.DB.Model(&models.RentPayment{}).
			Select("COALESCE(SUM(monto), 0) as sum").
			Where("contract_id = ? AND estado = ? AND tipo IN ? AND deleted_at IS NULL",
				contract.ID, models.PaymentStatusPagado,
				[]models.PaymentType{models.PaymentTypeCuota, models.PaymentTypeDeposito, models.PaymentTypeAnticres}).
			Scan(&tr)
		summary.TotalPagado = tr.Sum

		// Incluir el depósito junto con las cuotas en la lista de pagos
		h.DB.Where("contract_id = ? AND deleted_at IS NULL", contract.ID).
			Order("anio ASC, mes_correspondiente ASC, tipo ASC").
			Find(&summary.Pagos)

		result = append(result, summary)
	}

	return c.JSON(fiber.Map{
		"inquilinos": result,
		"total":      len(result),
	})
}

// ConfirmReceipt - El inquilino confirma que recibió la devolución (anticrés o depósito)
// POST /api/rent-payments/:id/confirm-receipt
func (h *RentPaymentHandler) ConfirmReceipt(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	paymentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de pago inválido",
		})
	}

	// El pago debe ser una devolución y pertenecer al inquilino autenticado
	var payment models.RentPayment
	if err := h.DB.Where("id = ? AND inquilino_id = ? AND tipo IN ?", paymentID, userID,
		[]models.PaymentType{models.PaymentTypeDevolucionAnticres, models.PaymentTypeDevolucionDeposito}).
		First(&payment).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Devolución no encontrada",
		})
	}

	if payment.Estado == models.PaymentStatusPagado {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Esta devolución ya fue confirmada",
		})
	}

	payment.Estado = models.PaymentStatusPagado
	payment.FechaPago = time.Now()

	if err := h.DB.Save(&payment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error confirmando la devolución",
		})
	}

	// 🔔 Notificar al propietario
	var contract models.RentalContract
	if err := h.DB.Select("property_id").First(&contract, payment.ContractID).Error; err == nil {
		tipoTexto := "del depósito"
		if payment.Tipo == models.PaymentTypeDevolucionAnticres {
			tipoTexto = "del anticrés"
		}
		h.DB.Create(&models.Notification{
			UserID:     payment.VendedorID,
			PropertyID: &contract.PropertyID,
			Tipo:       "devolucion_confirmada",
			Titulo:     "Devolución confirmada",
			Mensaje:    "El inquilino confirmó la recepción de la devolución " + tipoTexto,
			IsRead:     false,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Devolución confirmada",
		"pago":    payment,
	})
}
