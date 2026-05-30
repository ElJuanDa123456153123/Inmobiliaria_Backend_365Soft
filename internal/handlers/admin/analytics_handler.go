package admin

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// AnalyticsHandler - Handler para obtener estadísticas del sistema
type AnalyticsHandler struct {
	DB *config.Database
}

// NewAnalyticsHandler - Crea una nueva instancia del handler de analytics
func NewAnalyticsHandler() (*AnalyticsHandler, error) {
	db, err := config.GetDatabase()
	if err != nil {
		return nil, err
	}
	return &AnalyticsHandler{DB: db}, nil
}

// GetAnalytics - Retorna estadísticas completas
func (h *AnalyticsHandler) GetAnalytics(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	now := time.Now()

	// 1. Total usuarios registrados (rol = 'usuario')
	var totalUsers int64
	if err := db.DB.Model(&models.User{}).Where("rol = ? AND deleted_at IS NULL", models.RoleUsuario).Count(&totalUsers).Error; err != nil {
		fmt.Printf("❌ Error counting users: %v\n", err)
	}

	// 2. Total propiedades publicadas
	var totalProperties int64
	if err := db.DB.Model(&models.Property{}).Where("deleted_at IS NULL").Count(&totalProperties).Error; err != nil {
		fmt.Printf("❌ Error counting properties: %v\n", err)
	}

	// 3. Propiedades activas vs pendientes
	var activeProperties int64
	if err := db.DB.Model(&models.Property{}).Where("estado_propiedad = ? AND deleted_at IS NULL", models.PropertyStatusActivo).Count(&activeProperties).Error; err != nil {
		fmt.Printf("❌ Error counting active properties: %v\n", err)
	}

	var pendingProperties int64
	if err := db.DB.Model(&models.Property{}).Where("estado_propiedad = ? AND deleted_at IS NULL", models.PropertyStatusPendiente).Count(&pendingProperties).Error; err != nil {
		fmt.Printf("❌ Error counting pending properties: %v\n", err)
	}

	// 4. Ingresos del mes (mes actual, pagos completados)
	var monthlyIncome float64
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	if err := db.DB.Model(&models.Payment{}).
		Where("estado = ? AND fecha_pago >= ? AND fecha_pago <= ? AND deleted_at IS NULL", "completado", startOfMonth, endOfMonth).
		Select("COALESCE(SUM(monto), 0)").
		Row().Scan(&monthlyIncome); err != nil {
		fmt.Printf("❌ Error summing monthly income: %v\n", err)
	}

	// 5. Gráfico: Propiedades por día (últimos 30 días)
	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	dayCountsMap := make(map[string]int64)
	var dayList []string
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		dayCountsMap[dateStr] = 0
		dayList = append(dayList, dateStr)
	}

	thirtyDaysAgo := time.Date(now.Year(), now.Month(), now.Day()-29, 0, 0, 0, 0, now.Location())

	type dbDayCount struct {
		DateStr string
		Count   int64
	}
	var dbCounts []dbDayCount
	if err := db.DB.Model(&models.Property{}).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date_str, count(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", thirtyDaysAgo).
		Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
		Scan(&dbCounts).Error; err != nil {
		fmt.Printf("❌ Error grouping properties by day: %v\n", err)
	}

	for _, dbc := range dbCounts {
		if _, exists := dayCountsMap[dbc.DateStr]; exists {
			dayCountsMap[dbc.DateStr] = dbc.Count
		}
	}

	propertiesByDay := make([]DayCount, 0, 30)
	for _, dateStr := range dayList {
		propertiesByDay = append(propertiesByDay, DayCount{
			Date:  dateStr,
			Count: dayCountsMap[dateStr],
		})
	}

	// 6. Gráfico: Propiedades por tipo (casa, departamento, terreno, oficina, local)
	type TypeCount struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var dbTypeCounts []TypeCount
	if err := db.DB.Model(&models.Property{}).
		Select("tipo_propiedad as type, count(*) as count").
		Where("deleted_at IS NULL").
		Group("tipo_propiedad").
		Scan(&dbTypeCounts).Error; err != nil {
		fmt.Printf("❌ Error grouping properties by type: %v\n", err)
	}

	typeMap := map[string]int64{
		"casa":         0,
		"departamento": 0,
		"terreno":      0,
		"oficina":      0,
		"local":        0,
	}
	for _, tc := range dbTypeCounts {
		typeMap[tc.Type] = tc.Count
	}

	propertiesByType := []TypeCount{
		{Type: "casa", Count: typeMap["casa"]},
		{Type: "departamento", Count: typeMap["departamento"]},
		{Type: "terreno", Count: typeMap["terreno"]},
		{Type: "oficina", Count: typeMap["oficina"]},
		{Type: "local", Count: typeMap["local"]},
	}

	// 7. Gráfico: Ingresos por mes (últimos 12 meses)
	type MonthIncome struct {
		Month  string  `json:"month"`
		Amount float64 `json:"amount"`
	}

	monthIncomeMap := make(map[string]float64)
	var monthList []string
	for i := 11; i >= 0; i-- {
		d := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
		monthStr := d.Format("2006-01")
		monthIncomeMap[monthStr] = 0.0
		monthList = append(monthList, monthStr)
	}

	twelveMonthsAgo := time.Date(now.Year(), now.Month()-11, 1, 0, 0, 0, 0, now.Location())

	type dbMonthIncome struct {
		MonthStr string
		Amount   float64
	}
	var dbIncomes []dbMonthIncome
	if err := db.DB.Model(&models.Payment{}).
		Select("TO_CHAR(fecha_pago, 'YYYY-MM') as month_str, COALESCE(SUM(monto), 0) as amount").
		Where("estado = ? AND fecha_pago >= ? AND deleted_at IS NULL", "completado", twelveMonthsAgo).
		Group("TO_CHAR(fecha_pago, 'YYYY-MM')").
		Scan(&dbIncomes).Error; err != nil {
		fmt.Printf("❌ Error grouping payments by month: %v\n", err)
	}

	for _, dbi := range dbIncomes {
		if _, exists := monthIncomeMap[dbi.MonthStr]; exists {
			monthIncomeMap[dbi.MonthStr] = dbi.Amount
		}
	}

	incomeByMonth := make([]MonthIncome, 0, 12)
	for _, monthStr := range monthList {
		incomeByMonth = append(incomeByMonth, MonthIncome{
			Month:  monthStr,
			Amount: monthIncomeMap[monthStr],
		})
	}

	// 8. Tabla: Propiedades más vistas (las 10 con más vistas)
	var mostViewedProperties []models.Property
	if err := db.DB.
		Preload("PropertyImages").
		Order("vistas_contador DESC").
		Limit(10).
		Find(&mostViewedProperties).Error; err != nil {
		fmt.Printf("❌ Error fetching most viewed properties: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"total_users":            totalUsers,
		"total_properties":       totalProperties,
		"active_properties":      activeProperties,
		"pending_properties":     pendingProperties,
		"income_current_month":   monthlyIncome,
		"properties_by_day":      propertiesByDay,
		"properties_by_type":     propertiesByType,
		"income_by_month":        incomeByMonth,
		"most_viewed_properties": mostViewedProperties,
	})
}
