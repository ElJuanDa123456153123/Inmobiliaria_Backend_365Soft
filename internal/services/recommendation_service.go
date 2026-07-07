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

// RecommendationService - Servicio de recomendaciones de propiedades
// SIN valuación automática de precios con IA
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

// UserProfile - Perfil de preferencias del usuario
type UserProfile struct {
	UserID                      uint
	PreferredZones             []string
	PreferredPropertyTypes      []models.PropertyType
	PreferredOperationTypes    []models.OperationType
	PreferredPriceRange        PriceRange
	PreferredDormitorios       []string
	PreferredBanos             []string
	AveragePriceViewed         float64
	HasGarajePreference        bool
	AmenitiesPreferences       []string
}

// PriceRange - Rango de precios preferido
type PriceRange struct {
	Min float64
	Max float64
}

// PropertyRecommendation - Propiedad recomendada con score
type PropertyRecommendation struct {
	Property    models.Property `json:"property"`
	Score       float64         `json:"score"`
	Reason      string          `json:"reason"`
	MatchFactors []string       `json:"match_factors"`
}

// GetContentBasedRecommendations - Obtiene recomendaciones basadas en el perfil del usuario
// Algoritmo: Analiza favoritos, vistas y búsquedas para construir un perfil de preferencias
func (s *RecommendationService) GetContentBasedRecommendations(userID uint, limit int) ([]PropertyRecommendation, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// 1. Construir perfil del usuario
	profile, err := s.buildUserProfile(userID)
	if err != nil {
		return nil, err
	}

	fmt.Printf("📊 Perfil de usuario %d construido: %d zonas, %d tipos de propiedad\n",
		userID, len(profile.PreferredZones), len(profile.PreferredPropertyTypes))

	// 2. Obtener propiedades activas que no han sido vistas por el usuario
	var properties []models.Property
	err = s.db.Where("estado_propiedad = ?", models.PropertyStatusActivo).
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

	fmt.Printf("🔍 Evaluando %d propiedades para recomendar...\n", len(properties))

	// 3. Calcular score para cada propiedad
	recommendations := make([]PropertyRecommendation, 0)
	for _, property := range properties {
		score, reason, factors := s.calculatePropertyScore(property, profile)

		// Solo recomendar si tiene un score mínimo
		if score >= 0.3 {
			recommendations = append(recommendations, PropertyRecommendation{
				Property:    property,
				Score:       score,
				Reason:      reason,
				MatchFactors: factors,
			})
		}
	}

	// 4. Ordenar por score descendente
	s.sortRecommendations(recommendations)

	// 5. Limitar resultados
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	fmt.Printf("✅ %d recomendaciones generadas para usuario %d\n", len(recommendations), userID)
	return recommendations, nil
}

// buildUserProfile - Construye el perfil de preferencias del usuario
func (s *RecommendationService) buildUserProfile(userID uint) (*UserProfile, error) {
	profile := &UserProfile{
		UserID:                 userID,
		PreferredZones:        make([]string, 0),
		PreferredPropertyTypes: make([]models.PropertyType, 0),
		PreferredOperationTypes: make([]models.OperationType, 0),
		PreferredDormitorios:   make([]string, 0),
		PreferredBanos:        make([]string, 0),
		AmenitiesPreferences:  make([]string, 0),
	}

	// 1. Analizar favoritos (peso más alto)
	favorites, err := s.FavoriteRepo.GetByUser(userID)
	if err == nil && len(favorites) > 0 {
		fmt.Printf("📌 Analizando %d favoritos...\n", len(favorites))
		for _, fav := range favorites {
			// Property es un struct embedded, no un puntero
			if fav.Property.ID != 0 { // Verificar si está cargado
				profile.addPropertyPreferences(fav.Property, 2.0) // Peso alto
			}
		}
	}

	// 2. Analizar historial de vistas
	viewHistory, err := s.ViewHistoryRepo.GetByUser(userID)
	if err == nil && len(viewHistory) > 0 {
		fmt.Printf("👁️  Analizando %d vistas...\n", len(viewHistory))
		for _, view := range viewHistory {
			// Property es un struct embedded, no un puntero
			if view.Property.ID != 0 { // Verificar si está cargado
				profile.addPropertyPreferences(view.Property, 1.0) // Peso medio
			}
		}
	}

	// 3. Analizar búsquedas guardadas
	savedSearches, err := s.SavedSearchRepo.GetActiveByUser(userID)
	if err == nil && len(savedSearches) > 0 {
		fmt.Printf("🔍 Analizando %d búsquedas guardadas...\n", len(savedSearches))
		for _, search := range savedSearches {
			profile.addSearchPreferences(json.RawMessage(search.Filtros), 1.5) // Puesto medio-alto
		}
	}

	// 4. Calcular promedio de precios visto
	if profile.AveragePriceViewed > 0 {
		// Establecer rango de precio preferido (+- 30% del promedio)
		profile.PreferredPriceRange = PriceRange{
			Min: profile.AveragePriceViewed * 0.7,
			Max: profile.AveragePriceViewed * 1.3,
		}
	}

	return profile, nil
}

// addPropertyPreferences - Agrega preferencias basadas en una propiedad
func (p *UserProfile) addPropertyPreferences(property models.Property, weight float64) {
	// Agregar zona
	if property.Zona != "" && !p.contains(p.PreferredZones, property.Zona) {
		p.PreferredZones = append(p.PreferredZones, property.Zona)
	}

	// Agregar tipo de propiedad
	if property.TipoPropiedad != "" && !p.containsPropertyType(property.TipoPropiedad) {
		p.PreferredPropertyTypes = append(p.PreferredPropertyTypes, property.TipoPropiedad)
	}

	// Agregar tipo de operación
	if property.TipoOperacion != "" && !p.containsOperationType(property.TipoOperacion) {
		p.PreferredOperationTypes = append(p.PreferredOperationTypes, property.TipoOperacion)
	}

	// Actualizar promedio de precios (promedio móvil)
	if property.Precio > 0 {
		if p.AveragePriceViewed == 0 {
			p.AveragePriceViewed = property.Precio
		} else {
			// Promedio ponderado
			p.AveragePriceViewed = (p.AveragePriceViewed + property.Precio) / 2
		}
	}

	// Agregar dormitorios preferidos
	if property.Dormitorios != "" && property.Dormitorios != "sin-definir" {
		if !p.contains(p.PreferredDormitorios, property.Dormitorios) {
			p.PreferredDormitorios = append(p.PreferredDormitorios, property.Dormitorios)
		}
	}

	// Agregar baños preferidos
	if property.Banos != "" && property.Banos != "sin-definir" {
		if !p.contains(p.PreferredBanos, property.Banos) {
			p.PreferredBanos = append(p.PreferredBanos, property.Banos)
		}
	}
}

// addSearchPreferences - Agrega preferencias basadas en filtros de búsqueda
func (p *UserProfile) addSearchPreferences(filtrosJSON json.RawMessage, weight float64) {
	// Convertir JSON a map
	var filtros map[string]interface{}
	if err := json.Unmarshal(filtrosJSON, &filtros); err != nil {
		return
	}

	// Extraer zona de los filtros
	if zona, ok := filtros["zona"].(string); ok && zona != "" {
		if !p.contains(p.PreferredZones, zona) {
			p.PreferredZones = append(p.PreferredZones, zona)
		}
	}

	// Extraer tipo de propiedad
	if tipoProp, ok := filtros["tipo_propiedad"].(string); ok && tipoProp != "" {
		if !p.containsPropertyType(models.PropertyType(tipoProp)) {
			p.PreferredPropertyTypes = append(p.PreferredPropertyTypes, models.PropertyType(tipoProp))
		}
	}

	// Extraer tipo de operación
	if tipoOp, ok := filtros["tipo_operacion"].(string); ok && tipoOp != "" {
		if !p.containsOperationType(models.OperationType(tipoOp)) {
			p.PreferredOperationTypes = append(p.PreferredOperationTypes, models.OperationType(tipoOp))
		}
	}
}

// calculatePropertyScore - Calcula el score de una propiedad basado en el perfil
// Algoritmo: Coincidencia de preferencias (sin valuación de precios)
func (s *RecommendationService) calculatePropertyScore(property models.Property, profile *UserProfile) (float64, string, []string) {
	score := 0.0
	factors := make([]string, 0)
	reason := "Propiedad recomendada basada en tu historial"

	// 1. Coincidencia de zona (30% del peso)
	if profile.contains(profile.PreferredZones, property.Zona) {
		score += 0.3
		factors = append(factors, "Zona preferida")
	}

	// 2. Coincidencia de tipo de propiedad (15% del peso)
	if profile.containsPropertyType(property.TipoPropiedad) {
		score += 0.15
		factors = append(factors, "Tipo de propiedad")
	}

	// 3. Coincidencia de tipo de operación (15% del peso)
	if profile.containsOperationType(property.TipoOperacion) {
		score += 0.15
		factors = append(factors, "Tipo de operación")
	}

	// 4. Coincidencia de precio (20% del peso)
	// SOLO si el usuario ha visto suficientes propiedades para tener un promedio
	if profile.AveragePriceViewed > 0 {
		priceDiff := math.Abs(property.Precio - profile.AveragePriceViewed) / profile.AveragePriceViewed
		if priceDiff <= 0.3 { // Dentro del 30% del promedio
			score += 0.2
			factors = append(factors, "Precio acorde a tu historial")
		} else if priceDiff <= 0.5 { // Dentro del 50%
			score += 0.1
		}
	}

	// 5. Coincidencia de dormitorios (10% del peso)
	if profile.contains(profile.PreferredDormitorios, property.Dormitorios) {
		score += 0.1
		factors = append(factors, "Dormitorios")
	}

	// 6. Coincidencia de baños (10% del peso)
	if profile.contains(profile.PreferredBanos, property.Banos) {
		score += 0.1
		factors = append(factors, "Baños")
	}

	// Construir razón
	if len(factors) > 0 {
		reason = fmt.Sprintf("Recomendado por: %s", joinStrings(factors, ", "))
	}

	return score, reason, factors
}

// sortRecommendations - Ordena recomendaciones por score descendente
func (s *RecommendationService) sortRecommendations(recommendations []PropertyRecommendation) {
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
}

// Helper functions

func (p *UserProfile) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (p *UserProfile) containsPropertyType(item models.PropertyType) bool {
	for _, t := range p.PreferredPropertyTypes {
		if t == item {
			return true
		}
	}
	return false
}

func (p *UserProfile) containsOperationType(item models.OperationType) bool {
	for _, t := range p.PreferredOperationTypes {
		if t == item {
			return true
		}
	}
	return false
}

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
