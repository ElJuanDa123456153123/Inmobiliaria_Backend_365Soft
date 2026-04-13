package services

import (
	"context"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/validators"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// SearchService - Interface para servicio de búsqueda
type SearchService interface {
	// Búsqueda principal
	Search(ctx context.Context, query string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)

	// Búsquedas específicas
	SearchByLocation(ctx context.Context, ciudad, zona string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)
	SearchByPriceRange(ctx context.Context, min, max float64, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)
	SearchByType(ctx context.Context, tipos []models.PropertyType, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error)

	// Autocomplete
	AutocompleteCities(ctx context.Context, query string) ([]string, error)
	AutocompleteZones(ctx context.Context, ciudad, query string) ([]string, error)

	// Sugerencias
	GetSuggestedProperties(ctx context.Context, userID uint, limit int) ([]*dto.PropertyResponse, error)
	GetTrendingProperties(ctx context.Context, limit int) ([]*dto.PropertyResponse, error)
}

// searchService - Implementación del servicio
type searchService struct {
	propertyRepo repository.PropertyRepository
	validator     *validators.PropertyValidator
}

// NewSearchService - Crea una nueva instancia del servicio
func NewSearchService(propertyRepo repository.PropertyRepository) SearchService {
	return &searchService{
		propertyRepo: propertyRepo,
		validator:    validators.NewPropertyValidator(),
	}
}

// ========== BÚSQUEDA PRINCIPAL ==========

// Search - Búsqueda full-text de propiedades
func (s *searchService) Search(ctx context.Context, query string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Limpiar y normalizar query
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, apperrors.ValidationError("query", "query no puede estar vacío")
	}

	// 2. Validar filtros
	if errs := s.validator.ValidatePropertyFilter(filter); len(errs) > 0 {
		details := validators.FormatErrorsForResponse(errs)
		return nil, apperrors.ValidationErrors(details)
	}

	// 3. Establecer defaults
	filter.SetDefaults()

	// 4. Buscar propiedades
	properties, total, err := s.propertyRepo.Search(ctx, query, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 5. Mapear a responses
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	// 6. Crear response
	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// ========== BÚSQUEDAS ESPECÍFICAS ==========

// SearchByLocation - Búsqueda por ubicación
func (s *searchService) SearchByLocation(ctx context.Context, ciudad, zona string, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Crear filtro de ubicación
	if filter == nil {
		filter = &dto.PropertyFilter{}
	}
	filter.SetDefaults()

	// 2. Aplicar filtros de ubicación
	if ciudad != "" {
		filter.Ciudad = ciudad
	}

	if zona != "" {
		filter.Zona = zona
	}

	// 3. Validar filtros
	if errs := s.validator.ValidatePropertyFilter(filter); len(errs) > 0 {
		details := validators.FormatErrorsForResponse(errs)
		return nil, apperrors.ValidationErrors(details)
	}

	// 4. Buscar
	properties, total, err := s.propertyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 5. Mapear
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// SearchByPriceRange - Búsqueda por rango de precio
func (s *searchService) SearchByPriceRange(ctx context.Context, min, max float64, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Validar rango
	if min < 0 || max < 0 {
		return nil, apperrors.InvalidPrice("El precio no puede ser negativo")
	}

	if min >= max {
		return nil, apperrors.InvalidPrice("El precio mínimo debe ser menor al máximo")
	}

	// 2. Crear filtro
	if filter == nil {
		filter = &dto.PropertyFilter{}
	}
	filter.SetDefaults()
	filter.PrecioMin = &min
	filter.PrecioMax = &max

	// 3. Buscar
	properties, total, err := s.propertyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 4. Mapear
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// SearchByType - Búsqueda por tipo de propiedad
func (s *searchService) SearchByType(ctx context.Context, tipos []models.PropertyType, filter *dto.PropertyFilter) (*dto.PropertyListResponse, error) {
	// 1. Validar tipos
	if len(tipos) == 0 {
		return nil, apperrors.ValidationError("tipos", "Debe especificar al menos un tipo de propiedad")
	}

	// 2. Convertir a strings
	tipoStrings := make([]string, len(tipos))
	for i, tipo := range tipos {
		tipoStrings[i] = string(tipo)
	}

	// 3. Crear filtro
	if filter == nil {
		filter = &dto.PropertyFilter{}
	}
	filter.SetDefaults()
	filter.TipoPropiedad = tipoStrings

	// 4. Buscar
	properties, total, err := s.propertyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 5. Mapear
	items := make([]dto.PropertyListItemResponse, len(properties))
	for i, property := range properties {
		items[i] = dto.MapPropertyToListItem(property)
	}

	pagination := dto.NewPaginationResponse(filter.Page, filter.Limit, total)
	return &dto.PropertyListResponse{
		Data:       items,
		Pagination: pagination,
	}, nil
}

// ========== AUTOCOMPLETE ==========

// AutocompleteCities - Autocomplete de ciudades
func (s *searchService) AutocompleteCities(ctx context.Context, query string) ([]string, error) {
	// 1. Limpiar query
	query = strings.TrimSpace(query)
	if query == "" {
		return []string{}, nil
	}

	// 2. Buscar ciudades distintas
	query = strings.ToLower(query)

	// TODO: Implementar query real para obtener ciudades distintas
	// Por ahora, retornar ciudades hardcodeadas
	ciudades := []string{
		"Santa Cruz de la Sierra",
		"La Paz",
		"Cochabamba",
		"Sucre",
		"El Alto",
	}

	// Filtrar las que coinciden
	result := make([]string, 0)
	for _, ciudad := range ciudades {
		if strings.Contains(strings.ToLower(ciudad), query) {
			result = append(result, ciudad)
		}
	}

	return result, nil
}

// AutocompleteZones - Autocomplete de zonas
func (s *searchService) AutocompleteZones(ctx context.Context, ciudad, query string) ([]string, error) {
	// 1. Limpiar query
	query = strings.TrimSpace(query)
	if query == "" {
		return []string{}, nil
	}

	// 2. Buscar zonas distintas para la ciudad
	query = strings.ToLower(query)

	// TODO: Implementar query real
	zonas := map[string][]string{
		"Santa Cruz de la Sierra": {
			"Equipetrol",
			"San Martín",
			"El Urubó",
			"Las Palmas",
			"Los Pinos",
		},
		"La Paz": {
			"Calacoto",
			"San Pedro",
			"Sopocachi",
			"Miraflores",
		},
	}

	zonasCiudad, ok := zonas[ciudad]
	if !ok {
		return []string{}, nil
	}

	// Filtrar
	result := make([]string, 0)
	for _, zona := range zonasCiudad {
		if strings.Contains(strings.ToLower(zona), query) {
			result = append(result, zona)
		}
	}

	return result, nil
}

// ========== SUGERENCIAS ==========

// GetSuggestedProperties - Propiedades sugeridas para un usuario
func (s *searchService) GetSuggestedProperties(ctx context.Context, userID uint, limit int) ([]*dto.PropertyResponse, error) {
	// 1. Obtener propiedades destacadas como sugerencia
	properties, err := s.propertyRepo.GetFeatured(ctx, limit)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 2. Mapear a responses
	responses := make([]*dto.PropertyResponse, len(properties))
	for i, property := range properties {
		prop := dto.MapPropertyToResponse(property, false)
		responses[i] = prop
	}

	return responses, nil
}

// GetTrendingProperties - Propiedades en tendencia
func (s *searchService) GetTrendingProperties(ctx context.Context, limit int) ([]*dto.PropertyResponse, error) {
	// 1. Crear filtro para propiedades populares
	filter := &dto.PropertyFilter{
		Page:    1,
		Limit:   limit,
		OrderBy: "mas_visitados",
	}
	filter.SetDefaults()

	// 2. Buscar
	properties, _, err := s.propertyRepo.List(ctx, filter)
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 3. Mapear
	responses := make([]*dto.PropertyResponse, len(properties))
	for i, property := range properties {
		prop := dto.MapPropertyToResponse(property, false)
		responses[i] = prop
	}

	return responses, nil
}

// ========== HELPER METHODS ==========

// buildSearchQuery - Construye query de búsqueda
func (s *searchService) buildSearchQuery(query string) string {
	// 1. Normalizar query
	query = strings.ToLower(query)
	query = strings.TrimSpace(query)

	// 2. Eliminar palabras comunes (stop words)
	stopWords := []string{"el", "la", "los", "las", "un", "una", "de", "en", "con"}
	words := strings.Fields(query)

	filtered := make([]string, 0, len(words))
	for _, word := range words {
		isStopWord := false
		for _, stop := range stopWords {
			if word == stop {
				isStopWord = true
				break
			}
		}
		if !isStopWord {
			filtered = append(filtered, word)
		}
	}

	// 3. Unir con AND para búsqueda booleana
	return strings.Join(filtered, " & ")
}

// extractSearchTerms - Extrae términos de búsqueda específicos
func (s *searchService) extractSearchTerms(query string) *SearchTerms {
	terms := &SearchTerms{}

	query = strings.ToLower(query)

	// Buscar palabras clave de tipo de propiedad
	tipoKeywords := map[string]models.PropertyType{
		"casa":        models.PropertyTypeCasa,
		"departamento": models.PropertyTypeDepartamento,
		"depto":       models.PropertyTypeDepartamento,
		"terreno":     models.PropertyTypeTerreno,
		"local":       models.PropertyTypeLocal,
		"oficina":     models.PropertyTypeOficina,
		"galpon":      models.PropertyTypeGalpon,
		"hacienda":    models.PropertyTypeHacienda,
		"lote":        models.PropertyTypeLote,
	}

	for keyword, tipo := range tipoKeywords {
		if strings.Contains(query, keyword) {
			terms.Tipos = append(terms.Tipos, tipo)
		}
	}

	// Buscar palabras clave de operación
	if strings.Contains(query, "venta") || strings.Contains(query, "vender") {
		operacion := models.OperationVenta
		terms.Operacion = &operacion
	}

	if strings.Contains(query, "alquiler") || strings.Contains(query, "alquilar") {
		operacion := models.OperationAlquiler
		terms.Operacion = &operacion
	}

	if strings.Contains(query, "anticretico") || strings.Contains(query, "anticrético") {
		operacion := models.OperationAnticretico
		terms.Operacion = &operacion
	}

	// Buscar amenities
	if strings.Contains(query, "piscina") {
		piscina := true
		terms.TienePiscina = &piscina
	}

	if strings.Contains(query, "jardin") || strings.Contains(query, "jardín") {
		jardin := true
		terms.TieneJardin = &jardin
	}

	return terms
}

// SearchTerms - Términos de búsqueda extraídos
type SearchTerms struct {
	Tipos         []models.PropertyType
	Operacion     *models.PropertyOperation
	TienePiscina  *bool
	TieneJardin   *bool
}

// NormalizeQuery - Normaliza query de búsqueda
func NormalizeQuery(query string) string {
	query = strings.ToLower(query)
	query = strings.TrimSpace(query)

	// Eliminar acentos
	accents := map[rune]string{
		'á': "a", 'é': "e", 'í': "i", 'ó': "o", 'ú': "u",
		'à': "a", 'è': "e", 'ì': "i", 'ò': "o", 'ù': "u",
		'ä': "a", 'ë': "e", 'ï': "i", 'ö': "o", 'ü': "u",
		'â': "a", 'ê': "e", 'î': "i", 'ô': "o", 'û': "u",
		'ã': "a", 'ẽ': "e", 'ĩ': "i", 'õ': "o", 'ũ': "u",
	}

	for old, new := range accents {
		query = strings.ReplaceAll(query, string(old), new)
	}

	return query
}

// CalculateRelevanceScore - Calcula score de relevancia para sorting
func CalculateRelevanceScore(property *models.Property, query string) float64 {
	score := 0.0
	query = strings.ToLower(query)

	// 1. Coincidencia exacta en título (+50)
	if strings.Contains(strings.ToLower(property.Titulo), query) {
		score += 50
	}

	// 2. Coincidencia exacta en descripción (+20)
	if strings.Contains(strings.ToLower(property.Descripcion), query) {
		score += 20
	}

	// 3. Coincidencia en ciudad (+10)
	if strings.Contains(strings.ToLower(property.Ciudad), query) {
		score += 10
	}

	// 4. Coincidencia en zona (+5)
	if strings.Contains(strings.ToLower(property.Zona), query) {
		score += 5
	}

	// 5. Propiedad destacada (+15)
	if property.Destacado {
		score += 15
	}

	// 6. Factor de visitas (normalizado 0-10)
	visitScore := float64(property.NumeroVisitas) / 100.0
	if visitScore > 10 {
		visitScore = 10
	}
	score += visitScore

	return score
}

// FilterByRelevance - Filtra y ordena propiedades por relevancia
func FilterByRelevance(properties []*models.Property, query string) []*models.Property {
	// 1. Calcular scores
	type ScoredProperty struct {
		Property *models.Property
		Score    float64
	}

	scored := make([]ScoredProperty, len(properties))
	for i, property := range properties {
		scored[i] = ScoredProperty{
			Property: property,
			Score:    CalculateRelevanceScore(property, query),
		}
	}

	// 2. Ordenar por score (bubble sort simple)
	for i := 0; i < len(scored)-1; i++ {
		for j := 0; j < len(scored)-i-1; j++ {
			if scored[j].Score < scored[j+1].Score {
				scored[j], scored[j+1] = scored[j+1], scored[j]
			}
		}
	}

	// 3. Extraer propiedades ordenadas
	result := make([]*models.Property, len(scored))
	for i, sp := range scored {
		result[i] = sp.Property
	}

	return result
}
