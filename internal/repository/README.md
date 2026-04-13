# Repository Layer - Data Access Layer

## 📋 Índice

- [PropertyRepository](#propertyrepository) - CRUD de propiedades
- [CategoryRepository](#categoryrepository) - CRUD de categorías
- [AuditRepository](#auditrepository) - Auditoría de acciones
- [BaseRepository](#baserepository) - Funcionalidades compartidas

---

## PropertyRepository

Interfaz completa para gestión de propiedades en BD.

```go
type PropertyRepository interface {
    // CRUD básico
    Create(ctx context.Context, property *models.Property) error
    GetByID(ctx context.Context, id uint) (*models.Property, error)
    Update(ctx context.Context, property *models.Property) error
    Delete(ctx context.Context, id uint) error

    // Listados
    List(ctx context.Context, filter *dto.PropertyFilter) ([]*models.Property, int64, error)
    ListByUser(ctx context.Context, userID uint, page, limit int) ([]*models.Property, int64, error)
    GetFeatured(ctx context.Context, limit int) ([]*models.Property, error)
    GetRelated(ctx context.Context, propertyID uint, limit int) ([]*models.Property, error)

    // Búsqueda
    Search(ctx context.Context, query string, filter *dto.PropertyFilter) ([]*models.Property, int64, error)

    // Fotos
    AddPhoto(ctx context.Context, photo *models.PropertyPhoto) error
    UpdatePhoto(ctx context.Context, photo *models.PropertyPhoto) error
    DeletePhoto(ctx context.Context, photoID uint) error
    GetPhotos(ctx context.Context, propertyID uint) ([]models.PropertyPhoto, error)
    SetPortada(ctx context.Context, propertyID, photoID uint) error
    ReorderPhotos(ctx context.Context, propertyID uint, orden []uint) error

    // Categorías
    AddCategory(ctx context.Context, propertyID, categoryID uint) error
    RemoveCategory(ctx context.Context, propertyID, categoryID uint) error
    GetCategories(ctx context.Context, propertyID uint) ([]models.Category, error)

    // Métricas
    IncrementVisits(ctx context.Context, propertyID uint) error
    IncrementContacts(ctx context.Context, propertyID uint) error

    // Utilidades
    ExistsBySlug(ctx context.Context, slug string) (bool, error)
    ExistsByID(ctx context.Context, id uint) (bool, error)
    IsOwner(ctx context.Context, propertyID, userID uint) (bool, error)
}
```

### Ejemplo de uso

```go
import "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"

// Crear repositorio
propertyRepo := repository.NewPropertyRepository(db)

// Crear propiedad
property := &models.Property{
    Titulo: "Hermosa casa",
    // ...
}
err := propertyRepo.Create(context.Background(), property)

// Obtener con eager loading
property, err := propertyRepo.GetByID(context.Background(), id)
// Incluye: Fotos (ordenadas), Categorías, User

// Listar con filtros
filter := &dto.PropertyFilter{
    TipoPropiedad: []string{"casa", "departamento"},
    Ciudad:        "Santa Cruz",
    Page:          1,
    Limit:         20,
}
properties, total, err := propertyRepo.List(context.Background(), filter)

// Buscar propiedades relacionadas
related, err := propertyRepo.GetRelated(context.Background(), propertyID, 4)
```

---

## CategoryRepository

Gestión de categorías con conteo de propiedades.

```go
type CategoryRepository interface {
    // CRUD básico
    Create(ctx context.Context, category *models.Category) error
    GetByID(ctx context.Context, id uint) (*models.Category, error)
    GetBySlug(ctx context.Context, slug string) (*models.Category, error)
    Update(ctx context.Context, category *models.Category) error
    Delete(ctx context.Context, id uint) error

    // Listados
    List(ctx context.Context, onlyActive bool) ([]*models.Category, error)
    ListWithPropertiesCount(ctx context.Context) ([]*CategoryWithCount, error)

    // Utilidades
    ExistsBySlug(ctx context.Context, slug string) (bool, error)
    ExistsByName(ctx context.Context, name string) (bool, error)
}
```

### Ejemplo de uso

```go
categoryRepo := repository.NewCategoryRepository(db)

// Listar categorías activas
categories, err := categoryRepo.List(context.Background(), true)

// Listar con contador de propiedades
categoriesWithCount, err := categoryRepo.ListWithPropertiesCount(context.Background())
// Retorna: [{Category, PropertiesCount: 15}, ...]

// Obtener por slug
category, err := categoryRepo.GetBySlug(context.Background(), "residencial")
```

---

## AuditRepository

Sistema completo de auditoría de acciones.

```go
type AuditRepository interface {
    // Log de acciones
    LogCreate(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error
    LogUpdate(ctx context.Context, userID uint, entidad string, entityID uint, before, after interface{}) error
    LogDelete(ctx context.Context, userID uint, entidad string, entityID uint, data interface{}) error
    LogAction(ctx context.Context, entry *AuditLogEntry) error

    // Consultas
    GetByUserID(ctx context.Context, userID uint, page, limit int) ([]*models.AuditLog, int64, error)
    GetByEntity(ctx context.Context, entidad string, entityID uint, page, limit int) ([]*models.AuditLog, int64, error)
    GetByAction(ctx context.Context, accion models.AuditAction, page, limit int) ([]*models.AuditLog, int64, error)
    GetByRequestID(ctx context.Context, requestID string) ([]*models.AuditLog, error)

    // Estadísticas
    GetActionStats(ctx context.Context, userID uint, startDate, endDate time.Time) (*ActionStats, error)
}
```

### Ejemplo de uso

```go
auditRepo := repository.NewAuditRepository(db)

// Log de creación
err := auditRepo.LogCreate(context.Background(), userID, "propiedad", propertyID, property)

// Log de actualización con before/after
err := auditRepo.LogUpdate(context.Background(), userID, "propiedad", propertyID, beforeProperty, afterProperty)

// Log personalizado
entry := &repository.AuditLogEntry{
    UserID:    &userID,
    Accion:    models.ActionView,
    Entidad:   models.EntityProperty,
    EntityID:  &propertyID,
    IPAddress: c.IP(),
    UserAgent: c.Get("User-Agent"),
    RequestID: c.Locals("request_id"),
}
err := auditRepo.LogAction(context.Background(), entry)

// Obtener logs por usuario
logs, total, err := auditRepo.GetByUserID(context.Background(), userID, 1, 20)

// Estadísticas
stats, err := auditRepo.GetActionStats(context.Background(), userID, startDate, endDate)
```

---

## BaseRepository

Funcionalidades compartidas para todos los repositorios.

### Transacciones

```go
baseRepo := repository.NewBaseRepository(db)

// Ejecutar en transacción
err := baseRepo.WithTransaction(context.Background(), func(tx *gorm.DB) error {
    // Múltiples operaciones
    if err := tx.Create(&property).Error; err != nil {
        return err // Rollback automático
    }

    if err := tx.Create(&photo).Error; err != nil {
        return err // Rollback automático
    }

    return nil // Commit automático
})
```

### Query Builder

```go
// Construir query compleja
result, err := baseRepo.NewQueryBuilder().
    Model(&models.Property{}).
    Where("estado = ?", models.StatusActiva).
    WhereIn("tipo_propiedad", []string{"casa", "departamento"}).
    WhereBetween("precio", 10000, 500000).
    WhereLike("ciudad", "Santa").
    Pagination(1, 20).
    OrderBy("precio", "DESC").
    OrderBy("created_at", "DESC").
    Preload("Fotos").
    Preload("Categorias").
    WithCount().
    Exec(&properties)

// Resultado tiene: Total, Page, Limit, TotalPages
```

### Bulk Operations

```go
// Crear múltiples registros
properties := []*models.Property{property1, property2, property3}
err := baseRepo.BulkCreate(context.Background(), &properties)

// Actualizar múltiples
err := baseRepo.BulkUpdate(context.Background(), &models.Property{}, map[string]interface{}{
    "destacado": true,
})

// Eliminar múltiples
ids := []uint{1, 2, 3}
err := baseRepo.BulkDelete(context.Background(), &models.Property{}, ids)
```

---

## Características Avanzadas

### Eager Loading

Todos los repositorios usan eager loading para optimizar consultas:

```go
// PropertyRepository eager loading
property, _ := propertyRepo.GetByID(ctx, id)
// Incluye automáticamente:
// - Fotos (ordenadas por orden ASC)
// - Categorías
// - User
```

### Full-Text Search

```go
// Búsqueda con PostgreSQL full-text search
properties, total, err := propertyRepo.Search(ctx, "casa piscina", filter)
```

### Transacciones Automáticas

Las operaciones complejas usan transacciones automáticamente:

```go
// SetPortada usa transacción internamente
err := propertyRepo.SetPortada(ctx, propertyID, photoID)
// 1. Desmarca todas las fotos (ROLLBACK si falla)
// 2. Marca la foto seleccionada (ROLLBACK si falla)
// 3. Commit si todo ok
```

### Optimizaciones

- **Índices**: Queries usan índices creados en migraciones
- **Preloading**: Eager loading para evitar N+1
- **Count**: Contador eficiente con SQL COUNT
- **Transactions**: ACID compliance en operaciones críticas
- **Soft Delete**: Respeto automático de deleted_at

---

## Patrones Implementados

### 1. Repository Pattern
Separación entre lógica de negocio y acceso a datos.

### 2. Interface Segregation
Cada repositorio tiene su interfaz específica.

### 3. Dependency Injection
Repos se inyectan en services via constructor.

### 4. Context Management
Todas las funciones reciben `context.Context`.

### 5. Error Handling
Errores wrappeados con contexto adicional.

---

## Ejemplo Completo: Service usando Repository

```go
type PropertyService struct {
    propertyRepo repository.PropertyRepository
    auditRepo    repository.AuditRepository
}

func NewPropertyService(
    propertyRepo repository.PropertyRepository,
    auditRepo repository.AuditRepository,
) *PropertyService {
    return &PropertyService{
        propertyRepo: propertyRepo,
        auditRepo:    auditRepo,
    }
}

func (s *PropertyService) CreateProperty(ctx context.Context, req *dto.CreatePropertyRequest, userID uint) (*dto.PropertyResponse, error) {
    // 1. Crear modelo
    property := req.ToModel(userID)

    // 2. Guardar en BD
    if err := s.propertyRepo.Create(ctx, property); err != nil {
        return nil, err
    }

    // 3. Log de auditoría
    if err := s.auditRepo.LogCreate(ctx, userID, "propiedad", property.ID, property); err != nil {
        // Log no debe fallar el create
        log.Printf("error loggeando creación: %v", err)
    }

    // 4. Retornar response
    return dto.MapPropertyToResponse(property, false), nil
}
```
