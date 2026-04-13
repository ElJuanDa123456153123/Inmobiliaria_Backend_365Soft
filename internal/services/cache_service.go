package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/redis/go-redis/v9"
	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// CacheService - Interface para servicio de caché
type CacheService interface {
	// Propiedades
	GetProperty(ctx context.Context, id uint) (*models.Property, error)
	SetProperty(ctx context.Context, property *models.Property, ttl time.Duration) error
	InvalidateProperty(ctx context.Context, id uint) error

	GetPropertyList(ctx context.Context, filter string) (*CachedPropertyList, error)
	SetPropertyList(ctx context.Context, filter string, properties []*models.Property, ttl time.Duration) error
	InvalidatePropertyList(ctx context.Context) error

	// Categorías
	GetCategories(ctx context.Context) ([]*models.Category, error)
	SetCategories(ctx context.Context, categories []*models.Category, ttl time.Duration) error
	InvalidateCategories(ctx context.Context) error

	// Contadores
	IncrementVisits(ctx context.Context, propertyID uint) (int64, error)
	IncrementContacts(ctx context.Context, propertyID uint) (int64, error)

	// Gestión de caché
	ClearAll(ctx context.Context) error
	GetStats(ctx context.Context) (*CacheStats, error)
}

// CachedPropertyList - Lista de propiedades en caché
type CachedPropertyList struct {
	Properties    []*models.Property `json:"properties"`
	Total         int64              `json:"total"`
	CachedAt      time.Time          `json:"cached_at"`
}

// CacheStats - Estadísticas del caché
type CacheStats struct {
	Keys     int64 `json:"keys"`
	Memory   int64 `json:"memory"`
	Hits     int64 `json:"hits"`
	Misses   int64 `json:"misses"`
	HitRatio float64 `json:"hit_ratio"`
}

// cacheService - Implementación del servicio con Redis
type cacheService struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCacheService - Crea una nueva instancia del servicio
func NewCacheService(redisAddr, redisPassword string, db int) (CacheService, error) {
	// Crear cliente Redis
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       db,
	})

	// Probar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error conectando a Redis: %w", err)
	}

	return &cacheService{
		client: client,
		ttl:    15 * time.Minute, // TTL por defecto: 15 minutos
	}, nil
}

// ========== PROPIEDADES ==========

// GetProperty - Obtiene propiedad desde caché
func (c *cacheService) GetProperty(ctx context.Context, id uint) (*models.Property, error) {
	key := c.propertyKey(id)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error obteniendo de caché", http.StatusInternalServerError)
	}

	var property models.Property
	if err := json.Unmarshal(data, &property); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error deserializando propiedad", http.StatusInternalServerError)
	}

	return &property, nil
}

// SetProperty - Guarda propiedad en caché
func (c *cacheService) SetProperty(ctx context.Context, property *models.Property, ttl time.Duration) error {
	key := c.propertyKey(property.ID)

	// Serializar propiedad
	data, err := json.Marshal(property)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error serializando propiedad", http.StatusInternalServerError)
	}

	// Guardar en caché
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error guardando en caché", http.StatusInternalServerError)
	}

	return nil
}

// InvalidateProperty - Invalida propiedad en caché
func (c *cacheService) InvalidateProperty(ctx context.Context, id uint) error {
	key := c.propertyKey(id)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error invalidando caché", http.StatusInternalServerError)
	}

	return nil
}

// ========== LISTA DE PROPIEDADES ==========

// GetPropertyList - Obtiene lista de propiedades desde caché
func (c *cacheService) GetPropertyList(ctx context.Context, filter string) (*CachedPropertyList, error) {
	key := c.propertyListKey(filter)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error obteniendo lista de caché", http.StatusInternalServerError)
	}

	var cachedList CachedPropertyList
	if err := json.Unmarshal(data, &cachedList); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error deserializando lista", http.StatusInternalServerError)
	}

	return &cachedList, nil
}

// SetPropertyList - Guarda lista de propiedades en caché
func (c *cacheService) SetPropertyList(ctx context.Context, filter string, properties []*models.Property, ttl time.Duration) error {
	key := c.propertyListKey(filter)

	cachedList := &CachedPropertyList{
		Properties: properties,
		CachedAt:   time.Now(),
	}

	// Calcular total
	total := int64(len(properties))
	cachedList.Total = total

	// Serializar
	data, err := json.Marshal(cachedList)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error serializando lista", http.StatusInternalServerError)
	}

	// Guardar en caché
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error guardando lista en caché", http.StatusInternalServerError)
	}

	return nil
}

// InvalidatePropertyList - Invalida todas las listas de propiedades
func (c *cacheService) InvalidatePropertyList(ctx context.Context) error {
	pattern := "properties:list:*"

	// Obtener todas las keys que coinciden
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	keys := []string{}

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error escaneando keys", http.StatusInternalServerError)
	}

	// Eliminar todas las keys
	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error eliminando keys", http.StatusInternalServerError)
		}
	}

	return nil
}

// ========== CATEGORÍAS ==========

// GetCategories - Obtiene categorías desde caché
func (c *cacheService) GetCategories(ctx context.Context) ([]*models.Category, error) {
	key := "categories:all"

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error obteniendo categorías de caché", http.StatusInternalServerError)
	}

	var categories []*models.Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error deserializando categorías", http.StatusInternalServerError)
	}

	return categories, nil
}

// SetCategories - Guarda categorías en caché
func (c *cacheService) SetCategories(ctx context.Context, categories []*models.Category, ttl time.Duration) error {
	key := "categories:all"

	data, err := json.Marshal(categories)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error serializando categorías", http.StatusInternalServerError)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error guardando categorías en caché", http.StatusInternalServerError)
	}

	return nil
}

// InvalidateCategories - Invalida categorías en caché
func (c *cacheService) InvalidateCategories(ctx context.Context) error {
	key := "categories:all"

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error invalidando categorías", http.StatusInternalServerError)
	}

	return nil
}

// ========== CONTADORES ==========

// IncrementVisits - Incrementa contador de visitas
func (c *cacheService) IncrementVisits(ctx context.Context, propertyID uint) (int64, error) {
	key := c.propertyVisitsKey(propertyID)

	newValue, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error incrementando visitas", http.StatusInternalServerError)
	}

	// Establecer expiry de 1 día
	c.client.Expire(ctx, key, 24*time.Hour)

	return newValue, nil
}

// IncrementContacts - Incrementa contador de contactos
func (c *cacheService) IncrementContacts(ctx context.Context, propertyID uint) (int64, error) {
	key := c.propertyContactsKey(propertyID)

	newValue, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error incrementando contactos", http.StatusInternalServerError)
	}

	// Establecer expiry de 1 día
	c.client.Expire(ctx, key, 24*time.Hour)

	return newValue, nil
}

// ========== GESTIÓN DE CACHÉ ==========

// ClearAll - Limpia todo el caché
func (c *cacheService) ClearAll(ctx context.Context) error {
	if err := c.client.FlushAll(ctx).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error limpiando caché", http.StatusInternalServerError)
	}

	return nil
}

// GetStats - Obtiene estadísticas del caché
func (c *cacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	_, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error obteniendo info de Redis", http.StatusInternalServerError)
	}

	// Parsear info para obtener keys y memory
	// (Simplificado, en producción usar INFO command parsing)

	stats := &CacheStats{
		Keys:     0,
		Memory:   0,
		Hits:     0,
		Misses:   0,
		HitRatio: 0.0,
	}

	// Obtener stats usando DBSize
	keys, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternal, "error obteniendo tamaño de BD", http.StatusInternalServerError)
	}
	stats.Keys = keys

	return stats, nil
}

// ========== HELPER FUNCTIONS ==========

// propertyKey - Genera key para propiedad individual
func (c *cacheService) propertyKey(id uint) string {
	return fmt.Sprintf("property:%d", id)
}

// propertyListKey - Genera key para lista de propiedades
func (c *cacheService) propertyListKey(filter string) string {
	return fmt.Sprintf("properties:list:%s", filter)
}

// propertyVisitsKey - Genera key para contador de visitas
func (c *cacheService) propertyVisitsKey(id uint) string {
	return fmt.Sprintf("property:%d:visits", id)
}

// propertyContactsKey - Genera key para contador de contactos
func (c *cacheService) propertyContactsKey(id uint) string {
	return fmt.Sprintf("property:%d:contacts", id)
}

// hashFilter - Genera hash de filtros para usar como key
func (c *cacheService) hashFilter(filter interface{}) string {
	// Simplificado: usar JSON como key
	// En producción usar un hash real (MD5, SHA256)
	data, _ := json.Marshal(filter)
	return string(data)
}

// ========== CACHE-ASIDE PATTERN ==========

// GetOrFetchProperty - Obiene propiedad de caché o DB
func (c *cacheService) GetOrFetchProperty(ctx context.Context, id uint, fetchFunc func() (*models.Property, error)) (*models.Property, error) {
	// 1. Intentar obtener de caché
	property, err := c.GetProperty(ctx, id)
	if err == nil && property != nil {
		return property, nil // Cache hit
	}

	// 2. Cache miss: obtener de DB
	property, err = fetchFunc()
	if err != nil {
		return nil, err
	}

	// 3. Guardar en caché
	_ = c.SetProperty(ctx, property, c.ttl)

	return property, nil
}

// GetOrFetchPropertyList - Obtiene lista de caché o DB
func (c *cacheService) GetOrFetchPropertyList(ctx context.Context, filter string, fetchFunc func() ([]*models.Property, int64, error)) (*CachedPropertyList, error) {
	// 1. Intentar obtener de caché
	cached, err := c.GetPropertyList(ctx, filter)
	if err == nil && cached != nil {
		return cached, nil // Cache hit
	}

	// 2. Cache miss: obtener de DB
	properties, total, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// 3. Guardar en caché
	_ = c.SetPropertyList(ctx, filter, properties, c.ttl)

	return &CachedPropertyList{
		Properties: properties,
		Total:      total,
		CachedAt:   time.Now(),
	}, nil
}

// ========== INVALIDACIÓN INTELIGENTE ==========

// InvalidatePropertyRelated - Invalida propiedad y listas relacionadas
func (c *cacheService) InvalidatePropertyRelated(ctx context.Context, property *models.Property) error {
	// 1. Invalidar propiedad individual
	if err := c.InvalidateProperty(ctx, property.ID); err != nil {
		return err
	}

	// 2. Invalidar listas (todas, ya que no sabemos qué filtros usar)
	if err := c.InvalidatePropertyList(ctx); err != nil {
		return err
	}

	// 3. Si está destacada, invalidar destacadas también
	if property.Destacado {
		if err := c.InvalidateFeatured(ctx); err != nil {
			return err
		}
	}

	return nil
}

// InvalidateFeatured - Invalida caché de propiedades destacadas
func (c *cacheService) InvalidateFeatured(ctx context.Context) error {
	key := "properties:featured"

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternal, "error invalidando destacadas", http.StatusInternalServerError)
	}

	return nil
}

// WarmUpCache - Calienta el caché con datos frecuentes
func (c *cacheService) WarmUpCache(ctx context.Context, fetchFuncs map[string]interface{}) error {
	for name, fetchFn := range fetchFuncs {
		switch v := fetchFn.(type) {
		case func() (*models.Property, error):
			property, err := v()
			if err == nil {
				_ = c.SetProperty(ctx, property, c.ttl)
			}

		case func() ([]*models.Property, error):
			properties, err := v()
			if err == nil {
				_ = c.SetPropertyList(ctx, name, properties, c.ttl)
			}
		}
	}

	return nil
}
