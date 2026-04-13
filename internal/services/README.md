# Services Layer - Business Logic

Capa de lógica de negocio que coordina repositorios y aplica reglas del negocio.

## 📋 Servicios Implementados

### PropertyService
Lógica de negocio completa para gestión de propiedades.

```go
type PropertyService interface {
    // CRUD
    CreateProperty(ctx, req, userID) (*PropertyResponse, error)
    GetProperty(ctx, id, includeContacto) (*PropertyResponse, error)
    UpdateProperty(ctx, id, req, userID) (*PropertyResponse, error)
    DeleteProperty(ctx, id, userID) error

    // Listados
    ListProperties(ctx, filter) (*PropertyListResponse, error)
    ListUserProperties(ctx, userID, page, limit) (*PropertyListResponse, error)
    GetFeaturedProperties(ctx, limit) ([]*PropertyResponse, error)
    GetRelatedProperties(ctx, id, limit) ([]*PropertyResponse, error)

    // Fotos
    UploadPhotos(ctx, id, req, userID) (*PropertyResponse, error)
    SetPhotoAsPortada(ctx, id, photoID, userID) error
    ReorderPhotos(ctx, id, orden, userID) error
    DeletePhoto(ctx, id, photoID, userID) error

    // Búsqueda
    SearchProperties(ctx, query, filter) (*PropertyListResponse, error)

    // Categorías
    AddCategory(ctx, id, categoryID, userID) error
    RemoveCategory(ctx, id, categoryID, userID) error
}
```

#### Ejemplo de uso

```go
import "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/services"

// Crear servicio
propertyService := services.NewPropertyService(
    propertyRepo,
    categoryRepo,
    auditRepo,
)

// Crear propiedad
req := &dto.CreatePropertyRequest{
    Titulo:      "Hermosa casa",
    Descripcion: "Descripción larga...",
    Precio:      ptrFloat64(85000),
    Ciudad:      "Santa Cruz",
}
response, err := propertyService.CreateProperty(context.Background(), req, userID)

// Listar propiedades
filter := &dto.PropertyFilter{
    TipoPropiedad: []string{"casa"},
    Ciudad:        "Santa Cruz",
    Page:          1,
    Limit:         20,
}
list, err := propertyService.ListProperties(context.Background(), filter)
```

### SearchService
Búsqueda avanzada y autocompletado.

```go
type SearchService interface {
    // Búsqueda principal
    Search(ctx, query, filter) (*PropertyListResponse, error)

    // Búsquedas específicas
    SearchByLocation(ctx, ciudad, zona, filter) (*PropertyListResponse, error)
    SearchByPriceRange(ctx, min, max, filter) (*PropertyListResponse, error)
    SearchByType(ctx, tipos, filter) (*PropertyListResponse, error)

    // Autocomplete
    AutocompleteCities(ctx, query) ([]string, error)
    AutocompleteZones(ctx, ciudad, query) ([]string, error)

    // Sugerencias
    GetSuggestedProperties(ctx, userID, limit) ([]*PropertyResponse, error)
    GetTrendingProperties(ctx, limit) ([]*PropertyResponse, error)
}
```

---

## 🔄 Flujo de Operaciones

### Crear Propiedad

```
1. Validar request (validators)
   ↓
2. Crear modelo desde DTO
   ↓
3. Generar slug único (slug generator)
   ↓
4. Generar Google Maps URL (utils)
   ↓
5. Guardar en BD (propertyRepo.Create)
   ↓
6. Añadir categorías si se especificaron
   ↓
7. Log de auditoría (auditRepo.LogCreate)
   ↓
8. Retornar response con relaciones
```

### Actualizar Propiedad

```
1. Verificar propiedad existe
   ↓
2. Verificar dueño (permission check)
   ↓
3. Validar request
   ↓
4. Guardar estado anterior (audit trail)
   ↓
5. Aplicar updates (applyUpdates helper)
   ↓
6. Regenerar slug si cambió título
   ↓
7. Actualizar Google Maps URL si cambiaron coordenadas
   ↓
8. Guardar cambios (propertyRepo.Update)
   ↓
9. Log de auditoría with before/after
   ↓
10. Recargar y retornar response
```

### Upload de Fotos

```
1. Verificar propiedad existe
   ↓
2. Verificar dueño
   ↓
3. Validar cantidad (3-7 fotos)
   ↓
4. Validar tamaño (< 5MB cada una)
   ↓
5. Validar formato (JPEG, PNG, WEBP)
   ↓
6. Procesar imágenes (resize, thumbnail)
   ↓
7. Subir a storage (R2/local)
   ↓
8. Crear registros en BD
   ↓
9. Marcar primera como portada si no hay
   ↓
10. Recargar y retornar response
```

---

## 🎨 Validaciones de Negocio

### Validaciones en CreateProperty:
- ✅ Campos requeridos presentes
- ✅ Título mínimo 10 caracteres
- ✅ Descripción mínimo 50 caracteres
- ✅ Tipo y operación válidos
- ✅ Precio > 0
- ✅ Coordenadas en rangos válidos
- ✅ Si hay amenities, debe haber metraje construido
- ✅ Si hay latitud, debe haber longitud

### Validaciones en UpdateProperty:
- ✅ Al menos un campo presente
- ✅ Transiciones de estado válidas
- ✅ Usuario es dueño

### Validaciones en UploadPhotos:
- ✅ Mínimo 1 foto, máximo 7
- ✅ Tamaño máximo 5MB por foto
- ✅ Formatos válidos: JPEG, PNG, WEBP
- ✅ Warning si < 3 fotos (no bloquea)

---

## 📊 Sistema de Errores

Uso de `pkg/errors` para errores profesionales:

```go
import apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"

// Error de validación
return nil, apperrors.ValidationErrors(details)

// Recurso no encontrado
return nil, apperrors.PropertyNotFound(id)

// No autorizado
return nil, apperrors.Unauthorized()

// Prohibido (no es dueño)
return nil, apperrors.NotOwner("propiedad")

// Error de negocio
return nil, apperrors.InvalidPrice("El precio debe ser positivo")

// Error de base de datos
return nil, apperrors.DatabaseError(err)

// Error interno
return nil, apperrors.InternalError(err)
```

### Mapping de HTTP Status

| Error Code              | HTTP Status |
|-------------------------|-------------|
| VALIDATION_ERROR        | 400         |
| INVALID_INPUT           | 400         |
| NOT_FOUND               | 404         |
| UNAUTHORIZED            | 401         |
| FORBIDDEN               | 403         |
| ALREADY_EXISTS          | 409         |
| DATABASE_ERROR          | 500         |
| INTERNAL_ERROR          | 500         |

---

## 🔍 Características de Búsqueda

### Full-Text Search
```go
// Búsqueda con PostgreSQL full-text search
results, err := searchService.Search(ctx, "casa piscina", filter)
// WHERE to_tsvector('spanish', titulo || ' ' || descripcion) 
// @@ to_tsquery('spanish', 'casa & piscina')
```

### Búsqueda por Ubicación
```go
// Búsqueda por ciudad y/o zona
results, err := searchService.SearchByLocation(ctx, "Santa Cruz", "Equipetrol", filter)
```

### Búsqueda por Rango de Precio
```go
// Propiedades entre $50,000 y $100,000
results, err := searchService.SearchByPriceRange(ctx, 50000, 100000, filter)
```

### Autocomplete
```go
// Autocomplete de ciudades
cities, err := searchService.AutocompleteCities(ctx, "San")
// ["Santa Cruz de la Sierra", "San Javier", ...]

// Autocomplete de zonas
zones, err := searchService.AutocompleteZones(ctx, "Santa Cruz", "Eq")
// ["Equipetrol", "El Urubó", ...]
```

### Score de Relevancia
```go
// Calcula score de relevancia para sorting
score := CalculateRelevanceScore(property, query)
// +50 Título coincide
// +20 Descripción coincide
// +10 Ciudad coincide
// +15 Destacado
// +0-10 Visitas (normalizado)
```

---

## 💡 Patrones de Diseño

### 1. Service Layer Pattern
```go
type propertyService struct {
    propertyRepo repository.PropertyRepository
    categoryRepo repository.CategoryRepository
    auditRepo    repository.AuditRepository
    validator    *PropertyValidator
}

// Inyección de dependencias
func NewPropertyService(
    propertyRepo repository.PropertyRepository,
    categoryRepo repository.CategoryRepository,
    auditRepo repository.AuditRepository,
) PropertyService {
    return &propertyService{
        propertyRepo: propertyRepo,
        categoryRepo: categoryRepo,
        auditRepo:    auditRepo,
        validator:    NewPropertyValidator(),
    }
}
```

### 2. Error Wrapping
```go
// Wrappear errores con contexto
return nil, apperrors.Wrap(err, apperrors.ErrCodeDatabase, "creando propiedad", http.StatusInternalServerError)
```

### 3. Validation Chain
```go
// 1. Validar request
if errs := validator.ValidateCreateProperty(req); len(errs) > 0 {
    return nil, apperrors.ValidationErrors(details)
}

// 2. Validar negocio
if !property.IsOwnedBy(userID) {
    return nil, apperrors.NotOwner("propiedad")
}

// 3. Validar estado
if property.Estado != StatusActiva {
    return nil, apperrors.InvalidState("Propiedad no está activa")
}
```

### 4. Audit Trail
```go
// Log de creación
_ = auditRepo.LogCreate(ctx, userID, "propiedad", property.ID, property)

// Log de actualización con before/after
_ = auditRepo.LogUpdate(ctx, userID, "propiedad", id, beforeCopy, afterCopy)

// Log personalizado
_ = auditRepo.LogAction(ctx, &AuditLogEntry{
    UserID: &userID,
    Accion: models.ActionView,
    Entidad: models.EntityProperty,
    EntityID: &propertyID,
    Metadata: map[string]interface{}{
        "source": "web",
        "device": "mobile",
    },
})
```

---

## 🚀 Optimizaciones

### Eager Loading
```go
// Cargar relaciones en una sola query
property, err := propertyRepo.GetByID(ctx, id)
// Incluye automáticamente:
// - Fotos (ordenadas)
// - Categorías
// - User
```

### Slug Generation
```go
// Generar slug único automáticamente
slug := slugGenerator.GenerateUnique(title, func(slug string) bool {
    exists, _ := propertyRepo.ExistsBySlug(ctx, slug)
    return exists
})
// "hermosa-casa" → "hermosa-casa-2" → "hermosa-casa-3"
```

### Smart Updates
```go
// Actualizar solo campos que cambiaron
if req.Titulo != nil {
    property.Titulo = *req.Titulo
}
// Regenerar slug solo si cambió título
if req.Titulo != nil && *req.Titulo != oldTitle {
    property.Slug = generateUniqueSlug(*req.Titulo)
}
```

### Transaction Safety
```go
// Operaciones atómicas en repositorio
err := propertyRepo.SetPortada(ctx, propertyID, photoID)
// Internamente usa transacción:
// BEGIN TRANSACTION
// UPDATE fotos SET es_portada = false WHERE property_id = ?
// UPDATE fotos SET es_portada = true WHERE id = ?
// COMMIT (o ROLLBACK si falla)
```
