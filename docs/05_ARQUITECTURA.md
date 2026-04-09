# 🏗️ Arquitectura Técnica - SaaS Inmobiliario 365Soft

## 🎯 Visión General

Este documento define la arquitectura técnica del backend, patrones de diseño, estructura de código y mejores prácticas.

---

## 📐 Arquitectura General

### Patrón de Arquitectura

**Arquitectura en capas (Layered Architecture):**

```
┌─────────────────────────────────────────┐
│         Presentation Layer               │
│      (Handlers - Fiber Routes)          │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Business Logic Layer            │
│         (Services)                      │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Data Access Layer               │
│      (Repositories - GORM)              │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Database Layer                  │
│      (PostgreSQL + Redis)               │
└─────────────────────────────────────────┘
```

### Patrones Implementados

1. **Repository Pattern**: Separar lógica de negocio del acceso a datos
2. **Service Layer**: Contener la lógica de negocio
3. **DTO Pattern**: Controlar exposición de datos en API
4. **Dependency Injection**: Inyectar dependencias en services/handlers
5. **Middleware Chain**: Auth → Roles → Audit → Handler

---

## 📁 Estructura de Directorios

```
Inmobiliaria_Backend_365Soft/
├── cmd/
│   ├── api/
│   │   └── main.go                    # Entry point API
│   ├── worker/
│   │   └── main.go                    # Background jobs
│   └── seed/
│       └── main.go                    # Data seeders
│
├── internal/
│   ├── config/
│   │   ├── database.go                # PostgreSQL config
│   │   ├── redis.go                   # Redis config
│   │   ├── storage.go                 # Cloudflare R2 config
│   │   └── config.go                  # General config
│   │
│   ├── models/
│   │   ├── user.go                    # Usuario model
│   │   ├── property.go                # Property models
│   │   ├── publication.go             # Publication models
│   │   ├── payment.go                 # Payment models
│   │   ├── client.go                  # CRM models
│   │   ├── audit.go                   # Audit models
│   │   └── notification.go            # Notification models
│   │
│   ├── handlers/
│   │   ├── auth_handler.go            # Auth endpoints
│   │   ├── property_handler.go        # Property endpoints
│   │   ├── publication_handler.go     # Publication endpoints
│   │   ├── payment_handler.go         # Payment endpoints
│   │   ├── client_handler.go          # CRM endpoints
│   │   ├── admin_handler.go           # Admin endpoints
│   │   └── search_handler.go          # Search endpoints
│   │
│   ├── services/
│   │   ├── auth_service.go            # Auth logic
│   │   ├── property_service.go        # Property logic
│   │   ├── publication_service.go     # Publication logic
│   │   ├── payment_service.go         # Payment logic (Bisa)
│   │   ├── crm_service.go             # CRM logic
│   │   ├── search_service.go          # Search logic
│   │   ├── storage_service.go         # File upload logic
│   │   ├── cache_service.go           # Redis cache logic
│   │   ├── notification_service.go    # Notification logic
│   │   └── audit_service.go           # Audit logic
│   │
│   ├── repository/
│   │   ├── user_repo.go               # User data access
│   │   ├── property_repo.go           # Property data access
│   │   ├── publication_repo.go        # Publication data access
│   │   ├── payment_repo.go            # Payment data access
│   │   ├── client_repo.go             # Client data access
│   │   └── notification_repo.go       # Notification data access
│   │
│   ├── middleware/
│   │   ├── auth.go                    # JWT validation
│   │   ├── role.go                    # Role authorization
│   │   ├── audit.go                   # Audit logging
│   │   ├── rate_limit.go              # Rate limiting
│   │   └── cors.go                    # CORS configuration
│   │
│   ├── dto/
│   │   ├── auth_dto.go                # Auth DTOs
│   │   ├── property_dto.go            # Property DTOs
│   │   ├── publication_dto.go         # Publication DTOs
│   │   ├── payment_dto.go             # Payment DTOs
│   │   └── client_dto.go              # Client DTOs
│   │
│   ├── validators/
│   │   ├── property_validator.go      # Property validations
│   │   ├── user_validator.go          # User validations
│   │   └── payment_validator.go       # Payment validations
│   │
│   ├── workers/
│   │   ├── publication_worker.go      # Renewal reminders
│   │   ├── payment_worker.go          # Payment verification
│   │   └── notification_worker.go     # Notification sender
│   │
│   └── utils/
│       ├── slug.go                    # Slug generation
│       ├── currency.go                # Currency formatting
│       ├── image.go                   # Image processing
│       ├── maps.go                    # Google Maps API
│       └── helpers.go                 # Helper functions
│
├── pkg/
│   ├── errors/
│   │   └── errors.go                  # Custom errors
│   ├── response/
│   │   └── response.go                # HTTP responses
│   └── logger/
│       └── logger.go                  # Structured logging
│
├── docs/
│   ├── 00_INDEX_DOCUMENTACION.md      # This file
│   ├── 01_REQUERIMIENTOS_FUNCIONALES.md
│   ├── 02_REQUERIMIENTOS_NO_FUNCIONALES.md
│   ├── 03_MODELO_DE_DATOS.md
│   ├── 04_API_ENDPOINTS.md
│   ├── 05_ARQUITECTURA.md
│   ├── 06_FLUJOS_DE_NEGOCIO.md
│   └── modules/
│       └── ...                        # Módulo-specific docs
│
├── tests/
│   ├── unit/
│   │   ├── property_service_test.go
│   │   └── payment_service_test.go
│   └── integration/
│       └── api_test.go
│
├── .env                                # Environment variables
├── .env.example                        # Environment template
├── .gitignore
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## 🔧 Stack Tecnológico

### Backend
- **Lenguaje:** Go 1.26.1
- **Framework:** Fiber v2.52.12 (Express-inspired web framework)
- **ORM:** GORM v1.31.1
- **Validación:** go-playground/validator v10
- **Auth:** golang-jwt/jwt v5.3.1

### Base de Datos
- **Principal:** PostgreSQL 15+
- **Cache:** Redis 7+
- **ORM:** GORM con driver pgx/v5

### Storage
- **Object Storage:** Cloudflare R2 (S3-compatible)
- **CDN:** Cloudflare

### Pagos
- **Método:** Banco Bisa QR
- **Webhooks:** HTTP POST endpoints

### Infrastructure
- **Container:** Docker
- **Orchestration:** docker-compose (dev), Kubernetes (prod)
- **Reverse Proxy:** Nginx

---

## 🎨 Patrones de Diseño

### 1. Repository Pattern

**Propósito:** Separar la lógica de negocio del acceso a datos.

```go
// internal/repository/property_repo.go
type PropertyRepository interface {
    Create(ctx context.Context, property *models.Property) error
    GetByID(ctx context.Context, id uint) (*models.Property, error)
    List(ctx context.Context, filter PropertyFilter) ([]*models.Property, int64, error)
    Update(ctx context.Context, property *models.Property) error
    Delete(ctx context.Context, id uint) error
}

type propertyRepository struct {
    db *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) PropertyRepository {
    return &propertyRepository{db: db}
}

func (r *propertyRepository) Create(ctx context.Context, property *models.Property) error {
    return r.db.WithContext(ctx).Create(property).Error
}
```

### 2. Service Layer

**Propósito:** Contener la lógica de negocio y orquestar repositorios.

```go
// internal/services/property_service.go
type PropertyService interface {
    CreateProperty(ctx context.Context, req *dto.CreatePropertyRequest) (*dto.PropertyResponse, error)
    GetProperty(ctx context.Context, id uint) (*dto.PropertyResponse, error)
    ListProperties(ctx context.Context, filter dto.PropertyFilter) (*dto.PropertyListResponse, error)
}

type propertyService struct {
    repo   repository.PropertyRepository
    cache  cache.CacheService
    audit  audit.AuditService
}

func NewPropertyService(
    repo repository.PropertyRepository,
    cache cache.CacheService,
    audit audit.AuditService,
) PropertyService {
    return &propertyService{
        repo:  repo,
        cache: cache,
        audit: audit,
    }
}

func (s *propertyService) CreateProperty(ctx context.Context, req *dto.CreatePropertyRequest) (*dto.PropertyResponse, error) {
    // 1. Validar
    if err := validators.ValidateCreateProperty(req); err != nil {
        return nil, err
    }

    // 2. Mapear DTO a Model
    property := req.ToModel()

    // 3. Crear en BD
    if err := s.repo.Create(ctx, property); err != nil {
        return nil, err
    }

    // 4. Invalidar cache
    s.cache.InvalidatePropertyList(ctx)

    // 5. Audit
    s.audit.LogCreate(ctx, "propiedad", property.ID, property)

    // 6. Mapear a Response
    return dto.ToPropertyResponse(property), nil
}
```

### 3. DTO Pattern

**Propósito:** Controlar qué datos se exponen en la API.

```go
// internal/dto/property_dto.go
type CreatePropertyRequest struct {
    Titulo       string  `json:"titulo" validate:"required,min=10,max=200"`
    Descripcion  string  `json:"descripcion" validate:"required,min=50"`
    TipoPropiedad string `json:"tipo_propiedad" validate:"required,oneof=casa departamento terreno"`
    Operacion     string `json:"operacion" validate:"required,oneof=venta alquiler anticretico"`
    Precio       float64 `json:"precio" validate:"required,gt=0"`
    Ciudad       string  `json:"ciudad" validate:"required,min=3"`
    // ... otros campos
}

type PropertyResponse struct {
    ID           uint                 `json:"id"`
    Titulo       string               `json:"titulo"`
    Descripcion  string               `json:"descripcion"`
    Precio       float64              `json:"precio"`
    Moneda       string               `json:"moneda"`
    Fotos        []PhotoResponse      `json:"fotos"`
    Contacto     *ContactoResponse    `json:"contacto,omitempty"` // Solo si autenticado
    CreatedAt    time.Time            `json:"created_at"`
}

func ToPropertyResponse(p *models.Property) *PropertyResponse {
    return &PropertyResponse{
        ID:          p.ID,
        Titulo:      p.Titulo,
        Descripcion: p.Descripcion,
        // ...
    }
}
```

### 4. Dependency Injection

**Propósito:** Inyectar dependencias para facilitar testing.

```go
// internal/handlers/property_handler.go
type PropertyHandler struct {
    service services.PropertyService
}

func NewPropertyHandler(service services.PropertyService) *PropertyHandler {
    return &PropertyHandler{service: service}
}

func (h *PropertyHandler) CreateProperty(c *fiber.Ctx) error {
    var req dto.CreatePropertyRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(errorResponse(err))
    }

    userID := c.Locals("user_id").(uint)
    req.UserID = userID

    property, err := h.service.CreateProperty(c.Context(), &req)
    if err != nil {
        return c.Status(500).JSON(errorResponse(err))
    }

    return c.Status(201).JSON(successResponse(property))
}
```

---

## 🔄 Flujo de una Request

```
1. CLIENT REQUEST
   ↓
2. FIBER ROUTER
   ↓
3. MIDDLEWARE CHAIN
   - CORS
   - Logger
   - Rate Limiting
   - Auth (JWT validation)
   - Role Authorization
   - Audit Logging
   ↓
4. HANDLER (Controller)
   - Parse request body
   - Call service
   - Return response
   ↓
5. SERVICE (Business Logic)
   - Validate input
   - Call repository
   - Call external services
   - Apply business rules
   ↓
6. REPOSITORY (Data Access)
   - Query database
   - Return data
   ↓
7. DATABASE (PostgreSQL)
   - Execute query
   - Return results
   ↓
8. RESPONSE (reverse order)
   - Repository → Service → Handler → Middleware → Client
```

---

## 🚀 Middleware Chain

### Orden de Middleware

```go
// cmd/api/main.go
app.Use(middleware.CORS())          // 1. CORS headers
app.Use(middleware.Logger())        // 2. Request logging
app.Use(middleware.Recover())       // 3. Panic recovery
app.Use(middleware.RateLimit())     // 4. Rate limiting
app.Use(middleware.Auth())          // 5. JWT validation
app.Use(middleware.Role())          // 6. Role authorization
app.Use(middleware.Audit())         // 7. Audit logging
```

### Auth Middleware

```go
// internal/middleware/auth.go
func AuthMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 1. Extract token from Authorization header
        token := c.Get("Authorization")
        if token == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Authorization header required",
            })
        }

        // 2. Validate token
        claims, err := jwt.ValidateToken(token)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid token",
            })
        }

        // 3. Store user info in context
        c.Locals("user_id", claims.UserID)
        c.Locals("user_role", claims.Role)

        // 4. Continue to next middleware/handler
        return c.Next()
    }
}
```

### Role Middleware

```go
// internal/middleware/role.go
func RoleMiddleware(allowedRoles ...models.UserRole) fiber.Handler {
    return func(c *fiber.Ctx) error {
        userRole := c.Locals("user_role").(models.UserRole)

        if !contains(allowedRoles, userRole) {
            return c.Status(403).JSON(fiber.Map{
                "error": "Insufficient permissions",
            })
        }

        return c.Next()
    }
}

// Usage:
// app.Get("/admin/users", middleware.Auth(), middleware.Role(models.RoleAdmin), adminHandler.ListUsers)
```

---

## 💾 Cache Strategy

### Redis Cache Implementation

```go
// internal/services/cache_service.go
type CacheService interface {
    GetProperty(ctx context.Context, id uint) (*models.Property, error)
    SetProperty(ctx context.Context, property *models.Property, ttl time.Duration) error
    InvalidateProperty(ctx context.Context, id uint) error
    GetPropertyList(ctx context.Context, filter string) ([]*models.Property, error)
    SetPropertyList(ctx context.Context, filter string, properties []*models.Property, ttl time.Duration) error
}

type cacheService struct {
    client *redis.Client
}

func (s *cacheService) GetProperty(ctx context.Context, id uint) (*models.Property, error) {
    key := fmt.Sprintf("property:%d", id)
    data, err := s.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, nil // Cache miss
    }
    if err != nil {
        return nil, err
    }

    var property models.Property
    if err := json.Unmarshal(data, &property); err != nil {
        return nil, err
    }

    return &property, nil
}
```

### Cache-Aside Pattern

```go
// Service layer
func (s *propertyService) GetProperty(ctx context.Context, id uint) (*dto.PropertyResponse, error) {
    // 1. Try cache
    property, err := s.cache.GetProperty(ctx, id)
    if err == nil && property != nil {
        return dto.ToPropertyResponse(property), nil
    }

    // 2. Cache miss: get from DB
    property, err = s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. Store in cache
    s.cache.SetProperty(ctx, property, 1*time.Hour)

    return dto.ToPropertyResponse(property), nil
}
```

### TTLs (Time To Live)

| Data | TTL | Justificación |
|------|-----|---------------|
| Property individual | 1 hora | Cambia poco, se accede mucho |
| Property list | 15 min | Cambia frecuentemente (nuevas) |
| Categories | 24 horas | Cambia muy poco |
| User session | 30 min | Seguridad |
| Search results | 30 min | Balance frescura/rendimiento |

---

## 📤 File Upload Strategy

### Cloudflare R2 Integration

```go
// internal/services/storage_service.go
type StorageService interface {
    UploadPropertyPhoto(ctx context.Context, propertyID uint, file io.Reader, filename string) (string, error)
    UploadThumbnail(ctx context.Context, imageURL string) (string, error)
    DeletePhoto(ctx context.Context, url string) error
}

type storageService struct {
    client *s3.Client
    config *config.StorageConfig
}

func (s *storageService) UploadPropertyPhoto(
    ctx context.Context,
    propertyID uint,
    file io.Reader,
    filename string,
) (string, error) {
    // 1. Generate path: propiedades/{property_id}/{timestamp}_{filename}
    path := fmt.Sprintf("propiedades/%d/%d_%s", propertyID, time.Now().Unix(), filename)

    // 2. Upload to R2
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(path),
        Body:   file,
        ACL:    types.ObjectCannedACLPublicRead,
    })
    if err != nil {
        return "", err
    }

    // 3. Return public URL
    url := fmt.Sprintf("%s/%s", s.config.PublicURL, path)
    return url, nil
}
```

### Image Processing

```go
// internal/utils/image.go
func ProcessImage(file io.Reader) (io.Reader, error) {
    // 1. Decode image
    img, format, err := image.Decode(file)
    if err != nil {
        return nil, err
    }

    // 2. Resize if needed (max 1920x1080)
    img = resize.Thumbnail(1920, 1080, img, resize.Lanczos3)

    // 3. Compress
    var buf bytes.Buffer
    quality := 85
    err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
    if err != nil {
        return nil, err
    }

    return &buf, nil
}
```

---

## 🔐 Security Implementation

### Password Hashing

```go
// internal/services/auth_service.go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### JWT Token Generation

```go
// internal/utils/jwt.go
func GenerateToken(user *models.User) (string, error) {
    claims := jwt.MapClaims{
        "user_id": user.ID,
        "email":   user.Email,
        "role":    user.Rol,
        "exp":     time.Now().Add(24 * time.Hour).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
```

### Input Validation

```go
// internal/validators/property_validator.go
import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreateProperty(req *dto.CreatePropertyRequest) error {
    if err := validate.Struct(req); err != nil {
        return err
    }

    // Custom validations
    if req.Precio <= 0 {
        return errors.New("precio must be greater than 0")
    }

    return nil
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// tests/unit/property_service_test.go
func TestPropertyService_CreateProperty(t *testing.T) {
    // Setup
    mockRepo := &MockPropertyRepository{}
    mockCache := &MockCacheService{}
    mockAudit := &MockAuditService{}
    service := NewPropertyService(mockRepo, mockCache, mockAudit)

    // Test data
    req := &dto.CreatePropertyRequest{
        Titulo:       "Casa bonita",
        Descripcion:  "Hermosa casa en venta",
        TipoPropiedad: "casa",
        Operacion:    "venta",
        Precio:       50000,
        Ciudad:       "Santa Cruz",
    }

    // Execute
    response, err := service.CreateProperty(context.Background(), req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Equal(t, req.Titulo, response.Titulo)
    assert.Equal(t, 1, mockRepo.CreateCallCount)
}
```

### Integration Tests

```go
// tests/integration/api_test.go
func TestPropertyAPI_CreateProperty(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer cleanupTestDB(db)

    // Setup test app
    app := setupTestApp(db)

    // Create test user and get token
    token := createTestUserAndLogin(t, app)

    // Create request
    reqBody := createPropertyRequestBody()
    req := httptest.NewRequest("POST", "/api/propiedades", strings.NewReader(reqBody))
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Perform request
    resp, err := app.Test(req)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)

    var response map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&response)
    assert.True(t, response["success"].(bool))
}
```

---

## 📊 Monitoring & Logging

### Structured Logging

```go
// pkg/logger/logger.go
import "go.uber.org/zap"

type Logger struct {
    *zap.Logger
}

func NewLogger() *Logger {
    logger, _ := zap.NewProduction()
    return &Logger{logger}
}

func (l *Logger) LogCreateProperty(userID uint, propertyID uint) {
    l.Info("property_created",
        zap.Uint("user_id", userID),
        zap.Uint("property_id", propertyID),
        zap.Time("timestamp", time.Now()),
    )
}
```

### Metrics (Prometheus)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    propertyCreatedCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "properties_created_total",
            Help: "Total number of properties created",
        },
        []string{"user_role"},
    )

    propertyListDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "property_list_duration_seconds",
            Help:    "Duration of property list requests",
            Buckets: prometheus.DefBuckets,
        },
    )
)
```

---

## 🚀 Deployment Strategy

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env .env
EXPOSE 3001
CMD ["./main"]
```

### docker-compose.yml

```yaml
version: '3.8'
services:
  api:
    build: .
    ports:
      - "3001:3001"
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/inmobiliaria
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15
    environment:
      POSTGRES_DB: inmobiliaria
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  worker:
    build: .
    command: ./worker
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/inmobiliaria
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

volumes:
  postgres_data:
  redis_data:
```

---

## 📝 Conventions

### Code Style

- **Files:** `snake_case.go`
- **Packages:** lowercase, single word
- **Exports:** `PascalCase`
- **Privates:** `camelCase`
- **Constants:** `PascalCase`
- **Interfaces:** `PascalCase` + `er` suffix (e.g., `PropertyRepository`)
- **Errors:** wrap with `fmt.Errorf("context: %w", err)`

### Git Conventional Commits

```
feat: add property creation endpoint
fix: resolve race condition in cache
docs: update API documentation
refactor: extract validation logic
test: add integration tests for payments
chore: upgrade dependencies
```

---

**Última actualización:** 2026-04-09
**Versión:** 1.0.0
**Autor:** Scrum Master 365Soft
