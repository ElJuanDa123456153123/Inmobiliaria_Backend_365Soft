package services

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"gorm.io/gorm"
)

// Pesos del algoritmo híbrido de recomendaciones
const (
	// Peso del score colaborativo vs. contenido en la combinación final
	pesoContenido     = 0.6
	pesoColaborativo  = 0.4
	// Señal máxima: marcar como favorito pesa el doble que una vista
	pesoFavorito      = 2.0
	pesoVista         = 1.0
	pesoBusqueda      = 1.5
	// Máximo de usuarios candidatos a comparar (performance)
	maxUsuariosCandidatos = 300
)

// RecommendationService - Servicio de recomendaciones de propiedades
// Algoritmo híbrido: filtrado por contenido (perfil de preferencias ponderado)
// + filtrado colaborativo (similitud Jaccard entre usuarios)
type RecommendationService struct {
	db              *gorm.DB
	FavoriteRepo    repository.FavoriteRepositoryInterface
	ViewHistoryRepo  repository.ViewHistoryRepositoryInterface
	SavedSearchRepo repository.SavedSearchRepositoryInterface
}

// NewRecommendationService - Crea un nuevo servicio de recomendaciones
func NewRecommendationService(
	db *gorm.DB,
	favoriteRepo repository.FavoriteRepositoryInterface,
	viewHistoryRepo repository.ViewHistoryRepositoryInterface,
	savedSearchRepo repository.SavedSearchRepositoryInterface,
) *RecommendationService {
	return &RecommendationService{
		db:              db,
		FavoriteRepo:    favoriteRepo,
		ViewHistoryRepo:  viewHistoryRepo,
		SavedSearchRepo: savedSearchRepo,
	}
}

// UserProfile - Perfil de preferencias ponderado del usuario.
// Cada dimensión acumula pesos: ver una zona suma pesoVista, favoritarla pesoFavorito,
// una búsqueda guardada pesoBusqueda. El score de cada factor se normaliza contra pesoFavorito.
type UserProfile struct {
	UserID               uint
	ZoneWeights          map[string]float64
	PropertyTypeWeights  map[models.PropertyType]float64
	OperationTypeWeights map[models.OperationType]float64
	DormitoriosWeights   map[string]float64
	BanosWeights         map[string]float64
	PriceWeightedSum     float64
	PriceWeightTotal     float64
	AveragePriceViewed   float64
}

// NewUserProfile - Crea un perfil vacío
func NewUserProfile(userID uint) *UserProfile {
	return &UserProfile{
		UserID:               userID,
		ZoneWeights:          make(map[string]float64),
		PropertyTypeWeights:  make(map[models.PropertyType]float64),
		OperationTypeWeights: make(map[models.OperationType]float64),
		DormitoriosWeights:   make(map[string]float64),
		BanosWeights:         make(map[string]float64),
	}
}

// PriceRange - Rango de precios preferido
type PriceRange struct {
	Min float64
	Max float64
}

// PropertyRecommendation - Propiedad recomendada con score
type PropertyRecommendation struct {
	Property     models.Property `json:"property"`
	Score        float64         `json:"score"`
	Reason       string          `json:"reason"`
	MatchFactors []string        `json:"match_factors"`
}

// GetRecommendations - Genera recomendaciones híbridas (contenido + colaborativo)
// para el usuario, combinando ambos scores con los pesos definidos.
func (s *RecommendationService) GetRecommendations(userID uint, limit int) ([]PropertyRecommendation, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// 1. Construir perfil ponderado del usuario
	profile := s.buildUserProfile(userID)

	fmt.Printf("📊 Perfil de usuario %d: %d zonas, %d tipos propiedad, %d tipos operación\n",
		userID, len(profile.ZoneWeights), len(profile.PropertyTypeWeights), len(profile.OperationTypeWeights))

	// 2. Propiedades activas que el usuario aún no vio ni marcó
	var properties []models.Property
	err := s.db.Where("estado_propiedad = ?", models.PropertyStatusActivo).
		Where("id NOT IN (SELECT property_id FROM view_histories WHERE user_id = ?)", userID).
		Where("id NOT IN (SELECT property_id FROM favorites WHERE user_id = ?)", userID).
		Find(&properties).Error
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	if len(properties) == 0 {
		fmt.Println("⚠️  No hay propiedades disponibles para recomendar")
		return []PropertyRecommendation{}, nil
	}

	// 3. Score colaborativo (Jaccard) para estas propiedades
	collaborativeScores, err := s.getCollaborativeScores(userID)
	if err != nil {
		// No bloquear el servicio si falla el colaborativo
		fmt.Printf("⚠️  Filtrado colaborativo no disponible: %v\n", err)
		collaborativeScores = map[uint]float64{}
	}

	fmt.Printf("🔍 Evaluando %d propiedades (contenido + %d scores colaborativos)...\n",
		len(properties), len(collaborativeScores))

	// 4. Score híbrido por propiedad
	recommendations := make([]PropertyRecommendation, 0)
	for _, property := range properties {
		contentScore, reason, factors := s.calculatePropertyScore(property, profile)

		collabScore := collaborativeScores[property.ID]
		if collabScore > 0 {
			factors = append(factors, "Usuarios con gustos similares")
			if reason == "Propiedad recomendada basada en tu historial" {
				reason = "A usuarios con gustos similares también les interesó"
			}
		}

		finalScore := pesoContenido*contentScore + pesoColaborativo*collabScore

		// Umbral mínimo de calidad (relajado si hay señal colaborativa fuerte)
		umbral := 0.3
		if collabScore >= 0.5 {
			umbral = 0.2
		}
		if finalScore >= umbral {
			recommendations = append(recommendations, PropertyRecommendation{
				Property:     property,
				Score:        finalScore,
				Reason:       reason,
				MatchFactors: factors,
			})
		}
	}

	// 5. Ordenar por score descendente
	sortRecommendations(recommendations)

	// 6. Limitar resultados
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	fmt.Printf("✅ %d recomendaciones híbridas generadas para usuario %d\n", len(recommendations), userID)
	return recommendations, nil
}

// GetContentBasedRecommendations - Alias de compatibilidad (usa el algoritmo híbrido)
func (s *RecommendationService) GetContentBasedRecommendations(userID uint, limit int) ([]PropertyRecommendation, error) {
	return s.GetRecommendations(userID, limit)
}

// buildUserProfile - Construye el perfil de preferencias ponderado del usuario
func (s *RecommendationService) buildUserProfile(userID uint) *UserProfile {
	profile := NewUserProfile(userID)

	// 1. Favoritos (peso alto: señal de intención explícita)
	favorites, err := s.FavoriteRepo.GetByUser(userID)
	if err == nil && len(favorites) > 0 {
		fmt.Printf("📌 Analizando %d favoritos...\n", len(favorites))
		for _, fav := range favorites {
			if fav.Property.ID != 0 {
				profile.addPropertyPreferences(fav.Property, pesoFavorito)
			}
		}
	}

	// 2. Historial de vistas (peso medio)
	viewHistory, err := s.ViewHistoryRepo.GetByUser(userID)
	if err == nil && len(viewHistory) > 0 {
		fmt.Printf("👁️  Analizando %d vistas...\n", len(viewHistory))
		for _, view := range viewHistory {
			if view.Property.ID != 0 {
				profile.addPropertyPreferences(view.Property, pesoVista)
			}
		}
	}

	// 3. Búsquedas guardadas (peso medio-alto: lo que el usuario pide explícitamente)
	savedSearches, err := s.SavedSearchRepo.GetActiveByUser(userID)
	if err == nil && len(savedSearches) > 0 {
		fmt.Printf("🔍 Analizando %d búsquedas guardadas...\n", len(savedSearches))
		for _, search := range savedSearches {
			profile.addSearchPreferences(json.RawMessage(search.Filtros), pesoBusqueda)
		}
	}

	// 4. Precio promedio ponderado → rango preferido (+-30%)
	if profile.PriceWeightTotal > 0 {
		profile.AveragePriceViewed = profile.PriceWeightedSum / profile.PriceWeightTotal
	}

	return profile
}

// addPropertyPreferences - Acumula los pesos de una propiedad en cada dimensión del perfil
func (p *UserProfile) addPropertyPreferences(property models.Property, weight float64) {
	if property.Zona != "" {
		p.ZoneWeights[property.Zona] += weight
	}
	if property.TipoPropiedad != "" {
		p.PropertyTypeWeights[property.TipoPropiedad] += weight
	}
	if property.TipoOperacion != "" {
		p.OperationTypeWeights[property.TipoOperacion] += weight
	}
	if property.Dormitorios != "" && property.Dormitorios != "sin-definir" {
		p.DormitoriosWeights[property.Dormitorios] += weight
	}
	if property.Banos != "" && property.Banos != "sin-definir" {
		p.BanosWeights[property.Banos] += weight
	}
	// Promedio ponderado de precios
	if property.Precio > 0 {
		p.PriceWeightedSum += property.Precio * weight
		p.PriceWeightTotal += weight
	}
}

// addSearchPreferences - Acumula pesos desde los filtros de una búsqueda guardada
func (p *UserProfile) addSearchPreferences(filtrosJSON json.RawMessage, weight float64) {
	var filtros map[string]interface{}
	if err := json.Unmarshal(filtrosJSON, &filtros); err != nil {
		return
	}

	if zona, ok := filtros["zona"].(string); ok && zona != "" {
		p.ZoneWeights[zona] += weight
	}
	if tipoProp, ok := filtros["tipo_propiedad"].(string); ok && tipoProp != "" {
		p.PropertyTypeWeights[models.PropertyType(tipoProp)] += weight
	}
	if tipoOp, ok := filtros["tipo_operacion"].(string); ok && tipoOp != "" {
		p.OperationTypeWeights[models.OperationType(tipoOp)] += weight
	}
}

// calculatePropertyScore - Score por contenido: cada factor aporta proporcionalmente
// a la fuerza de la preferencia acumulada (normalizada contra pesoFavorito = señal máxima)
func (s *RecommendationService) calculatePropertyScore(property models.Property, profile *UserProfile) (float64, string, []string) {
	score := 0.0
	factors := make([]string, 0)
	reason := "Propiedad recomendada basada en tu historial"

	// 1. Zona (30% del peso) — proporcional a cuánta señal hay de esa zona
	if w := profile.ZoneWeights[property.Zona]; w > 0 {
		aporte := 0.30 * math.Min(1.0, w/pesoFavorito)
		score += aporte
		factors = append(factors, "Zona preferida")
	}

	// 2. Tipo de propiedad (15%)
	if w := profile.PropertyTypeWeights[property.TipoPropiedad]; w > 0 {
		score += 0.15 * math.Min(1.0, w/pesoFavorito)
		factors = append(factors, "Tipo de propiedad")
	}

	// 3. Tipo de operación (15%)
	if w := profile.OperationTypeWeights[property.TipoOperacion]; w > 0 {
		score += 0.15 * math.Min(1.0, w/pesoFavorito)
		factors = append(factors, "Tipo de operación")
	}

	// 4. Precio (20%) — contra el promedio ponderado del historial
	if profile.AveragePriceViewed > 0 && property.Precio > 0 {
		priceDiff := math.Abs(property.Precio-profile.AveragePriceViewed) / profile.AveragePriceViewed
		if priceDiff <= 0.3 {
			score += 0.2
			factors = append(factors, "Precio acorde a tu historial")
		} else if priceDiff <= 0.5 {
			score += 0.1
		}
	}

	// 5. Dormitorios (10%)
	if w := profile.DormitoriosWeights[property.Dormitorios]; w > 0 {
		score += 0.10 * math.Min(1.0, w/pesoFavorito)
		factors = append(factors, "Dormitorios")
	}

	// 6. Baños (10%)
	if w := profile.BanosWeights[property.Banos]; w > 0 {
		score += 0.10 * math.Min(1.0, w/pesoFavorito)
		factors = append(factors, "Baños")
	}

	if len(factors) > 0 {
		reason = fmt.Sprintf("Recomendado por: %s", joinStrings(factors, ", "))
	}

	return score, reason, factors
}

// getCollaborativeScores - Filtrado colaborativo por similitud Jaccard:
// "usuarios que interactuaron con las mismas propiedades que tú, también se interesaron en..."
//
// 1. Construye el set de interacción del usuario (favoritos ∪ vistas)
// 2. Encuentra usuarios candidatos que comparten al menos 1 propiedad
// 3. Calcula Jaccard(usuario, candidato) = |A∩B| / |A∪B|
// 4. Cada propiedad de los candidatos (no vista por el usuario) acumula la similitud
// 5. Normaliza los scores a [0,1]
func (s *RecommendationService) getCollaborativeScores(userID uint) (map[uint]float64, error) {
	// Set de interacción del usuario
	userFavs, err := s.getUserPropertyIDs("favorites", userID)
	if err != nil {
		return nil, err
	}
	userViews, err := s.getUserPropertyIDs("view_histories", userID)
	if err != nil {
		return nil, err
	}

	userSet := make(map[uint]bool)
	for id := range userFavs {
		userSet[id] = true
	}
	for id := range userViews {
		userSet[id] = true
	}
	if len(userSet) == 0 {
		return map[uint]float64{}, nil // sin actividad no hay colaborativo
	}

	// Candidatos: usuarios que marcaron/vieron alguna propiedad del set del usuario
	type userHit struct {
		UserID       uint
		SharedCount  int64
	}
	var candidates []userHit
	err = s.db.Raw(`
		SELECT user_id, COUNT(DISTINCT property_id) as shared_count
		FROM (
			SELECT user_id, property_id FROM favorites WHERE property_id IN ?
			UNION
			SELECT user_id, property_id FROM view_histories WHERE property_id IN ?
		) inter
		WHERE user_id != ?
		GROUP BY user_id
		ORDER BY shared_count DESC
		LIMIT ?
	`, keys(userSet), keys(userSet), userID, maxUsuariosCandidatos).Scan(&candidates).Error
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return map[uint]float64{}, nil
	}

	// Sets de interacción de los candidatos
	candidateIDs := make([]uint, 0, len(candidates))
	for _, c := range candidates {
		candidateIDs = append(candidateIDs, c.UserID)
	}

	type pairRow struct {
		UserID     uint
		PropertyID uint
	}
	var favPairs, viewPairs []pairRow
	s.db.Table("favorites").Where("user_id IN ?", candidateIDs).Select("user_id, property_id").Scan(&favPairs)
	s.db.Table("view_histories").Where("user_id IN ?", candidateIDs).Select("user_id, property_id").Scan(&viewPairs)

	candidateSets := make(map[uint]map[uint]bool)
	addPair := func(rows []pairRow) {
		for _, r := range rows {
			if candidateSets[r.UserID] == nil {
				candidateSets[r.UserID] = make(map[uint]bool)
			}
			candidateSets[r.UserID][r.PropertyID] = true
		}
	}
	addPair(favPairs)
	addPair(viewPairs)

	// Jaccard + acumulación de scores por propiedad
	rawScores := make(map[uint]float64)
	for _, candSet := range candidateSets {
		intersection, union := 0, len(userSet)
		for id := range candSet {
			if userSet[id] {
				intersection++
			} else {
				union++
			}
		}
		if intersection == 0 {
			continue
		}
		jaccard := float64(intersection) / float64(union)

		// Las propiedades del candidato que el usuario aún no conoce son candidatas a recomendar
		for propID := range candSet {
			if !userSet[propID] {
				rawScores[propID] += jaccard
			}
		}
	}

	// Normalizar a [0,1] dividiendo por el máximo
	maxScore := 0.0
	for _, v := range rawScores {
		if v > maxScore {
			maxScore = v
		}
	}
	normalized := make(map[uint]float64, len(rawScores))
	for propID, v := range rawScores {
		if maxScore > 0 {
			normalized[propID] = v / maxScore
		}
	}

	fmt.Printf("🤝 Colaborativo: %d candidatos, %d propiedades puntuadas\n", len(candidates), len(normalized))
	return normalized, nil
}

// getUserPropertyIDs - IDs de propiedades de una tabla de interacción para un usuario
func (s *RecommendationService) getUserPropertyIDs(table string, userID uint) (map[uint]bool, error) {
	rows := make([]uint, 0)
	err := s.db.Table(table).Where("user_id = ?", userID).Pluck("property_id", &rows).Error
	if err != nil {
		return nil, err
	}
	set := make(map[uint]bool, len(rows))
	for _, id := range rows {
		set[id] = true
	}
	return set, nil
}

// keys - Extrae las claves de un set como slice
func keys(set map[uint]bool) []uint {
	result := make([]uint, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	return result
}

// sortRecommendations - Ordena recomendaciones por score descendente
func sortRecommendations(recommendations []PropertyRecommendation) {
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
}

// joinStrings - Une strings con separador
func joinStrings(slice []string, separator string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += separator + slice[i]
	}
	return result
}
