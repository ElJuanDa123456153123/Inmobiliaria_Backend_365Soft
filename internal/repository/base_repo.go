package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// BaseRepository - Funcionalidades compartidas para todos los repositorios
type BaseRepository struct {
	db *gorm.DB
}

// NewBaseRepository - Crea una nueva instancia del repositorio base
func NewBaseRepository(db *gorm.DB) *BaseRepository {
	return &BaseRepository{
		db: db,
	}
}

// ========== TRANSACCIONES ==========

// WithTransaction - Ejecuta una función dentro de una transacción
func (r *BaseRepository) WithTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// WithTransactionResult - Ejecuta una función dentro de una transacción y retorna resultado
func (r *BaseRepository) WithTransactionResult(ctx context.Context, fn func(*gorm.DB) (interface{}, error)) (interface{}, error) {
	var result interface{}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = fn(tx)
		return err
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ========== QUERY BUILDERS ==========

// QueryBuilder - Constructor de queries complejas
type QueryBuilder struct {
	db        *gorm.DB
	withCount bool
	page      int
	limit     int
	orderBy   []string
	preloads  []string
	filters   map[string]interface{}
}

// NewQueryBuilder - Crea un nuevo QueryBuilder
func (r *BaseRepository) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		db:       r.db,
		withCount: false,
		page:     1,
		limit:    20,
		orderBy:  []string{"created_at DESC"},
		filters:  make(map[string]interface{}),
	}
}

// Model - Especifica el modelo
func (qb *QueryBuilder) Model(model interface{}) *QueryBuilder {
	qb.db = qb.db.Model(model)
	return qb
}

// Where - Añade condición WHERE
func (qb *QueryBuilder) Where(query string, args ...interface{}) *QueryBuilder {
	qb.db = qb.db.Where(query, args...)
	return qb
}

// WhereIn - Añade condición WHERE IN
func (qb *QueryBuilder) WhereIn(field string, values interface{}) *QueryBuilder {
	qb.db = qb.db.Where(field+" IN ?", values)
	return qb
}

// WhereBetween - Añade condición WHERE BETWEEN
func (qb *QueryBuilder) WhereBetween(field string, min, max interface{}) *QueryBuilder {
	qb.db = qb.db.Where(field+" BETWEEN ? AND ?", min, max)
	return qb
}

// WhereLike - Añade condición LIKE
func (qb *QueryBuilder) WhereLike(field string, value string) *QueryBuilder {
	qb.db = qb.db.Where(field+" ILIKE ?", "%"+value+"%")
	return qb
}

// WhereNull - Añade condición IS NULL
func (qb *QueryBuilder) WhereNull(field string) *QueryBuilder {
	qb.db = qb.db.Where(field + " IS NULL")
	return qb
}

// WhereNotNull - Añade condición IS NOT NULL
func (qb *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
	qb.db = qb.db.Where(field + " IS NOT NULL")
	return qb
}

// WhereOr - Añade condición OR
func (qb *QueryBuilder) WhereOr(query string, args ...interface{}) *QueryBuilder {
	qb.db = qb.db.Or(query, args...)
	return qb
}

// Pagination - Configura paginación
func (qb *QueryBuilder) Pagination(page, limit int) *QueryBuilder {
	qb.page = page
	qb.limit = limit
	return qb
}

// OrderBy - Añade ordenamiento
func (qb *QueryBuilder) OrderBy(field string, direction ...string) *QueryBuilder {
	dir := "ASC"
	if len(direction) > 0 {
		dir = direction[0]
	}
	qb.orderBy = append(qb.orderBy, field+" "+dir)
	return qb
}

// Preload - Añade eager loading
func (qb *QueryBuilder) Preload(field string) *QueryBuilder {
	qb.preloads = append(qb.preloads, field)
	return qb
}

// WithCount - Habilita contador de total
func (qb *QueryBuilder) WithCount() *QueryBuilder {
	qb.withCount = true
	return qb
}

// Exec - Ejecuta la query y retorna resultados
func (qb *QueryBuilder) Exec(dest interface{}) (*QueryResult, error) {
	result := &QueryResult{}

	// Aplicar eager loading
	for _, preload := range qb.preloads {
		qb.db = qb.db.Preload(preload)
	}

	// Aplicar ordenamiento
	for _, order := range qb.orderBy {
		qb.db = qb.db.Order(order)
	}

	// Contar total si se solicita
	if qb.withCount {
		if err := qb.db.Count(&result.Total).Error; err != nil {
			return nil, fmt.Errorf("error contando: %w", err)
		}
	}

	// Aplicar paginación
	offset := (qb.page - 1) * qb.limit
	qb.db = qb.db.Offset(offset).Limit(qb.limit)

	// Ejecutar query
	if err := qb.db.Find(dest).Error; err != nil {
		return nil, fmt.Errorf("error ejecutando query: %w", err)
	}

	result.Page = qb.page
	result.Limit = qb.limit
	result.TotalPages = int(result.Total) / qb.limit
	if int(result.Total)%qb.limit > 0 {
		result.TotalPages++
	}

	return result, nil
}

// QueryResult - Resultado de query compleja
type QueryResult struct {
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// ========== BULK OPERATIONS ==========

// BulkCreate - Crea múltiples registros en una sola query
func (r *BaseRepository) BulkCreate(ctx context.Context, data interface{}) error {
	return r.db.WithContext(ctx).CreateInBatches(data, 100).Error
}

// BulkUpdate - Actualiza múltiples registros
func (r *BaseRepository) BulkUpdate(ctx context.Context, model interface{}, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(model).Updates(updates).Error
}

// BulkDelete - Elimina múltiples registros
func (r *BaseRepository) BulkDelete(ctx context.Context, model interface{}, ids []uint) error {
	return r.db.WithContext(ctx).Delete(model, ids).Error
}

// ========== UTILIDADES ==========

// Exists - Verifica si existe un registro
func (r *BaseRepository) Exists(ctx context.Context, model interface{}, where ...interface{}) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(model).Where(where[0], where[1:]...).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Count - Cuenta registros
func (r *BaseRepository) Count(ctx context.Context, model interface{}, where ...interface{}) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(model)

	if len(where) > 0 {
		query = query.Where(where[0], where[1:]...)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Pluck - Obtiene una columna de valores
func (r *BaseRepository) Pluck(ctx context.Context, model interface{}, column string, dest interface{}) error {
	return r.db.WithContext(ctx).Model(model).Pluck(column, dest).Error
}

// Distinct - Obtiene valores distintos de una columna
func (r *BaseRepository) Distinct(ctx context.Context, model interface{}, column string, dest interface{}) error {
	return r.db.WithContext(ctx).Model(model).Distinct(column).Pluck(column, dest).Error
}

// ========== RAW QUERIES ==========

// RawSQL - Ejecuta SQL crudo
func (r *BaseRepository) RawSQL(ctx context.Context, sql string, dest interface{}, values ...interface{}) error {
	return r.db.WithContext(ctx).Raw(sql, values...).Scan(dest).Error
}

// ExecSQL - Ejecuta SQL crudo (sin retornar datos)
func (r *BaseRepository) ExecSQL(ctx context.Context, sql string, values ...interface{}) error {
	return r.db.WithContext(ctx).Exec(sql, values...).Error
}
