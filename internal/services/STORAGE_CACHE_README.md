# Storage y Caching - Servicios de Infraestructura

## 📋 Índice

- [StorageService](#storageservice) - Almacenamiento de imágenes
- [CacheService](#cacheservice) - Caché con Redis

---

## StorageService

Sistema de almacenamiento de imágenes local (adaptable a Cloudflare R2).

```go
type StorageService interface {
    // Upload
    UploadPropertyPhoto(ctx, propertyID, file, orden) (*PhotoUploadResult, error)
    UploadPropertyPhotos(ctx, propertyID, files) (*PhotoUploadResult, error)

    // Procesamiento
    ProcessImage(src io.Reader) (*ProcessedImages, error)
    GenerateThumbnail(src io.Reader, width, height int) ([]byte, string, error)

    // Gestión
    DeletePhoto(ctx, url) error
    PhotoExists(ctx, url) (bool, error)

    // URLs
    GetPublicURL(propertyID, filename) string
    GetThumbnailURL(propertyID, filename) string
}
```

### Ejemplo de uso

```go
import "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"

// Crear servicio
storageService := services.NewStorageService(
    "/uploads",           // Directorio local
    "http://localhost:3001/uploads", // URL base pública
)

// Upload desde PhotoFile
result, err := storageService.UploadPropertyPhoto(ctx, propertyID, &dto.PhotoFile{
    Filename: "casa.jpg",
    Size:     1024 * 1024, // 1MB
    Content:  fileBytes,
    MimeType: "image/jpeg",
}, 0)

// Resultado:
// URL:           /uploads/propiedades/123/123_1234567890.jpg
// URLThumbnail:  /uploads/propiedades/123/123_1234567890_thumb.jpg
// Width:         1920
// Height:        1080
// FileSize:      245KB (comprimido)
// ThumbnailSize: 15KB
```

### Procesamiento de Imágenes

```go
// Procesamiento automático:
processed, err := storageService.ProcessImage(imageReader)

// 1. Decodificar imagen
// 2. Redimensionar si > 1920x1080 (manteniendo aspect ratio)
// 3. Comprimir con calidad 85%
// 4. Generar thumbnail 300x300px (cover fit)

// Formatos soportados:
// - JPEG (calidad 85%)
// - PNG (sin pérdida)
// - WEBP (calidad 85%)
```

### Validaciones

```go
// Validaciones automáticas:
✅ Tamaño máximo: 5MB por foto
✅ Formatos: JPEG, PNG, WEBP
✅ Dimensiones mínimas: 800x600px
✅ Contenido coincide con MIME type
✅ Magic bytes validados

// Errores específicos:
apperrors.FileTooLarge(5)          // "El archivo excede el tamaño máximo de 5 MB"
apperrors.InvalidFileFormat([...])  // "Formato inválido. Solo JPEG, PNG y WEBP"
```

### Adaptación a Cloudflare R2

```go
// El interface es el mismo para R2
// Solo cambia la implementación interna:

type r2StorageService struct {
    client *s3.Client // AWS SDK S3 client (compatible con R2)
    bucket string
}

func (s *r2StorageService) UploadPropertyPhoto(ctx, propertyID uint, file *dto.PhotoFile, orden int) (*PhotoUploadResult, error) {
    // Subir a R2 en lugar de disco local
    // El resto del código es idéntico
    ...
}
```

---

## CacheService

Sistema de caché distribuido con Redis.

```go
type CacheService interface {
    // Propiedades
    GetProperty(ctx, id) (*models.Property, error)
    SetProperty(ctx, property, ttl) error
    InvalidateProperty(ctx, id) error

    // Listas
    GetPropertyList(ctx, filter) (*CachedPropertyList, error)
    SetPropertyList(ctx, filter, properties, ttl) error
    InvalidatePropertyList(ctx) error

    // Categorías
    GetCategories(ctx) ([]*models.Category, error)
    SetCategories(ctx, categories, ttl) error
    InvalidateCategories(ctx) error

    // Contadores
    IncrementVisits(ctx, propertyID) (int64, error)
    IncrementContacts(ctx, propertyID) (int64, error)

    // Gestión
    ClearAll(ctx) error
    GetStats(ctx) (*CacheStats, error)
}
```

### Ejemplo de uso

```go
// Crear servicio
cacheService, _ := services.NewCacheService(
    "localhost:6379", // Redis address
    "",                // Password
    0,                 // DB number
)

// Guardar en caché
err := cacheService.SetProperty(ctx, property, 1*time.Hour)
// TTL: 1 hora

// Obtener de caché
property, err := cacheService.GetProperty(ctx, id)
if err == nil && property != nil {
    // Cache hit - retornar sin query a BD
    return property
}
// Cache miss - obtener de BD

// Cache-aside pattern automático
property, err := cacheService.GetOrFetchProperty(ctx, id, func() (*models.Property, error) {
    return propertyRepo.GetByID(ctx, id)
})
// 1. Intenta caché
// 2. Si miss: llama a la función
// 3. Guarda resultado en caché
// 4. Retorna resultado
```

### TTLs Configurados

| Data | TTL | Justificación |
|------|-----|---------------|
| Propiedad individual | 1 hora | Cambia poco, se accede mucho |
| Lista de propiedades | 15 min | Cambia frecuentemente (nuevas) |
| Categorías | 24 horas | Cambia muy poco |
| Contadores (visitas/contactos) | 1 día | Métricas temporales |

### Invalidación Inteligente

```go
// Invalida propiedad y todo lo relacionado
err := cacheService.InvalidatePropertyRelated(ctx, property)

// Invalida:
✅ property:123 (individual)
✅ properties:list:* (todas las listas)
✅ properties:featured (si está destacada)

// Estrategia: Invalidar todo es más barato que calcular keys exactas
```

### Cache Hits vs Misses

```go
// Estadísticas del caché
stats, err := cacheService.GetStats(ctx)

// Resultado:
{
    "keys": 1500,
    "memory": 52428800, // 50MB
    "hits": 8500,
    "misses": 1500,
    "hit_ratio": 0.85  // 85% de hits!
}
```

---

## Integración con PropertyService

### Upload con Caching

```go
// En PropertyService.UploadPhotos():
func (s *PropertyService) UploadPhotos(ctx context.Context, id uint, req *dto.UploadPhotosRequest, userID uint) (*dto.PropertyResponse, error) {
    // 1. Verificar propiedad y dueño
    property, err := s.propertyRepo.GetByID(ctx, id)
    
    // 2. Subir fotos (storage service)
    for i, file := range req.Files {
        result, err := s.storageService.UploadPropertyPhoto(ctx, id, &file, i)
        // - Procesa imagen (resize + thumbnail)
        // - Guarda en disco local
        // - Retorna URLs
        
        // 3. Crear registro en BD
        photo := &models.PropertyPhoto{
            PropertyID:   id,
            URL:          result.URL,
            URLThumbnail: result.URLThumbnail,
            Orden:        i,
            FileName:     result.FileName,
            FileSize:     result.FileSize,
            Width:        result.Width,
            Height:       result.Height,
        }
        s.propertyRepo.AddPhoto(ctx, photo)
    }
    
    // 4. Invalidar caché
    _ = s.cacheService.InvalidateProperty(ctx, id)
    _ = s.cacheService.InvalidatePropertyList(ctx)
    
    // 5. Retornar
    return dto.MapPropertyToResponse(property, false), nil
}
```

### Get con Cache-Aside

```go
// En PropertyService.GetProperty():
func (s *PropertyService) GetProperty(ctx context.Context, id uint, includeContacto bool) (*dto.PropertyResponse, error) {
    // 1. Intentar caché primero
    property, err := s.cacheService.GetOrFetchProperty(ctx, id, func() (*models.Property, error) {
        // 2. Cache miss: obtener de BD
        return s.propertyRepo.GetByID(ctx, id)
    })
    
    if err != nil {
        return nil, err
    }
    
    // 3. Incrementar visitas (en caché)
    _, _ = s.cacheService.IncrementVisits(ctx, id)
    _, _ = s.propertyRepo.IncrementVisits(ctx, id) // En paralelo para BD
    
    // 4. Mapear a response
    return dto.MapPropertyToResponse(property, includeContacto), nil
}
```

---

## Optimizaciones

### Image Processing

```go
// Ahorro de espacio:
- Original: No se guarda (se descarta)
- Procesada: 1920x1080 max, calidad 85%
- Thumbnail: 300x300, calidad 80%

// Ejemplo de ahorro:
Original: 5MB (sin procesar)
Procesada: 200KB
Thumbnail: 15KB
Total: 215KB (95% de ahorro)
```

### Redis Memory Usage

```go
// Estimación de memoria por key:
- Propiedad individual: ~5KB (JSON)
- Lista de propiedades: ~50KB (20 items)
- Categorías: ~2KB (5 categorías)

// Con 1000 propiedades:
- 1000 propiedades: 5MB
- 100 listas promedio: 5MB
- Categorías: 2KB
Total: ~10MB en Redis
```

### Performance Gains

```go
// Sin caché (BD):
GET /api/propiedades/123
- DB Query: 50ms
- Eager loading: 30ms
Total: 80ms

// Con caché (Redis):
GET /api/propiedades/123
- Redis GET: 2ms
- JSON Unmarshal: 1ms
Total: 3ms (96% más rápido!)

// Para listados (más impacto):
GET /api/propiedades?page=1&limit=20
- BD: 200ms
- Caché: 5ms
40x más rápido
```

---

## Estrategia de Deploy

### Desarrollo (Local)

```bash
# 1. Crear directorio de upload
mkdir -p uploads/propiedades

# 2. Iniciar Redis con Docker
docker run -d -p 6379:6379 redis:7-alpine

# 3. Configurar .env
UPLOAD_DIR=./uploads
UPLOAD_PUBLIC_URL=http://localhost:3001/uploads
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Producción (Cloudflare R2)

```go
// Cambiar implementación:
storageService, _ := services.NewR2StorageService(
    os.Getenv("R2_ACCOUNT_ID"),
    os.Getenv("R2_ACCESS_KEY"),
    os.Getenv("R2_SECRET_KEY"),
    os.Getenv("R2_BUCKET"),
    os.Getenv("R2_PUBLIC_URL"),
)

// El resto del código no cambia
```

### Redis en Producción

```yaml
# docker-compose.yml
services:
  api:
    environment:
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
  
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
```

---

## Tips y Best Practices

### 1. Siempre usar Cache-Aside
```go
// ✅ Bien
property, err := cacheService.GetOrFetchProperty(ctx, id, fetchFunc)

// ❌ Mal (manual)
property, _ := cacheService.GetProperty(ctx, id)
if property == nil {
    property, _ = fetchFunc()
    cacheService.SetProperty(ctx, property, ttl)
}
```

### 2. Invalidar en Cascada
```go
// Al actualizar propiedad
err := cacheService.InvalidatePropertyRelated(ctx, property)
// Invalida todo lo relacionado para evitar datos stale

// Es más barato invalidar todo que calcular exactamente qué cambiar
```

### 3. Usar TTLs Apropiados
```go
// Datos que cambian poco: TTL largo (1-24 horas)
// Datos que cambian mucho: TTL corto (5-15 minutos)
// Contadores: TTL medio (1 día)

// Mejor caché con datos ligeramente stale que sin caché
```

### 4. Monitorear Hit Ratio
```go
stats, _ := cacheService.GetStats(ctx)

if stats.HitRatio < 0.5 {
    // Hit ratio < 50%: Revisar estrategia de caché
    // - Aumentar TTLs
    // - Cachear más datos
    // - Revisar patrones de acceso
}
```

### 5. Cleanup Jobs
```go
// Cron job para limpiar archivos antiguos
go func() {
    for range time.Tick(24 * time.Hour) {
        services.CleanupOldFiles(uploadDir, 30) // 30 días
    }
}()
```
