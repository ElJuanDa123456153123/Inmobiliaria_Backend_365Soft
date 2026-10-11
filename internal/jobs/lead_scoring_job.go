package jobs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
)

// LeadScoringJob - Job para recalcular automáticamente los scores de leads
type LeadScoringJob struct {
	DB                  *config.Database
	LeadScoringService  *services.LeadScoringService
}

// NewLeadScoringJob - Crea un nuevo job de lead scoring
func NewLeadScoringJob() (*LeadScoringJob, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	// Inicializar repositorios
	clientRepo := repository.NewClientRepository(db.DB)
	interactionRepo := repository.NewInteractionRepository(db.DB)
	viewHistoryRepo := repository.NewViewHistoryRepository(db.DB)
	favoriteRepo := repository.NewFavoriteRepository(db.DB)

	// Inicializar servicio
	leadScoringService := services.NewLeadScoringService(
		db.DB,
		clientRepo,
		interactionRepo,
		viewHistoryRepo,
		favoriteRepo,
	)

	return &LeadScoringJob{
		DB:                 db,
		LeadScoringService: leadScoringService,
	}, nil
}

// RecalculateStaleScores - Recalcula los scores que están obsoletos (más de 7 días sin recalcular)
func (j *LeadScoringJob) RecalculateStaleScores() error {
	fmt.Println("🔄 Iniciando recálculo de lead scores obsoletos...")

	// Buscar clientes que tienen scores obsoletos
	var staleClients []struct {
		ClientID uint
		LastCalculatedAt time.Time
	}

	// Usar SQL directo para encontrar scores obsoletos
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	err := j.DB.DB.Raw(`
		SELECT client_id, last_calculated_at
		FROM lead_scores
		WHERE last_calculated_at < ? OR last_calculated_at IS NULL
	`, sevenDaysAgo).Scan(&staleClients).Error

	if err != nil {
		fmt.Printf("❌ Error buscando scores obsoletos: %v\n", err)
		return err
	}

	if len(staleClients) == 0 {
		fmt.Println("✅ No hay scores obsoletos para recalcular")
		return nil
	}

	fmt.Printf("📦 Encontrados %d scores obsoletos para recalcular\n", len(staleClients))

	successCount := 0
	for _, client := range staleClients {
		_, err := j.LeadScoringService.UpdateLeadScore(client.ClientID)
		if err != nil {
			fmt.Printf("⚠️  Error recalculando score para cliente %d: %v\n", client.ClientID, err)
			continue
		}
		successCount++
	}

	fmt.Printf("✅ Recálculo completado: %d/%d leads actualizados\n", successCount, len(staleClients))
	return nil
}

// RecalculateAllScores - Recalcula TODOS los scores de leads
func (j *LeadScoringJob) RecalculateAllScores() error {
	fmt.Println("🔄 Iniciando recálculo COMPLETO de todos los lead scores...")

	err := j.LeadScoringService.RecalculateAllScores()
	if err != nil {
		fmt.Printf("❌ Error recalculando todos los scores: %v\n", err)
		return err
	}

	return nil
}

// PrintScoreStats - Imprime estadísticas de scoring
func (j *LeadScoringJob) PrintScoreStats() error {
	stats, err := j.LeadScoringService.GetScoreStats(0) // 0 = todos los asesores
	if err != nil {
		fmt.Printf("❌ Error obteniendo estadísticas: %v\n", err)
		return err
	}

	fmt.Println("\n📊 ESTADÍSTICAS DE LEAD SCORING")
	fmt.Println("================================")
	fmt.Printf("Total de Leads: %d\n", stats.Total)
	fmt.Printf("Promedio de Score: %.1f\n", stats.AvgScore)
	fmt.Printf("Leads ALTO 🔥: %d\n", stats.HighCount)
	fmt.Printf("Leads MEDIO ⚡: %d\n", stats.MediumCount)
	fmt.Printf("Leads BAJO ❄️: %d\n", stats.LowCount)

	if stats.Total > 0 {
		highPercentage := float64(stats.HighCount) / float64(stats.Total) * 100
		mediumPercentage := float64(stats.MediumCount) / float64(stats.Total) * 100
		lowPercentage := float64(stats.LowCount) / float64(stats.Total) * 100

		fmt.Println("\nDistribución Porcentual:")
		fmt.Printf("  🔥 ALTO: %.1f%%\n", highPercentage)
		fmt.Printf("  ⚡ MEDIO: %.1f%%\n", mediumPercentage)
		fmt.Printf("  ❄️ BAJO: %.1f%%\n", lowPercentage)
	}

	fmt.Println("================================")
	return nil
}

// StartScheduler - Inicia el scheduler para recalcular automáticamente
func (j *LeadScoringJob) StartScheduler(intervalHours int) {
	fmt.Printf("⏰ Scheduler de Lead Scoring iniciado: Ejecutando cada %d horas\n", intervalHours)

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	j.runJob()

	for range ticker.C {
		j.runJob()
	}
}

// runJob - Ejecuta el job completo (recalcular obsoletos + imprimir stats)
func (j *LeadScoringJob) runJob() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("🔄 Lead Scoring Job - %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 60))

	// 1. Recalcular scores obsoletos
	if err := j.RecalculateStaleScores(); err != nil {
		log.Printf("❌ Error en recálculo de scores obsoletos: %v\n", err)
	}

	// 2. Imprimir estadísticas
	if err := j.PrintScoreStats(); err != nil {
		log.Printf("❌ Error imprimiendo estadísticas: %v\n", err)
	}

	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// StartSchedulerFromEnv - Inicia el scheduler leyendo configuración desde variables de entorno
func (j *LeadScoringJob) StartSchedulerFromEnv() {
	// Leer intervalo desde variable de entorno (default: 24 horas)
	intervalHours := 24
	if ih := os.Getenv("LEAD_SCORING_INTERVAL_HOURS"); ih != "" {
		if val, err := strconv.Atoi(ih); err == nil && val > 0 {
			intervalHours = val
		}
	}

	j.StartScheduler(intervalHours)
}
