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

// GetAnalytics - Retorna estadísticas completas (filtradas por asesor si corresponde)
func (h *AnalyticsHandler) GetAnalytics(c *fiber.Ctx) error {
	db, err := config.GetDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error conectando a base de datos",
		})
	}

	// Verificar si es asesor
	advisor, isAdvisor := c.Locals("advisor").(*models.Advisor)

	now := time.Now()

	// Base query para propiedades (sin filtro de advisor por defecto)
	propertiesQuery := db.DB.Model(&models.Property{}).Where("deleted_at IS NULL")

	// Si es asesor, filtrar por advisor_id
	if isAdvisor && advisor != nil {
		propertiesQuery = propertiesQuery.Where("advisor_id = ?", advisor.ID)
	}

	// 1. Total propiedades asignadas (para asesores) o registradas (para admin)
	var totalProperties int64
	if err := propertiesQuery.Count(&totalProperties).Error; err != nil {
		fmt.Printf("❌ Error counting properties: %v\n", err)
	}

	// 2. Propiedades activas vs pendientes
	var activeProperties int64
	activeQuery := propertiesQuery.Where("estado_propiedad = ?", models.PropertyStatusActivo)
	if err := activeQuery.Count(&activeProperties).Error; err != nil {
		fmt.Printf("❌ Error counting active properties: %v\n", err)
	}

	var pendingProperties int64
	pendingQuery := propertiesQuery.Where("estado_propiedad = ?", models.PropertyStatusPendiente)
	if err := pendingQuery.Count(&pendingProperties).Error; err != nil {
		fmt.Printf("❌ Error counting pending properties: %v\n", err)
	}

	// 3. Total de usuarios (solo para admin)
	var totalUsers int64
	if !isAdvisor {
		if err := db.DB.Model(&models.User{}).Where("rol = ?", "usuario").Count(&totalUsers).Error; err != nil {
			fmt.Printf("❌ Error counting users: %v\n", err)
		}
	}

	// 4. Gráfico: Propiedades por día (últimos 30 días)
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
	dayQuery := db.DB.Model(&models.Property{}).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date_str, count(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", thirtyDaysAgo)

	// Si es asesor, filtrar por advisor_id
	if isAdvisor && advisor != nil {
		dayQuery = dayQuery.Where("advisor_id = ?", advisor.ID)
	}

	var dbCounts []dbDayCount
	if err := dayQuery.Group("TO_CHAR(created_at, 'YYYY-MM-DD')").Scan(&dbCounts).Error; err != nil {
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

	// 4. Gráfico: Propiedades por tipo
	type TypeCount struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var dbTypeCounts []TypeCount
	typeQuery := db.DB.Model(&models.Property{}).
		Select("tipo_propiedad as type, count(*) as count").
		Where("deleted_at IS NULL")

	// Si es asesor, filtrar por advisor_id
	if isAdvisor && advisor != nil {
		typeQuery = typeQuery.Where("advisor_id = ?", advisor.ID)
	}

	if err := typeQuery.Group("tipo_propiedad").Scan(&dbTypeCounts).Error; err != nil {
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

	// 5. Tabla: Propiedades más vistas (las 10 con más vistas del asesor)
	var mostViewedProperties []models.Property
	mostViewedQuery := db.DB.Model(&models.Property{}).
		Preload("PropertyImages")

	// Si es asesor, filtrar por advisor_id
	if isAdvisor && advisor != nil {
		mostViewedQuery = mostViewedQuery.Where("advisor_id = ?", advisor.ID)
	}

	if err := mostViewedQuery.Order("vistas_contador DESC").Limit(10).Find(&mostViewedProperties).Error; err != nil {
		fmt.Printf("❌ Error fetching most viewed properties: %v\n", err)
	}

	// 6. Ingresos del mes actual
	var incomeCurrentMonth float64
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	incomeQuery := db.DB.Model(&models.QrPago{}).
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", firstDayOfMonth)

	// Si es asesor, filtrar por sus propiedades
	if isAdvisor && advisor != nil {
		incomeQuery = incomeQuery.
			Where("property_id IN (SELECT id FROM propiedades WHERE advisor_id = ?)", advisor.ID)
	}

	if err := incomeQuery.Select("COALESCE(SUM(monto), 0)").Scan(&incomeCurrentMonth).Error; err != nil {
		fmt.Printf("❌ Error calculating income current month: %v\n", err)
	}

	// 7. Ingresos por mes (últimos 12 meses)
	type MonthIncome struct {
		Month  string  `json:"month"`
		Amount float64 `json:"amount"`
	}

	var incomesByMonth []MonthIncome

	twelveMonthsAgo := time.Date(now.Year(), now.Month()-11, 1, 0, 0, 0, 0, now.Location())

	incomeByMonthQuery := db.DB.Model(&models.QrPago{}).
		Select("TO_CHAR(fecha_proceso, 'YYYY-MM') as month, SUM(monto) as amount").
		Where("estado = ?", "PAGADO").
		Where("fecha_proceso >= ?", twelveMonthsAgo)

	// Si es asesor, filtrar
	if isAdvisor && advisor != nil {
		incomeByMonthQuery = incomeByMonthQuery.
			Where("property_id IN (SELECT id FROM propiedades WHERE advisor_id = ?)", advisor.ID)
	}

	if err := incomeByMonthQuery.
		Group("TO_CHAR(fecha_proceso, 'YYYY-MM')").
		Order("month ASC").
		Scan(&incomesByMonth).Error; err != nil {
		fmt.Printf("❌ Error calculating income by month: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"total_users":            totalUsers,
		"total_properties":       totalProperties,
		"active_properties":      activeProperties,
		"pending_properties":     pendingProperties,
		"properties_by_day":      propertiesByDay,
		"properties_by_type":     propertiesByType,
		"most_viewed_properties": mostViewedProperties,
		"income_current_month":   incomeCurrentMonth,
		"income_by_month":        incomesByMonth,
	})
}
