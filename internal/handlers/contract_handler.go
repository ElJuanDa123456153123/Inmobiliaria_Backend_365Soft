package handlers

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ContractHandler - Handler para contratos de alquiler
type ContractHandler struct {
	DB *gorm.DB
}

// NewContractHandler - Crea un nuevo handler de contratos
func NewContractHandler() (*ContractHandler, error) {
	dbConfig, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}

	return &ContractHandler{
		DB: dbConfig.DB,
	}, nil
}

// CreateContractRequest - Estructura para crear contrato
type CreateContractRequest struct {
	PropertyID    uint    `json:"property_id" validate:"required"`
	InquilinoID   uint    `json:"inquilino_id" validate:"required"`
	FechaInicio   string  `json:"fecha_inicio" validate:"required"`
	FechaFin      string  `json:"fecha_fin" validate:"required"`
	MontoMensual  float64 `json:"monto_mensual" validate:"required"`
}

// CreateContract - Crea un nuevo contrato de alquiler
// POST /api/contracts
func (h *ContractHandler) CreateContract(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Parsear body
	var req CreateContractRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Validar
	if req.PropertyID == 0 || req.InquilinoID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "property_id e inquilino_id son requeridos",
		})
	}

	if req.FechaInicio == "" || req.FechaFin == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fecha_inicio y fecha_fin son requeridas",
		})
	}

	if req.MontoMensual <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "monto_mensual debe ser mayor a 0",
		})
	}

	// Parsear fechas
	fechaInicio, err := time.Parse(time.RFC3339, req.FechaInicio)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha_inicio inválido",
		})
	}

	fechaFin, err := time.Parse(time.RFC3339, req.FechaFin)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Formato de fecha_fin inválido",
		})
	}

	// Validar que fecha fin > fecha inicio
	if fechaFin.Before(fechaInicio) || fechaFin.Equal(fechaInicio) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fecha_fin debe ser posterior a fecha_inicio",
		})
	}

	// Verificar que la propiedad existe y pertenece al usuario (vendedor)
	var prop models.Property
	if err := h.DB.Where("id = ? AND user_id = ?", req.PropertyID, userID).First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Propiedad no encontrada o no tienes permisos",
		})
	}

	// Verificar que el inquilino existe
	var inquilino models.User
	if err := h.DB.Where("id = ?", req.InquilinoID).First(&inquilino).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Inquilino no encontrado",
		})
	}

	// Crear contrato
	contract := models.RentalContract{
		PropertyID:   req.PropertyID,
		InquilinoID:  req.InquilinoID,
		VendedorID:   userID,
		FechaInicio:  fechaInicio,
		FechaFin:     fechaFin,
		MontoMensual: req.MontoMensual,
		Estado:       models.ContractStatusActivo,
	}

	if err := h.DB.Create(&contract).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creando contrato",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&contract, contract.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Contrato creado exitosamente",
		"contrato": contract,
	})
}

// GetMyContracts - Obtiene contratos del usuario (como vendedor o inquilino)
// GET /api/contracts/my
func (h *ContractHandler) GetMyContracts(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Obtener parámetros de filtrado
	estado := c.Query("estado", "")
	rol := c.Query("rol", "") // "vendedor" o "inquilino"

	// Construir query
	query := h.DB.Where("vendedor_id = ? OR inquilino_id = ?", userID, userID)

	// Aplicar filtros
	if estado != "" {
		query = query.Where("estado = ?", estado)
	}

	if rol == "vendedor" {
		query = query.Where("vendedor_id = ?", userID)
	} else if rol == "inquilino" {
		query = query.Where("inquilino_id = ?", userID)
	}

	// Obtener contratos
	var contracts []models.RentalContract
	if err := query.Preload("Property").Preload("Inquilino").Preload("Vendedor").
		Order("created_at DESC").
		Find(&contracts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error obteniendo contratos",
		})
	}

	return c.JSON(fiber.Map{
		"contratos": contracts,
		"total":     len(contracts),
	})
}

// GetContract - Obtiene detalle de un contrato
// GET /api/contracts/:id
func (h *ContractHandler) GetContract(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Verificar que el contrato existe y el usuario tiene acceso
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND (vendedor_id = ? OR inquilino_id = ?)", contractID, userID, userID).
		Preload("Property").
		Preload("Inquilino").
		Preload("Vendedor").
		First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"contrato": contract,
	})
}

// UpdateContractRequest - Estructura para actualizar contrato
type UpdateContractRequest struct {
	FechaFin     *string  `json:"fecha_fin,omitempty"`
	MontoMensual *float64 `json:"monto_mensual,omitempty"`
	Estado       *string  `json:"estado,omitempty"`
	DocumentoURL *string  `json:"documento_url,omitempty"`
}

// UpdateContract - Actualiza un contrato
// PATCH /api/contracts/:id
func (h *ContractHandler) UpdateContract(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Parsear body
	var req UpdateContractRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Error parsing request",
		})
	}

	// Verificar que el contrato existe y pertenece al usuario (vendedor)
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND vendedor_id = ?", contractID, userID).First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado o no tienes permisos",
		})
	}

	// Actualizar campos si se proporcionan
	if req.FechaFin != nil {
		fechaFin, err := time.Parse(time.RFC3339, *req.FechaFin)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de fecha_fin inválido",
			})
		}
		contract.FechaFin = fechaFin
	}

	if req.MontoMensual != nil {
		if *req.MontoMensual <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "monto_mensual debe ser mayor a 0",
			})
		}
		contract.MontoMensual = *req.MontoMensual
	}

	if req.Estado != nil {
		// Validar estado
		if *req.Estado != string(models.ContractStatusActivo) &&
			*req.Estado != string(models.ContractStatusVencido) &&
			*req.Estado != string(models.ContractStatusTerminado) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Estado inválido (debe ser: activo, vencido, terminado)",
			})
		}
		contract.Estado = models.ContractStatus(*req.Estado)
	}

	if req.DocumentoURL != nil {
		contract.DocumentoURL = *req.DocumentoURL
	}

	if err := h.DB.Save(&contract).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error actualizando contrato",
		})
	}

	// Cargar relaciones
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&contract, contract.ID)

	return c.JSON(fiber.Map{
		"message":  "Contrato actualizado",
		"contrato": contract,
	})
}

// TerminateContract - El propietario termina el contrato (anticipadamente o al vencimiento).
// Genera el pago de devolución (anticrés y/o depósito) pendiente de confirmación por el inquilino
// POST /api/contracts/:id/terminate
func (h *ContractHandler) TerminateContract(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Verificar que el contrato existe y pertenece al propietario
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND vendedor_id = ? AND deleted_at IS NULL", contractID, userID).
		Preload("Property").
		Preload("Inquilino").
		First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado",
		})
	}

	if contract.Estado != models.ContractStatusActivo {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "El contrato ya no está activo (estado: " + string(contract.Estado) + ")",
		})
	}

	now := time.Now()

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Contrato → terminado
		if err := tx.Model(&models.RentalContract{}).Where("id = ?", contract.ID).
			Update("estado", models.ContractStatusTerminado).Error; err != nil {
			return err
		}

		// 2. Liberar la propiedad si no tiene otros contratos activos
		var activeCount int64
		if err := tx.Model(&models.RentalContract{}).
			Where("property_id = ? AND id != ? AND estado = ? AND deleted_at IS NULL",
				contract.PropertyID, contract.ID, models.ContractStatusActivo).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount == 0 {
			if err := tx.Model(&models.Property{}).Where("id = ?", contract.PropertyID).
				Update("alquilada", false).Error; err != nil {
				return err
			}
		}

		// 3. Generar devolución del anticrés (anticrético) si aún no existe
		if contract.TipoContrato == models.ContractTypeAnticretico {
			var devolucionCount int64
			tx.Model(&models.RentPayment{}).
				Where("contract_id = ? AND tipo = ? AND deleted_at IS NULL", contract.ID, models.PaymentTypeDevolucionAnticres).
				Count(&devolucionCount)
			if devolucionCount == 0 {
				// El monto a devolver es el del pago de anticrés registrado
				var anticres models.RentPayment
				montoDevolucion := 0.0
				if err := tx.Where("contract_id = ? AND tipo = ? AND deleted_at IS NULL", contract.ID, models.PaymentTypeAnticres).
					Order("created_at DESC").First(&anticres).Error; err == nil {
					montoDevolucion = anticres.Monto
				}
				devolucion := models.RentPayment{
					ContractID:         contract.ID,
					InquilinoID:        contract.InquilinoID,
					VendedorID:         contract.VendedorID,
					MesCorrespondiente: int(now.Month()),
					Anio:               now.Year(),
					Tipo:               models.PaymentTypeDevolucionAnticres,
					Monto:              montoDevolucion,
					FechaPago:          now,
					Estado:             models.PaymentStatusPendiente,
				}
				if err := tx.Create(&devolucion).Error; err != nil {
					return err
				}
			}
		}

		// 4. Generar devolución del depósito (alquiler) si aplica y aún no existe
		if contract.TipoContrato != models.ContractTypeAnticretico && contract.MontoDeposito > 0 {
			var devolucionCount int64
			tx.Model(&models.RentPayment{}).
				Where("contract_id = ? AND tipo = ? AND deleted_at IS NULL", contract.ID, models.PaymentTypeDevolucionDeposito).
				Count(&devolucionCount)
			if devolucionCount == 0 {
				devolucion := models.RentPayment{
					ContractID:         contract.ID,
					InquilinoID:        contract.InquilinoID,
					VendedorID:         contract.VendedorID,
					MesCorrespondiente: int(now.Month()),
					Anio:               now.Year(),
					Tipo:               models.PaymentTypeDevolucionDeposito,
					Monto:              contract.MontoDeposito,
					FechaPago:          now,
					Estado:             models.PaymentStatusPendiente,
				}
				if err := tx.Create(&devolucion).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error terminando el contrato: " + err.Error(),
		})
	}

	// 🔔 Notificar al inquilino
	mensaje := "El contrato de \"" + contract.Property.Titulo + "\" finalizó."
	if contract.TipoContrato == models.ContractTypeAnticretico {
		mensaje += " El propietario debe registrar la devolución de tu anticrés. Confírmala cuando la recibas"
	} else if contract.MontoDeposito > 0 {
		mensaje += " El propietario debe registrar la devolución de tu depósito. Confírmala cuando la recibas"
	}
	h.DB.Create(&models.Notification{
		UserID:     contract.InquilinoID,
		PropertyID: &contract.PropertyID,
		Tipo:       "contrato_finalizado",
		Titulo:     "Contrato finalizado",
		Mensaje:    mensaje,
		IsRead:     false,
	})

	// Cargar contrato actualizado
	h.DB.Preload("Property").Preload("Inquilino").Preload("Vendedor").First(&contract, contract.ID)

	return c.JSON(fiber.Map{
		"message":  "Contrato finalizado. Devolución registrada pendiente de confirmación",
		"contrato": contract,
	})
}

// GeneratePDF - Genera y descarga el PDF del contrato
// GET /api/contracts/:id/pdf
func (h *ContractHandler) GeneratePDF(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	contractID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID de contrato inválido",
		})
	}

	// Verificar que el contrato existe y el usuario tiene acceso
	var contract models.RentalContract
	if err := h.DB.Where("id = ? AND (vendedor_id = ? OR inquilino_id = ?)", contractID, userID, userID).
		Preload("Property").
		Preload("Inquilino").
		Preload("Vendedor").
		First(&contract).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Contrato no encontrado",
		})
	}

	pdfBytes, err := generarPDFContrato(&contract)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error generando PDF: " + err.Error(),
		})
	}

	filename := fmt.Sprintf("contrato-%d.pdf", contract.ID)
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return c.Send(pdfBytes)
}
