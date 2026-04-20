package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient - Cliente de Redis con caché y pub/sub
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedis - Crea una nueva conexión a Redis
func NewRedis() (*RedisClient, error) {
	// Obtener configuración desde variables de entorno
	var redisAddr string
	var redisPassword string

	// Intentar obtener desde REDIS_URL (formato: redis://[password@]host:port)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		// Parsear URL de Redis
		// Formato esperado: redis://[:password@]host:port
		parsed := redisURL
		if len(parsed) > 8 { // "redis://" tiene 8 caracteres
			parsed = parsed[8:] // Remover "redis://"
		}

		// Extraer password si existe
		atIndex := -1
		for i, c := range parsed {
			if c == '@' {
				atIndex = i
				break
			}
		}

		if atIndex > 0 {
			redisPassword = parsed[:atIndex]
			redisAddr = parsed[atIndex+1:]
		} else if atIndex == 0 {
			// Caso "@host:port" (sin password)
			redisAddr = parsed[1:]
		} else {
			redisAddr = parsed
		}
	} else {
		// Usar REDIS_ADDR (formato: host:port)
		redisAddr = os.Getenv("REDIS_ADDR")
		redisPassword = os.Getenv("REDIS_PASSWORD")
	}

	// Valor por defecto para desarrollo
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisDB := 0

	// Crear cliente Redis
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()

	// Probar conexión
	maxRetries := 5
	retryDelay := 2 * time.Second

	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = client.Ping(ctx).Result()
		if err == nil {
			log.Println("✅ Conectado a Redis exitosamente")
			return &RedisClient{
				Client: client,
				Ctx:    ctx,
			}, nil
		}

		if i < maxRetries-1 {
			log.Printf("⏳ Intento %d/%d: No se pudo conectar a Redis. Reintentando en %v...",
				i+1, maxRetries, retryDelay)
			log.Printf("   Error: %v", err)
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("error conectando a Redis después de %d intentos: %w", maxRetries, err)
}

// Close - Cierra la conexión a Redis
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// CacheProperty - Guarda una propiedad en caché (5 minutos)
func (r *RedisClient) CacheProperty(propertyID uint, propertyData interface{}) error {
	key := fmt.Sprintf("property:%d", propertyID)
	return r.Client.Set(r.Ctx, key, propertyData, 5*time.Minute).Err()
}

// GetCachedProperty - Obtiene una propiedad desde la caché
func (r *RedisClient) GetCachedProperty(propertyID uint, dest interface{}) error {
	key := fmt.Sprintf("property:%d", propertyID)
	return r.Client.Get(r.Ctx, key).Scan(dest)
}

// InvalidateProperty - Elimina una propiedad de la caché
func (r *RedisClient) InvalidateProperty(propertyID uint) error {
	key := fmt.Sprintf("property:%d", propertyID)
	return r.Client.Del(r.Ctx, key).Err()
}

// CachePropertyList - Guarda una lista de propiedades en caché (2 minutos)
func (r *RedisClient) CachePropertyList(key string, data interface{}) error {
	cacheKey := fmt.Sprintf("properties:%s", key)
	return r.Client.Set(r.Ctx, cacheKey, data, 2*time.Minute).Err()
}

// GetCachedPropertyList - Obtiene una lista de propiedades desde la caché
func (r *RedisClient) GetCachedPropertyList(key string, dest interface{}) error {
	cacheKey := fmt.Sprintf("properties:%s", key)
	return r.Client.Get(r.Ctx, cacheKey).Scan(dest)
}

// InvalidatePropertyLists - Elimina todas las listas de propiedades de la caché
func (r *RedisClient) InvalidatePropertyLists() error {
	iter := r.Client.Scan(r.Ctx, 0, "properties:*", 0).Iterator()
	var keys []string

	for iter.Next(r.Ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.Client.Del(r.Ctx, keys...).Err()
	}

	return nil
}

// PublishPropertyUpdate - Publica una actualización de propiedad para tiempo real
func (r *RedisClient) PublishPropertyUpdate(action string, propertyID uint, data interface{}) error {
	channel := "property_updates"
	message := map[string]interface{}{
		"action":      action, // created, updated, deleted, status_changed
		"property_id": propertyID,
		"data":        data,
		"timestamp":   time.Now().Unix(),
	}

	return r.Client.Publish(r.Ctx, channel, message).Err()
}

// SubscribeToPropertyUpdates - Se suscribe a actualizaciones de propiedades en tiempo real
func (r *RedisClient) SubscribeToPropertyUpdates() (*redis.PubSub, error) {
	channel := "property_updates"
	pubsub := r.Client.Subscribe(r.Ctx, channel)

	// Verificar suscripción
	_, err := pubsub.Receive(r.Ctx)
	if err != nil {
		return nil, fmt.Errorf("error suscribiéndose a canal: %w", err)
	}

	return pubsub, nil
}

// CacheUserPropertyCount - Guarda el conteo de propiedades de un usuario (1 minuto)
func (r *RedisClient) CacheUserPropertyCount(userID uint, count int64) error {
	key := fmt.Sprintf("user:%d:property_count", userID)
	return r.Client.Set(r.Ctx, key, count, 1*time.Minute).Err()
}

// GetCachedUserPropertyCount - Obtiene el conteo de propiedades de un usuario desde la caché
func (r *RedisClient) GetCachedUserPropertyCount(userID uint) (int64, error) {
	key := fmt.Sprintf("user:%d:property_count", userID)
	val, err := r.Client.Get(r.Ctx, key).Int64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

// InvalidateUserPropertyCount - Elimina el conteo de propiedades de un usuario de la caché
func (r *RedisClient) InvalidateUserPropertyCount(userID uint) error {
	key := fmt.Sprintf("user:%d:property_count", userID)
	return r.Client.Del(r.Ctx, key).Err()
}
