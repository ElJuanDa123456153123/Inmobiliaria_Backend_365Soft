# Modelos de Datos - Inmobiliaria 365Soft

## 📋 Índice de Modelos

### Usuarios y Autenticación
- **User** (`user.go`) - Usuario del sistema con roles

### Propiedades
- **Property** (`property.go`) - Propiedad inmobiliaria principal
- **PropertyPhoto** (`property_photo.go`) - Fotos asociadas a una propiedad
- **Category** (`category.go`) - Categorías de clasificación
- **PropertyCategory** (`category.go`) - Tabla pivote muchos-a-muchos

### Auditoría
- **AuditLog** (`audit.go`) - Registro de todas las acciones del sistema

## 🔄 Relaciones entre Modelos

```
User (1) ─────< (N) Property
                     │
                     ├─< (3-7) PropertyPhoto
                     │
                     └─< (N) Category (M:N via PropertyCategory)

User (1) ─────< (N) AuditLog
Property (1) ──< (N) AuditLog (via EntityID)
```

## 📊 Constantes y Enums

### PropertyType
- `casa` - Casa residencial
- `departamento` - Departamento/apartamento
- `terreno` - Terreno baldío
- `local` - Local comercial
- `oficina` - Oficina
- `galpon` - Galpón/Nave industrial
- `hacienda` - Hacienda/Finca
- `lote` - Lote de terreno

### PropertyOperation
- `venta` - Venta de propiedad
- `alquiler` - Alquiler mensual
- `anticretico` - Anticrético

### PropertyStatus
- `borrador` - Creada pero no publicada
- `pendiente` - Esperando aprobación
- `activa` - Publicada y visible
- `inactiva` - Vencida o pausada
- `vendida` - Marcada como vendida
- `alquilada` - Marcada como alquilada
- `rechazada` - Rechazada por admin

### UserRole
- `usuario` - Usuario normal
- `agente` - Agente inmobiliario
- `inmobiliaria` - Inmobiliaria
- `admin` - Administrador

### AuditAction
- `crear` - Creación de registro
- `actualizar` - Actualización de registro
- `eliminar` - Eliminación de registro
- `login` - Inicio de sesión
- `logout` - Cierre de sesión
- `aprobar` - Aprobación de entidad
- `rechazar` - Rechazo de entidad
- `ver` - Visualización
- `exportar` - Exportación de datos

## 🗂️ Soft Delete

Todos los modelos usan **Soft Delete** mediante `gorm.DeletedAt`:
- Los registros no se eliminan físicamente
- Se marca `deleted_at` con timestamp
- GORM filtra automáticamente estos registros
- Los admin pueden restaurar registros

## 📷 Manejo de Fotos

### Características:
- **Mínimo recomendado**: 3 fotos (warning, no bloquea)
- **Máximo permitido**: 7 fotos (error, bloquea)
- **Portada**: Una foto marcada como portada por defecto
- **Orden**: Fotos ordenables manualmente
- **Thumbnails**: Generado automáticamente
- **Formatos**: JPEG, PNG, WEBP
- **Tamaño máximo**: 5MB por foto

## 🔍 SEO

Cada propiedad incluye:
- `slug` - URL amigable única (ej: `hermosa-casa-en-venta-123`)
- `meta_titulo` - Título para SEO
- `meta_descripcion` - Descripción para SEO

## 📊 Índices de Base de Datos

### Índices creados automáticamente:
- Primary keys en todos los IDs
- Unique index en `slug` (propiedades)
- Unique index en `email` (usuarios)
- Index en `deleted_at` (soft delete)

### Índices compuestos (creados manualmente):
```sql
idx_propiedades_busqueda - Para listado principal
idx_propiedades_precio - Para filtros de precio
idx_propiedades_ciudad_estado - Para búsquedas por ciudad
idx_propiedades_destacadas - Para propiedades destacadas
idx_fotos_propiedad_orden - Para orden de fotos
idx_fotos_portada - Para buscar portadas
idx_audit_logs_user_created - Para auditoría por usuario
idx_audit_logs_entity_created - Para auditoría por entidad
```

## 🧪 Testing

Para crear datos de prueba:
```bash
go run cmd/seed/main.go
```

## 📝 Notas Importantes

1. **Latitud/Longitud**: Usan `*float64` (nullable) para permitir propiedades sin ubicación precisa
2. **Precios**: `decimal(15,2)` en BD, `float64` en Go
3. **Textos largos**: Descripción usa `type:text` en PostgreSQL
4. **JSONB**: AuditLog usa `pq.Type` para manejo de JSON en PostgreSQL
5. **Timezones**: Todos los timestamps usan UTC en BD
