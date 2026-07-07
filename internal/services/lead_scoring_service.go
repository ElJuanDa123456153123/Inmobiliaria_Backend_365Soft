package services

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"gorm.io/gorm"
)

// LeadScoringService - Servicio de cálculo de scoring de leads
type LeadScoringService struct {
	db               *gorm.DB
	ClientRepo       repository.ClientRepositoryInterface
	InteractionRepo  repository.InteractionRepositoryInterface
	ViewHistoryRepo  repository.ViewHistoryRepositoryInterface
	FavoriteRepo     repository.FavoriteRepositoryInterface
}

// NewLeadScoringService - Crea un nuevo servicio de lead scoring
func NewLeadScoringService(
	db *gorm.DB,
	clientRepo repository.ClientRepositoryInterface,
	interactionRepo repository.InteractionRepositoryInterface,
	viewHistoryRepo repository.ViewHistoryRepositoryInterface,
	favoriteRepo repository.FavoriteRepositoryInterface,
) *LeadScoringService {
	return &LeadScoringService{
		db:               db,
		ClientRepo:       clientRepo,
		InteractionRepo:  interactionRepo,
		ViewHistoryRepo:  viewHistoryRepo,
		FavoriteRepo:     favoriteRepo,
	}
}

// LeadScoreData - Datos del scoring calculado
type LeadScoreData struct {
	Score                    int     `json:"score"`
	Level                    string  `json:"level"`
	HasBudget                bool    `json:"has_budget"`
	HasZone                  bool    `json:"has_zone"`
	RecentInteractionsCount  int     `json:"recent_interactions_count"`
	HasViewedProperties      bool    `json:"has_viewed_properties"`
	HasFavorites             bool    `json:"has_favorites"`
	HasSpecificProperty      bool    `json:"has_specific_property"`
	OriginBonus             int     `json:"origin_bonus"`
	StateBonus              int     `json:"state_bonus"`
	LastInteractionAt       *time.Time `json:"last_interaction_at,omitempty"`
	DaysSinceLastInteraction int    `json:"days_since_last_interaction"`
}

// CalculateLeadScore - Calcula el score de un lead basándose en múltiples factores
//
// Algoritmo de scoring (0-100 puntos):
// 1. Presupuesto definido (+20 puntos)
// 2. Zona específica (+15 puntos)
// 3. Interacciones en última semana (+20 puntos max, 5pts c/u)
// 4. Visto propiedades (+10 puntos)
// 5. Favoritos marcados (+10 puntos)
// 6. Propiedad específica de interés (+10 puntos)
// 7. Origen del lead (Referencia=+10, Web/WhatsApp=+5)
// 8. Estado del pipeline (Interesado+=+10, Contactado+=+5)
func (s *LeadScoringService) CalculateLeadScore(clientID uint) (*LeadScoreData, error) {
	// Obtener cliente con interacciones
	client, err := s.ClientRepo.GetWithInteractions(clientID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("Cliente no encontrado")
		}
		return nil, errors.DatabaseError(err)
	}

	scoreData := &LeadScoreData{
		Score: 0,
	}

	// 1. Presupuesto definido (+20 puntos)
	// NOTA: En una implementación completa, esto vendría de un campo de presupuesto
	// Por ahora, lo inferimos de si tiene una PropertyID específica
	scoreData.HasSpecificProperty = client.PropertyID != nil
	if scoreData.HasSpecificProperty {
		scoreData.Score += 10
	}

	// 2. Zona específica (+15 puntos)
	// NOTA: En una implementación completa, esto vendría de un campo de zona_preferida
	// Por ahora, lo inferimos de si tiene una PropertyID asociada
	scoreData.HasZone = client.PropertyID != nil
	if scoreData.HasZone {
		scoreData.Score += 15
	}

	// 3. Interacciones en última semana (+20 puntos max, 5pts c/u)
	recentInteractions, err := s.InteractionRepo.CountRecentByClient(clientID, 7)
	if err == nil {
		scoreData.RecentInteractionsCount = int(recentInteractions)
		interactionBonus := int(recentInteractions) * 5
		if interactionBonus > 20 {
			interactionBonus = 20
		}
		scoreData.Score += interactionBonus
	}

	// 4. Última interacción y días desde entonces
	lastInteraction, err := s.InteractionRepo.GetLastByClient(clientID)
	if err == nil && lastInteraction != nil {
		scoreData.LastInteractionAt = &lastInteraction.CreatedAt
		scoreData.DaysSinceLastInteraction = int(time.Since(lastInteraction.CreatedAt).Hours() / 24)

		// Aplicar decaimiento por inactividad (-5 puntos por cada 7 días sin contacto)
		if scoreData.DaysSinceLastInteraction > 7 {
			decay := (scoreData.DaysSinceLastInteraction / 7) * 5
			scoreData.Score -= decay
		}
	}

	// 5. Visto propiedades (+10 puntos)
	// NOTA: Esto requiere userID, no clientID
	// Por ahora, lo seteamos en false
	scoreData.HasViewedProperties = false
	// TODO: Implementar cuando se tenga relación Client <-> User

	// 6. Favoritos marcados (+10 puntos)
	// NOTA: Esto requiere userID, no clientID
	// Por ahora, lo seteamos en false
	scoreData.HasFavorites = false
	// TODO: Implementar cuando se tenga relación Client <-> User

	// 7. Origen del lead (Referencia=+10, Web/WhatsApp=+5)
	switch client.Origen {
	case models.ClientOriginReferencia:
		scoreData.OriginBonus = 10
	case models.ClientOriginWeb, models.ClientOriginWhatsApp, models.ClientOriginInstagram, models.ClientOriginFacebook:
		scoreData.OriginBonus = 5
	default:
		scoreData.OriginBonus = 0
	}
	scoreData.Score += scoreData.OriginBonus

	// 8. Estado del pipeline (Interesado+=+10, Contactado+=+5, EnNegociacion+=+15)
	switch client.Estado {
	case models.ClientStatusInteresado:
		scoreData.StateBonus = 10
	case models.ClientStatusEnNegociacion:
		scoreData.StateBonus = 15
	case models.ClientStatusContactado:
		scoreData.StateBonus = 5
	case models.ClientStatusNuevo:
		scoreData.StateBonus = 0
	default:
		scoreData.StateBonus = 0
	}
	scoreData.Score += scoreData.StateBonus

	// Asegurar que el score esté entre 0 y 100
	if scoreData.Score < 0 {
		scoreData.Score = 0
	}
	if scoreData.Score > 100 {
		scoreData.Score = 100
	}

	// Calcular nivel
	scoreData.Level = s.calculateLevel(scoreData.Score)

	fmt.Printf("📊 Lead Score calculado para cliente %d: %d pts (%s)\n",
		clientID, scoreData.Score, scoreData.Level)

	return scoreData, nil
}

// calculateLevel - Calcula el nivel basado en el score
func (s *LeadScoringService) calculateLevel(score int) string {
	if score >= 60 {
		return "ALTO"
	} else if score >= 30 {
		return "MEDIO"
	}
	return "BAJO"
}

// UpdateLeadScore - Actualiza o crea el score de un lead en la base de datos
func (s *LeadScoringService) UpdateLeadScore(clientID uint) (*models.LeadScore, error) {
	// Calcular score
	scoreData, err := s.CalculateLeadScore(clientID)
	if err != nil {
		return nil, err
	}

	// Buscar si ya existe un score para este cliente
	var existingScore models.LeadScore
	result := s.db.Where("client_id = ?", clientID).First(&existingScore)

	now := time.Now()

	if result.Error == nil {
		// Actualizar score existente
		existingScore.Score = scoreData.Score
		existingScore.Level = models.ScoreLevel(scoreData.Level)
		existingScore.HasBudget = scoreData.HasBudget
		existingScore.HasZone = scoreData.HasZone
		existingScore.RecentInteractionsCount = scoreData.RecentInteractionsCount
		existingScore.HasViewedProperties = scoreData.HasViewedProperties
		existingScore.HasFavorites = scoreData.HasFavorites
		existingScore.HasSpecificProperty = scoreData.HasSpecificProperty
		existingScore.OriginBonus = scoreData.OriginBonus
		existingScore.StateBonus = scoreData.StateBonus
		existingScore.LastInteractionAt = scoreData.LastInteractionAt
		existingScore.DaysSinceLastInteraction = scoreData.DaysSinceLastInteraction
		existingScore.LastCalculatedAt = &now

		if err := s.db.Save(&existingScore).Error; err != nil {
			return nil, errors.DatabaseError(err)
		}

		fmt.Printf("✅ Lead Score actualizado para cliente %d: %d pts (%s)\n",
			clientID, existingScore.Score, existingScore.Level)
		return &existingScore, nil
	}

	// Crear nuevo score
	if result.Error == gorm.ErrRecordNotFound {
		newScore := &models.LeadScore{
			ClientID:                 clientID,
			Score:                    scoreData.Score,
			Level:                    models.ScoreLevel(scoreData.Level),
			HasBudget:                scoreData.HasBudget,
			HasZone:                  scoreData.HasZone,
			RecentInteractionsCount:  scoreData.RecentInteractionsCount,
			HasViewedProperties:      scoreData.HasViewedProperties,
			HasFavorites:             scoreData.HasFavorites,
			HasSpecificProperty:      scoreData.HasSpecificProperty,
			OriginBonus:              scoreData.OriginBonus,
			StateBonus:               scoreData.StateBonus,
			LastInteractionAt:        scoreData.LastInteractionAt,
			DaysSinceLastInteraction: scoreData.DaysSinceLastInteraction,
			LastCalculatedAt:        &now,
		}

		if err := s.db.Create(newScore).Error; err != nil {
			return nil, errors.DatabaseError(err)
		}

		fmt.Printf("✅ Lead Score creado para cliente %d: %d pts (%s)\n",
			clientID, newScore.Score, newScore.Level)
		return newScore, nil
	}

	return nil, errors.DatabaseError(result.Error)
}

// GetLeadScore - Obtiene el score de un lead
func (s *LeadScoringService) GetLeadScore(clientID uint) (*models.LeadScore, error) {
	var leadScore models.LeadScore
	result := s.db.Where("client_id = ?", clientID).First(&leadScore)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Si no existe, calcular y crear
			return s.UpdateLeadScore(clientID)
		}
		return nil, errors.DatabaseError(result.Error)
	}

	// Verificar si está obsoleto (más de 7 días)
	if leadScore.IsStale() {
		// Recalcular
		return s.UpdateLeadScore(clientID)
	}

	return &leadScore, nil
}

// GetAllLeadScores - Obtiene todos los scores con opción de recálculo
func (s *LeadScoringService) GetAllLeadScores(recalculate bool) ([]models.LeadScore, error) {
	var scores []models.LeadScore

	if recalculate {
		// Recalcular todos los scores
		var clients []models.Client
		if err := s.db.Find(&clients).Error; err != nil {
			return nil, errors.DatabaseError(err)
		}

		scores = make([]models.LeadScore, 0, len(clients))
		for _, client := range clients {
			score, err := s.UpdateLeadScore(client.ID)
			if err != nil {
				fmt.Printf("⚠️  Error recalculando score para cliente %d: %v\n", client.ID, err)
				continue
			}
			scores = append(scores, *score)
		}

		fmt.Printf("✅ %d lead scores recalculados\n", len(scores))
	} else {
		// Obtener todos sin recalcular
		if err := s.db.Find(&scores).Error; err != nil {
			return nil, errors.DatabaseError(err)
		}
	}

	return scores, nil
}

// GetTopLeads - Obtiene los mejores leads ordenados por score
func (s *LeadScoringService) GetTopLeads(advisorID uint, limit int) ([]models.LeadScore, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var scores []models.LeadScore
	query := s.db.Order("score DESC").Limit(limit)

	if advisorID > 0 {
		// Filtrar por asesor
		query = query.Joins("JOIN clients ON clients.id = lead_scores.client_id").
			Where("clients.advisor_id = ?", advisorID)
	}

	if err := query.Find(&scores).Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("📊 Top %d leads obtenidos\n", len(scores))
	return scores, nil
}

// GetLeadsByLevel - Obtiene leads filtrados por nivel
func (s *LeadScoringService) GetLeadsByLevel(level models.ScoreLevel, advisorID uint) ([]models.LeadScore, error) {
	var scores []models.LeadScore
	query := s.db.Where("level = ?", level)

	if advisorID > 0 {
		// Filtrar por asesor
		query = query.Joins("JOIN clients ON clients.id = lead_scores.client_id").
			Where("clients.advisor_id = ?", advisorID)
	}

	if err := query.Find(&scores).Error; err != nil {
		return nil, errors.DatabaseError(err)
	}

	fmt.Printf("📊 %d leads con nivel %s\n", len(scores), level)
	return scores, nil
}

// RecalculateAllScores - Recalcula todos los scores de leads
func (s *LeadScoringService) RecalculateAllScores() error {
	fmt.Println("🔄 Iniciando recálculo de todos los lead scores...")

	var clients []models.Client
	if err := s.db.Find(&clients).Error; err != nil {
		return errors.DatabaseError(err)
	}

	successCount := 0
	for _, client := range clients {
		_, err := s.UpdateLeadScore(client.ID)
		if err != nil {
			fmt.Printf("⚠️  Error recalculando score para cliente %d: %v\n", client.ID, err)
			continue
		}
		successCount++
	}

	fmt.Printf("✅ Recálculo completado: %d/%d leads actualizados\n", successCount, len(clients))
	return nil
}

// GetScoreStats - Obtiene estadísticas de scoring
func (s *LeadScoringService) GetScoreStats(advisorID uint) (*ScoreStats, error) {
	// Estructura temporal para el mapeo de resultados
	var result struct {
		Total    int64   `json:"total"`
		AvgScore float64 `json:"avg_score"`
	}

	var err error

	// Usar SQL directo para tener control total
	if advisorID > 0 {
		// Query con filtro de asesor
		query := `
			SELECT COUNT(*) as total, AVG(ls.score) as avg_score
			FROM lead_scores ls
			JOIN clients c ON c.id = ls.client_id
			WHERE c.advisor_id = ?
		`
		err = s.db.Raw(query, advisorID).First(&result).Error
	} else {
		// Query sin filtro (admin ve todos)
		query := `
			SELECT COUNT(*) as total, AVG(score) as avg_score
			FROM lead_scores
		`
		err = s.db.Raw(query).First(&result).Error
	}

	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	// Contar por nivel
	var levelCounts []struct {
		Level string
		Count int64
	}

	if advisorID > 0 {
		// Query con filtro de asesor
		levelQuery := `
			SELECT ls.level, COUNT(*) as count
			FROM lead_scores ls
			JOIN clients c ON c.id = ls.client_id
			WHERE c.advisor_id = ?
			GROUP BY ls.level
		`
		err = s.db.Raw(levelQuery, advisorID).Scan(&levelCounts).Error
	} else {
		// Query sin filtro (admin ve todos)
		levelQuery := `
			SELECT level, COUNT(*) as count
			FROM lead_scores
			GROUP BY level
		`
		err = s.db.Raw(levelQuery).Scan(&levelCounts).Error
	}

	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	stats := &ScoreStats{
		Total:    result.Total,
		AvgScore: result.AvgScore,
	}

	stats.HighCount = 0
	stats.MediumCount = 0
	stats.LowCount = 0

	for _, lc := range levelCounts {
		switch lc.Level {
		case "ALTO":
			stats.HighCount = lc.Count
		case "MEDIO":
			stats.MediumCount = lc.Count
		case "BAJO":
			stats.LowCount = lc.Count
		}
	}

	return stats, nil
}

// ScoreStats - Estadísticas de scoring
type ScoreStats struct {
	Total       int64   `json:"total" gorm:"column:total"`
	AvgScore    float64 `json:"avg_score" gorm:"column:avg_score"`
	HighCount   int64   `json:"high_count"`
	MediumCount int64   `json:"medium_count"`
	LowCount    int64   `json:"low_count"`
}
