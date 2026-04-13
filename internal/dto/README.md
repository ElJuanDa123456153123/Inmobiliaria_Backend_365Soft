# DTOs - Data Transfer Objects

## 📋 Índice

### Request DTOs (Datos de Entrada)
- [CreatePropertyRequest](property_dto.go) - Crear propiedad
- [UpdatePropertyRequest](property_dto.go) - Actualizar propiedad
- [UploadPhotosRequest](property_dto.go) - Subir fotos
- [PropertyFilter](property_dto.go) - Filtros de búsqueda

### Response DTOs (Datos de Salida)
- [PropertyResponse](property_dto.go) - Propiedad completa
- [PropertyListResponse](property_dto.go) - Listado paginado
- [PropertyListItemResponse](property_dto.go) - Item simplificado
- [PhotoResponse](property_dto.go) - Foto de propiedad
- [CategoryResponse](property_dto.go) - Categoría
- [ContactoResponse](property_dto.go) - Información de contacto

### Pagination & Response
- [PaginationRequest](pagination.go) - Request de paginación
- [PaginationResponse](pagination.go) - Respuesta de paginación
- [Response](pagination.go) - Respuesta estándar de API

## 🎯 Uso de DTOs

### Ejemplo: Crear Propiedad

```go
import "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"

// Crear request desde JSON body
var req dto.CreatePropertyRequest
if err := c.BodyParser(&req); err != nil {
    return err
}

// Validar
validator := validators.NewPropertyValidator()
if errs := validator.ValidateCreateProperty(&req); errs != nil {
    return c.Status(400).JSON(dto.ValidationErrorResponse(errs))
}

// Convertir a modelo
property := req.ToModel(userID)
```

### Ejemplo: Response Exitosa

```go
property := &models.Property{...}
response := dto.MapPropertyToResponse(property, true) // true = incluir contacto

return c.JSON(dto.SuccessResponseWithMessage(
    response,
    "Propiedad creada exitosamente",
))
```

### Ejemplo: Response con Paginación

```go
properties := []*models.Property{...}
total := int64(100)

pagination := dto.NewPaginationResponse(1, 20, total)
response := dto.PropertyListResponse{
    Data:       properties,
    Pagination: pagination,
}

return c.JSON(response)
```

### Ejemplo: Response de Error

```go
if err != nil {
    return c.Status(404).JSON(dto.NotFoundResponse("Propiedad"))
}

if errors := validator.ValidateCreateProperty(&req); len(errors) > 0 {
    details := dto.FormatErrorsForResponse(errors)
    return c.Status(400).JSON(dto.ValidationErrorResponse(details))
}
```

## 📊 Convenciones

### Request DTOs:
- Tags `json:"field_name"` para mapeo JSON
- Tags `validate:"constraints"` para validaciones
- Pointers (`*type`) para campos opcionales en updates
- Métodos `ToModel()` para conversión a modelos

### Response DTOs:
- Campos `omitempty` para omitir vacíos
- Campos del modelo renombrados para API (snake_case)
- Relaciones eagerly loaded (Fotos, Categorías)
- Contacto solo cuando `includeContacto = true`

### Validation Tags:
- `required` - Campo obligatorio
- `min=X` - Mínimo X caracteres
- `max=X` - Máximo X caracteres
- `gte=X` - Greater than or equal X
- `lte=X` - Less than or equal X
- `oneof=a b c` - Debe ser uno de los valores
- `email` - Formato de email válido

## 🔄 Mappers

### Modelo → Response

```go
// Completos con relaciones
response := dto.MapPropertyToResponse(property, true)

// Simplificados para listados
item := dto.MapPropertyToListItem(property)
```

### Request → Modelo

```go
// Crear desde request
property := req.ToModel(userID)

// Actualizar desde request (manual)
if req.Titulo != nil {
    property.Titulo = *req.Titulo
}
```

## 📝 Custom Validators

Los validadores custom se registran en `validators/property_validator.go`:

```go
// Ejemplo de validador custom
func validatePrecioMinimo(fl validator.FieldLevel) bool {
    precio := fl.Field().Float()
    return precio > 0 && precio < 100000000 // Rango válido
}
```

## 🎨 Tips

1. **Usar helpers en handlers**:
   ```go
   if err := h.validator.ValidateCreateProperty(&req); err != nil {
       return h.errorResponse(c, err)
   }
   ```

2. **Response factory functions**:
   ```go
   // Exitoso
   dto.SuccessResponse(data)
   dto.SuccessResponseWithMessage(data, "Mensaje")

   // Errores
   dto.NotFoundResponse("recurso")
   dto.UnauthorizedResponse()
   dto.ForbiddenResponse()
   ```

3. **Paginación automática**:
   ```go
   filter := dto.PropertyFilter{Page: 1, Limit: 20}
   filter.SetDefaults() // Establece defaults
   offset := filter.GetOffset() // Calcula offset
   ```
