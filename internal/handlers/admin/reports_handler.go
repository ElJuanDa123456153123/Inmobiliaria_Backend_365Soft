package admin

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// ReportsHandler - Handler para reportes financieros y analíticos
type ReportsHandler struct {
	DB *config.Database
}

// NewReportsHandler - Crea una nueva instancia del handler de reportes
func NewReportsHandler() (*ReportsHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}
	return &ReportsHandler{DB: db}, nil
}

// IncomeReport - Estructura del reporte de ingresos
type IncomeReport struct {
	Period struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"period"`
	Summary struct {
		TotalIncome       float64 `json:"total_income"`
		PaymentCount      int64   `json:"payment_count"`
		AverageTicket     float64 `json:"average_ticket"`
		GrowthVsPrevious  float64 `json:"growth_vs_previous"`
	} `json:"summary"`
	ByPlan     map[string]PlanIncomeBreakdown `json:"by_plan"`
	ByMonth    []MonthIncome                  `json:"by_month"`
	ByAdvisor  []AdvisorIncomeBreakdown       `json:"by_advisor,omitempty"`
	Transactions []TransactionDetail         `json:"transactions,omitempty"`
}

// PlanIncomeBreakdown - Desglose por tipo de plan
type PlanIncomeBreakdown struct {
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}

// MonthIncome - Ingresos por mes
type MonthIncome struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

// AdvisorIncomeBreakdown - Ingresos por asesor
type AdvisorIncomeBreakdown struct {
	AdvisorID uint    `json:"advisor_id"`
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	Count     int64   `json:"count"`
}

// TransactionDetail - Detalle de transacción
type TransactionDetail struct {
	ID           uint    `json:"id"`
	Date         string  `json:"date"`
	ClientName   string  `json:"client_name"`
	PlanName     string  `json:"plan_name"`
	Amount       float64 `json:"amount"`
	PropertyID   *uint   `json:"property_id,omitempty"`
	PropertyTitle string `json:"property_title,omitempty"`
}

// GetIncomeReport - Obtiene reporte completo de ingresos
// GET /admin/reports/income?start_date=2026-01-01&end_date=2026-12-31&advisor_id=1&include_transactions=true
func (h *ReportsHandler) GetIncomeReport(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Extraer parámetros de consulta
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	advisorIDStr := c.Query("advisor_id")
	includeTransactions := c.QueryBool("include_transactions", false)

	// Establecer fechas por defecto (últimos 12 meses)
	now := time.Now()
	var startDate, endDate time.Time

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de start_date inválido. Use YYYY-MM-DD",
			})
		}
	} else {
		startDate = time.Date(now.Year(), now.Month()-11, 1, 0, 0, 0, 0, now.Location())
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Formato de end_date inválido. Use YYYY-MM-DD",
			})
		}
		// Incluir todo el día final
		endDate = endDate.Add(23 * time.Hour + 59 * time.Minute + 59 * time.Second)
	} else {
		endDate = now
	}

	// Verificar si es asesor
	advisor, isAdvisor := c.Locals("advisor").(*models.Advisor)

	// Filtrar por asesor si se especificó o si es asesor
	var advisorFilter *uint
	if advisorIDStr != "" {
		advisorID := c.QueryInt("advisor_id", 0)
		if advisorID > 0 {
			advisorIDUint := uint(advisorID)
			advisorFilter = &advisorIDUint
		}
	} else if isAdvisor && advisor != nil {
		advisorFilter = &advisor.ID
	}

	report := IncomeReport{}
	report.Period.Start = startDate.Format("2006-01-02")
	report.Period.End = endDate.Format("2006-01-02")

	// Base query para pagos pagados
	baseQuery := db.DB.Model(&models.QrPago{}).
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", startDate).
		Where("fecha_proceso <= ?", endDate)

	// Aplicar filtro de asesor si corresponde
	if advisorFilter != nil {
		baseQuery = baseQuery.
			Where("property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", *advisorFilter)
	}

	// 1. Calcular total de ingresos
	if err := baseQuery.Select("COALESCE(SUM(monto), 0)").Scan(&report.Summary.TotalIncome).Error; err != nil {
		fmt.Printf("❌ Error calculating total income: %v\n", err)
	}

	// 2. Contar pagos
	if err := baseQuery.Model(&models.QrPago{}).Count(&report.Summary.PaymentCount).Error; err != nil {
		fmt.Printf("❌ Error counting payments: %v\n", err)
	}

	// 3. Calcular ticket promedio
	if report.Summary.PaymentCount > 0 {
		report.Summary.AverageTicket = report.Summary.TotalIncome / float64(report.Summary.PaymentCount)
	}

	// 4. Calcular crecimiento vs mes anterior
	previousStartDate := startDate.AddDate(0, -1, 0)
	previousEndDate := startDate.Add(-time.Second)
	var previousMonthIncome float64

	previousQuery := db.DB.Model(&models.QrPago{}).
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", previousStartDate).
		Where("fecha_proceso <= ?", previousEndDate)

	if advisorFilter != nil {
		previousQuery = previousQuery.
			Where("property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", *advisorFilter)
	}

	if err := previousQuery.Select("COALESCE(SUM(monto), 0)").Scan(&previousMonthIncome).Error; err == nil {
		if previousMonthIncome > 0 {
			report.Summary.GrowthVsPrevious = ((report.Summary.TotalIncome - previousMonthIncome) / previousMonthIncome) * 100
		}
	}

	// 5. Desglose por tipo de plan
	type PlanIncomeResult struct {
		PlanName string  `gorm:"column:plan_name"`
		Amount   float64 `gorm:"column:amount"`
		Count    int64   `gorm:"column:count"`
	}

	var planResults []PlanIncomeResult
	planQuery := db.DB.Model(&models.QrPago{}).
		Select("p.nombre as plan_name, SUM(qp.monto) as amount, COUNT(*) as count").
		Joins("JOIN plans p ON p.id = qr_pagos.plan_id").
		Where("qr_pagos.estado = ?", "PAGADO").
		Where("qr_pagos.fecha_proceso >= ?", startDate).
		Where("qr_pagos.fecha_proceso <= ?", endDate)

	if advisorFilter != nil {
		planQuery = planQuery.
			Where("qr_pagos.property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", *advisorFilter)
	}

	if err := planQuery.Group("p.nombre").Scan(&planResults).Error; err == nil {
		report.ByPlan = make(map[string]PlanIncomeBreakdown)
		for _, pr := range planResults {
			report.ByPlan[pr.PlanName] = PlanIncomeBreakdown{
				Amount: pr.Amount,
				Count:  pr.Count,
			}
		}
	}

	// 6. Ingresos por mes
	var monthResults []MonthIncome
	monthQuery := db.DB.Model(&models.QrPago{}).
		Select("TO_CHAR(fecha_proceso, 'YYYY-MM') as month, SUM(monto) as amount").
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", startDate).
		Where("fecha_proceso <= ?", endDate)

	if advisorFilter != nil {
		monthQuery = monthQuery.
			Where("property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", *advisorFilter)
	}

	if err := monthQuery.
		Group("TO_CHAR(fecha_proceso, 'YYYY-MM')").
		Order("month ASC").
		Scan(&monthResults).Error; err == nil {
		report.ByMonth = monthResults
	}

	// 7. Ingresos por asesor (solo para admin)
	if !isAdvisor {
		type AdvisorResult struct {
			AdvisorID uint    `gorm:"column:advisor_id"`
			Name      string  `gorm:"column:name"`
			Amount    float64 `gorm:"column:amount"`
			Count     int64   `gorm:"column:count"`
		}

		var advisorResults []AdvisorResult
		advisorQuery := db.DB.Model(&models.QrPago{}).
			Select("a.id as advisor_id, a.nombre as name, SUM(qr_pagos.monto) as amount, COUNT(*) as count").
			Joins("JOIN properties p ON p.id = qr_pagos.property_id").
			Joins("JOIN advisors a ON a.id = p.advisor_id").
			Where("qr_pagos.estado = ?", "PAGADO").
			Where("qr_pagos.fecha_proceso >= ?", startDate).
			Where("qr_pagos.fecha_proceso <= ?", endDate)

		if advisorFilter != nil {
			advisorQuery = advisorQuery.Where("a.id = ?", *advisorFilter)
		}

		if err := advisorQuery.
			Group("a.id, a.nombre").
			Order("amount DESC").
			Scan(&advisorResults).Error; err == nil {
			for _, ar := range advisorResults {
				report.ByAdvisor = append(report.ByAdvisor, AdvisorIncomeBreakdown{
					AdvisorID: ar.AdvisorID,
					Name:      ar.Name,
					Amount:    ar.Amount,
					Count:     ar.Count,
				})
			}
		}
	}

	// 8. Transacciones detalladas (opcional)
	if includeTransactions {
		var transactions []TransactionDetail
		trxQuery := db.DB.Model(&models.QrPago{}).
			Select("qr_pagos.id, TO_CHAR(qr_pagos.fecha_proceso, 'YYYY-MM-DD') as date, "+
				"u.nombre as client_name, p.nombre as plan_name, qr_pagos.monto, "+
				"qr_pagos.property_id, prop.titulo as property_title").
			Joins("JOIN users u ON u.id = qr_pagos.user_id").
			Joins("JOIN plans p ON p.id = qr_pagos.plan_id").
			Joins("LEFT JOIN properties prop ON prop.id = qr_pagos.property_id").
			Where("qr_pagos.estado = ?", "PAGADO").
			Where("qr_pagos.fecha_proceso >= ?", startDate).
			Where("qr_pagos.fecha_proceso <= ?", endDate).
			Order("qr_pagos.fecha_proceso DESC")

		if advisorFilter != nil {
			trxQuery = trxQuery.
				Where("qr_pagos.property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", *advisorFilter)
		}

		if err := trxQuery.Scan(&transactions).Error; err == nil {
			report.Transactions = transactions
		}
	}

	return c.JSON(report)
}

// GetSubscriptionMetrics - Métricas de suscripciones
// GET /admin/reports/subscriptions
func (h *ReportsHandler) GetSubscriptionMetrics(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	now := time.Now()

	// Métricas de suscripciones
	var metrics struct {
		TotalSubscriptions     int64   `json:"total_subscriptions"`
		ActiveSubscriptions    int64   `json:"active_subscriptions"`
		MonthlyRecurringRevenue float64 `json:"monthly_recurring_revenue"`
		NewSubscriptionsThisMonth int64 `json:"new_subscriptions_this_month"`
		ChurnedSubscriptionsThisMonth int64 `json:"churned_subscriptions_this_month"`
		ByPlan                  map[string]int64 `json:"by_plan"`
	}

	// Total de suscripciones
	db.DB.Model(&models.Subscription{}).Count(&metrics.TotalSubscriptions)

	// Suscripciones activas
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	db.DB.Model(&models.Subscription{}).
		Where("estado = ?", "ACTIVA").
		Where("fecha_fin > ? OR fecha_fin IS NULL", now).
		Count(&metrics.ActiveSubscriptions)

	// MRR (Monthly Recurring Revenue)
	db.DB.Model(&models.Subscription{}).
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Where("subscriptions.estado = ?", "ACTIVA").
		Where("subscriptions.fecha_fin > ? OR subscriptions.fecha_fin IS NULL", now).
		Select("COALESCE(SUM(plans.precio), 0)").
		Scan(&metrics.MonthlyRecurringRevenue)

	// Nuevas suscripciones este mes
	db.DB.Model(&models.Subscription{}).
		Where("created_at >= ?", firstDayOfMonth).
		Count(&metrics.NewSubscriptionsThisMonth)

	// Suscripciones canceladas/vencidas este mes
	db.DB.Model(&models.Subscription{}).
		Where("updated_at >= ?", firstDayOfMonth).
		Where("estado IN ?", []string{"VENCIDA", "CANCELADA"}).
		Count(&metrics.ChurnedSubscriptionsThisMonth)

	// Desglose por plan
	type PlanCount struct {
		PlanName string `gorm:"column:plan_name"`
		Count    int64  `gorm:"column:count"`
	}

	var planCounts []PlanCount
	db.DB.Model(&models.Subscription{}).
		Select("p.nombre as plan_name, COUNT(*) as count").
		Joins("JOIN plans p ON p.id = subscriptions.plan_id").
		Where("subscriptions.estado = ?", "ACTIVA").
		Where("subscriptions.fecha_fin > ? OR subscriptions.fecha_fin IS NULL", now).
		Group("p.nombre").
		Scan(&planCounts)

	metrics.ByPlan = make(map[string]int64)
	for _, pc := range planCounts {
		metrics.ByPlan[pc.PlanName] = pc.Count
	}

	return c.JSON(metrics)
}

// GetFinancialOverview - Vista general financiera (para dashboard rápido)
// GET /admin/reports/overview
func (h *ReportsHandler) GetFinancialOverview(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	now := time.Now()

	// Verificar si es asesor
	advisor, isAdvisor := c.Locals("advisor").(*models.Advisor)

	var overview struct {
		IncomeCurrentMonth    float64 `json:"income_current_month"`
		IncomePreviousMonth  float64 `json:"income_previous_month"`
		GrowthPercent        float64 `json:"growth_percent"`
		PaymentCount         int64   `json:"payment_count"`
		AverageTicket        float64 `json:"average_ticket"`
	}

	// Ingresos mes actual
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	incomeQuery := db.DB.Model(&models.QrPago{}).
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", firstDayOfMonth)

	// Si es asesor, filtrar
	if isAdvisor && advisor != nil {
		incomeQuery = incomeQuery.
			Where("property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", advisor.ID)
	}

	if err := incomeQuery.Select("COALESCE(SUM(monto), 0)").Scan(&overview.IncomeCurrentMonth).Error; err != nil {
		fmt.Printf("❌ Error calculating income current month: %v\n", err)
	}

	// Ingresos mes anterior
	firstDayPreviousMonth := firstDayOfMonth.AddDate(0, -1, 0)
	lastDayPreviousMonth := firstDayOfMonth.Add(-time.Second)

	previousIncomeQuery := db.DB.Model(&models.QrPago{}).
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", firstDayPreviousMonth).
		Where("fecha_proceso <= ?", lastDayPreviousMonth)

	if isAdvisor && advisor != nil {
		previousIncomeQuery = previousIncomeQuery.
			Where("property_id IN (SELECT id FROM properties WHERE advisor_id = ?)", advisor.ID)
	}

	if err := previousIncomeQuery.Select("COALESCE(SUM(monto), 0)").Scan(&overview.IncomePreviousMonth).Error; err != nil {
		fmt.Printf("❌ Error calculating income previous month: %v\n", err)
	}

	// Crecimiento porcentual
	if overview.IncomePreviousMonth > 0 {
		overview.GrowthPercent = ((overview.IncomeCurrentMonth - overview.IncomePreviousMonth) / overview.IncomePreviousMonth) * 100
	}

	// Cantidad de pagos
	if err := incomeQuery.Model(&models.QrPago{}).Count(&overview.PaymentCount).Error; err != nil {
		fmt.Printf("❌ Error counting payments: %v\n", err)
	}

	// Ticket promedio
	if overview.PaymentCount > 0 {
		overview.AverageTicket = overview.IncomeCurrentMonth / float64(overview.PaymentCount)
	}

	return c.JSON(overview)
}
