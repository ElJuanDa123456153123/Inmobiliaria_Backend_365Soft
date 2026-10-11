package jobs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"
)

// RecommendationJob - Job semanal del motor de recomendaciones (RF-260):
// precalcula y cachea las recomendaciones híbridas de los usuarios activos
// para que la consulta en pantalla sea instantánea.
type RecommendationJob struct {
	Redis *config.RedisClient
	DB    *config.Database
}

// NewRecommendationJob - Crea el job de recomendaciones
func NewRecommendationJob() (*RecommendationJob, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	redisClient, err := config.NewRedis()
	if err != nil {
		fmt.Println("⚠️  [RecommendationJob] Redis no disponible, el job operará sin caché")
		redisClient = nil
	}

	return &RecommendationJob{
		Redis: redisClient,
		DB:    db,
	}, nil
}

// RunOnce - Ejecuta una pasada: calcula y cachea recomendaciones de usuarios activos.
// Usuario activo = con favoritos, vistas o búsquedas guardadas registradas.
func (j *RecommendationJob) RunOnce() {
	if j.Redis == nil {
		return // sin caché no tiene sentido precalcular
	}

	// Usuarios con actividad reciente (últimos 90 días)
	cutoff := time.Now().AddDate(0, 0, -90)
	var userIDs []uint
	err := j.DB.DB.Raw(`
		SELECT DISTINCT user_id FROM (
			SELECT user_id, MAX(created_at) AS ultima FROM favorites WHERE deleted_at IS NULL GROUP BY user_id
			UNION ALL
			SELECT user_id, MAX(viewed_at) AS ultima FROM view_histories GROUP BY user_id
			UNION ALL
			SELECT user_id, MAX(updated_at) AS ultima FROM busquedas_guardadas WHERE deleted_at IS NULL GROUP BY user_id
		) actividad
		WHERE ultima >= ?
		LIMIT 1000
	`, cutoff).Scan(&userIDs).Error
	if err != nil {
		fmt.Printf("❌ [RecommendationJob] Error obteniendo usuarios activos: %v\n", err)
		return
	}

	if len(userIDs) == 0 {
		fmt.Println("✅ [RecommendationJob] No hay usuarios activos para actualizar")
		return
	}

	fmt.Printf("🔄 [RecommendationJob] Precalculando recomendaciones para %d usuarios activos...\n", len(userIDs))

	// Servicio de recomendaciones
	favoriteRepo := repository.NewFavoriteRepository(j.DB.DB)
	viewHistoryRepo := repository.NewViewHistoryRepository(j.DB.DB)
	savedSearchRepo := repository.NewSavedSearchRepository(j.DB.DB)
	service := services.NewRecommendationService(j.DB.DB, favoriteRepo, viewHistoryRepo, savedSearchRepo)

	updated, skipped := 0, 0
	for _, userID := range userIDs {
		recommendations, err := service.GetRecommendations(userID, 10)
		if err != nil {
			fmt.Printf("⚠️  [RecommendationJob] Usuario %d: %v\n", userID, err)
			skipped++
			continue
		}
		if len(recommendations) == 0 {
			skipped++
			continue
		}
		if err := j.Redis.CacheRecommendations(userID, recommendations); err != nil {
			fmt.Printf("⚠️  [RecommendationJob] Error cacheando usuario %d: %v\n", userID, err)
			skipped++
			continue
		}
		updated++
	}

	fmt.Printf("✅ [RecommendationJob] Recomendaciones actualizadas: %d usuarios (%d sin datos suficientes)\n", updated, skipped)
}

// StartScheduler - Inicia el scheduler semanal (default: cada 168 horas = 7 días).
// Intervalo configurable con RECOMMENDATION_INTERVAL_HOURS.
func (j *RecommendationJob) StartSchedulerFromEnv() {
	interval := 168 // default: semanal
	if v := os.Getenv("RECOMMENDATION_INTERVAL_HOURS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 1 {
			interval = parsed
		}
	}

	fmt.Printf("⏰ [RecommendationJob] Scheduler de recomendaciones iniciado: cada %d horas\n", interval)

	// Ejecutar inmediatamente al inicio
	j.RunOnce()

	ticker := time.NewTicker(time.Duration(interval) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		j.RunOnce()
	}
}
