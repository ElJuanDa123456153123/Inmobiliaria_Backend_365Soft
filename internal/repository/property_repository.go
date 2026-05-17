package repository

import (
	"fmt"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// PropertyRepository - Repositorio para operaciones de base de datos de propiedades
type PropertyRepository struct {
	DB *config.Database
}

// NewPropertyRepository - Crea un nuevo repositorio de propiedades
func NewPropertyRepository(db *config.Database) *PropertyRepository {
	return &PropertyRepository{DB: db}
}

// Create - Crea una nueva propiedad
func (r *PropertyRepository) Create(property *models.Property) error {
	return r.DB.DB.Create(property).Error
}

// GetByID - Obtiene una propiedad por ID
func (r *PropertyRepository) GetByID(id uint) (*models.Property, error) {
	var property models.Property
	err := r.DB.DB.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("orden ASC")
		}).
		First(&property, id).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// GetByIDWithUser - Obtiene una propiedad por ID con información del usuario
func (r *PropertyRepository) GetByIDWithUser(id uint) (*models.Property, error) {
	var property models.Property
	err := r.DB.DB.
		Preload("User").
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("orden ASC")
		}).
		First(&property, id).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// GetByIDWithRelations - Obtiene una propiedad por ID con todas sus relaciones (User, PropertyImages, Advisor, PropertyAdvisors)
func (r *PropertyRepository) GetByIDWithRelations(id uint) (*models.Property, error) {
	var property models.Property
	err := r.DB.DB.
		Preload("User").
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("orden ASC")
		}).
		Preload("Advisor").
		Preload("PropertyAdvisors.Advisor"). // Incluir múltiples asesores asignados
		First(&property, id).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// Update - Actualiza una propiedad
func (r *PropertyRepository) Update(property *models.Property) error {
	return r.DB.DB.Save(property).Error
}

// Delete - Soft delete de una propiedad
func (r *PropertyRepository) Delete(property *models.Property) error {
	return r.DB.DB.Delete(property).Error
}

// List - Lista propiedades con filtros y paginación
func (r *PropertyRepository) List(listParams ListPropertyParams) ([]models.Property, int64, error) {
	query := r.DB.DB.Model(&models.Property{})

	// Aplicar filtros
	if listParams.Estado != "" {
		query = query.Where("estado_propiedad = ?", listParams.Estado)
	}

	if listParams.TipoOperacion != "" {
		query = query.Where("tipo_operacion = ?", listParams.TipoOperacion)
	}

	if listParams.TipoPropiedad != "" {
		query = query.Where("tipo_propiedad = ?", listParams.TipoPropiedad)
	}

	if listParams.Zona != "" {
		searchTerm := "%" + listParams.Zona + "%"
		query = query.Where("(zona ILIKE ? OR direccion ILIKE ? OR titulo ILIKE ?)", searchTerm, searchTerm, searchTerm)
	}

	if listParams.Moneda != "" {
		query = query.Where("moneda = ?", listParams.Moneda)
	}

	if listParams.PrecioMin > 0 {
		query = query.Where("precio >= ?", listParams.PrecioMin)
	}

	if listParams.PrecioMax > 0 {
		query = query.Where("precio <= ?", listParams.PrecioMax)
	}

	if listParams.MetrosMin > 0 {
		query = query.Where("metros_edificados >= ?", listParams.MetrosMin)
	}

	if listParams.MetrosMax > 0 {
		query = query.Where("metros_edificados <= ?", listParams.MetrosMax)
	}

	if listParams.Dormitorios != "" {
		query = query.Where("dormitorios = ?", listParams.Dormitorios)
	}

	if listParams.Banos != "" {
		query = query.Where("banos = ?", listParams.Banos)
	}

	if listParams.Garaje != "" {
		query = query.Where("garaje = ?", listParams.Garaje)
	}

	if listParams.AceptaPermuta != nil {
		query = query.Where("acepta_permuta = ?", *listParams.AceptaPermuta)
	}

	if listParams.Financia != nil {
		query = query.Where("financia = ?", *listParams.Financia)
	}

	if listParams.Condominio != nil {
		query = query.Where("condominio = ?", *listParams.Condominio)
	}

	if listParams.AptoOficina != nil {
		query = query.Where("apto_oficina = ?", *listParams.AptoOficina)
	}

	if listParams.ViviendaSocial != nil {
		query = query.Where("vivienda_social = ?", *listParams.ViviendaSocial)
	}

	if listParams.SearchText != "" {
		searchTerm := "%" + listParams.SearchText + "%"
		query = query.Where("(titulo ILIKE ? OR descripcion ILIKE ? OR zona ILIKE ? OR direccion ILIKE ?)", searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if listParams.UserID != nil {
		query = query.Where("user_id = ?", *listParams.UserID)
	}

	// Contar total (usando una sesión aislada)
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Aplicar ordenamiento
	orderByColumn := "created_at"
	switch listParams.OrderBy {
	case "precio":
		orderByColumn = "precio"
	case "vistas":
		orderByColumn = "vistas_contador"
	case "metros":
		orderByColumn = "metros_edificados"
	}

	// Aplicar paginación y obtener resultados (usando la query original)
	var properties []models.Property
	err := query.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("principal DESC, orden ASC")
		}).
		Preload("PropertyAdvisors.Advisor"). // Incluir múltiples asesores asignados
		Order(orderByColumn + " " + listParams.Order).
		Limit(listParams.Limit).
		Offset(listParams.Offset).
		Find(&properties).Error

	if err != nil {
		return nil, 0, err
	}

	return properties, total, nil
}

// IncrementViewCount - Incrementa el contador de vistas de una propiedad
func (r *PropertyRepository) IncrementViewCount(propertyID uint) error {
	return r.DB.DB.Model(&models.Property{}).
		Where("id = ?", propertyID).
		UpdateColumn("vistas_contador", gorm.Expr("vistas_contador + 1")).Error
}

// GetByUserID - Obtiene todas las propiedades de un usuario
func (r *PropertyRepository) GetByUserID(userID uint) ([]models.Property, error) {
	var properties []models.Property
	err := r.DB.DB.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("orden ASC")
		}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&properties).Error
	if err != nil {
		return nil, err
	}
	return properties, nil
}

// CountByUserID - Cuenta las propiedades de un usuario
func (r *PropertyRepository) CountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.DB.DB.Model(&models.Property{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByID - Verifica si existe una propiedad por ID
func (r *PropertyRepository) ExistsByID(id uint) (bool, error) {
	var count int64
	err := r.DB.DB.Model(&models.Property{}).
		Where("id = ?", id).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FullTextSearch - Búsqueda full-text en propiedades
func (r *PropertyRepository) FullTextSearch(query string, limit, offset int) ([]models.Property, int64, error) {
	// Búsqueda usando tsvector con ranking
	searchQuery := fmt.Sprintf(`
		SELECT id,
		       ts_rank(search_vector, plainto_tsquery('spanish', ?)) as rank
		FROM propiedades
		WHERE search_vector @@ plainto_tsquery('spanish', ?)
			AND estado_propiedad = 'activo'
			AND deleted_at IS NULL
		ORDER BY rank DESC, created_at DESC
		LIMIT ? OFFSET ?
	`)

	var results []struct {
		ID   uint
		Rank float64
	}

	if err := r.DB.DB.Raw(searchQuery, query, query, limit, offset).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	// Obtener IDs de propiedades
	var ids []uint
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	if len(ids) == 0 {
		return []models.Property{}, 0, nil
	}

	// Contar total
	var count int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM propiedades
		WHERE search_vector @@ plainto_tsquery('spanish', ?)
			AND estado_propiedad = 'activo'
			AND deleted_at IS NULL
	`)
	if err := r.DB.DB.Raw(countQuery, query).Scan(&count).Error; err != nil {
		return nil, 0, err
	}

	// Obtener propiedades completas
	var properties []models.Property
	if err := r.DB.DB.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Where("principal = ? OR principal IS NULL", true).
				Order("orden ASC")
		}).
		Where("id IN ?", ids).
		Order("created_at DESC").
		Find(&properties).Error; err != nil {
		return nil, 0, err
	}

	return properties, count, nil
}

// FullTextSearchWithFilters - Búsqueda full-text con filtros adicionales
func (r *PropertyRepository) FullTextSearchWithFilters(
	query string,
	filters ListSearchParams,
	limit, offset int,
) ([]models.Property, int64, error) {
	// Construir query SQL dinámico
	whereClause := "WHERE search_vector @@ plainto_tsquery('spanish', ?) AND estado_propiedad = 'activo' AND deleted_at IS NULL"
	args := []interface{}{query}

	// Agregar filtros
	if filters.TipoOperacion != "" {
		whereClause += " AND tipo_operacion = ?"
		args = append(args, filters.TipoOperacion)
	}
	if filters.TipoPropiedad != "" {
		whereClause += " AND tipo_propiedad = ?"
		args = append(args, filters.TipoPropiedad)
	}
	if filters.Moneda != "" {
		whereClause += " AND moneda = ?"
		args = append(args, filters.Moneda)
	}
	if filters.PrecioMin > 0 {
		whereClause += " AND precio >= ?"
		args = append(args, filters.PrecioMin)
	}
	if filters.PrecioMax > 0 {
		whereClause += " AND precio <= ?"
		args = append(args, filters.PrecioMax)
	}
	if filters.MetrosMin > 0 {
		whereClause += " AND metros_edificados >= ?"
		args = append(args, filters.MetrosMin)
	}
	if filters.MetrosMax > 0 {
		whereClause += " AND metros_edificados <= ?"
		args = append(args, filters.MetrosMax)
	}

	// Consulta de conteo
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM propiedades
		%s
	`, whereClause)

	var count int64
	if err := r.DB.DB.Raw(countQuery, args...).Scan(&count).Error; err != nil {
		return nil, 0, err
	}

	// Consulta de búsqueda con ranking
	searchQuery := fmt.Sprintf(`
		SELECT id, ts_rank(search_vector, plainto_tsquery('spanish', ?)) as rank
		FROM propiedades
		%s
		ORDER BY rank DESC, created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append([]interface{}{query}, args...)
	args = append(args, limit, offset)

	var results []struct {
		ID   uint
		Rank float64
	}

	if err := r.DB.DB.Raw(searchQuery, args...).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	if len(results) == 0 {
		return []models.Property{}, count, nil
	}

	// Obtener propiedades completas
	var ids []uint
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	var properties []models.Property
	err := r.DB.DB.
		Preload("PropertyImages", func(db *gorm.DB) *gorm.DB {
			return db.Where("principal = ? OR principal IS NULL", true).
				Order("orden ASC")
		}).
		Where("id IN ?", ids).
		Find(&properties).Error

	if err != nil {
		return nil, 0, err
	}

	return properties, count, nil
}

// ListSearchParams - Parámetros de búsqueda
type ListSearchParams struct {
	TipoOperacion string
	TipoPropiedad string
	Moneda        string
	PrecioMin     float64
	PrecioMax     float64
	MetrosMin     float64
	MetrosMax     float64
}

// ListPropertyParams - Parámetros para listar propiedades
type ListPropertyParams struct {
	Estado         string
	TipoOperacion  string
	TipoPropiedad  string
	Zona           string
	Moneda         string
	PrecioMin      float64
	PrecioMax      float64
	MetrosMin      float64
	MetrosMax      float64
	Dormitorios    string
	Banos          string
	Garaje         string
	AceptaPermuta  *bool
	Financia       *bool
	Condominio     *bool
	AptoOficina    *bool
	ViviendaSocial *bool
	SearchText     string
	UserID         *uint
	OrderBy        string
	Order          string
	Limit          int
	Offset         int
}
