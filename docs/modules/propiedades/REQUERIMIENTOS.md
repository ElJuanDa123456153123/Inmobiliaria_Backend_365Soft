# 🏠 Módulo de Propiedades - Documentación Completa

## 🎯 Visión General del Módulo

El módulo de propiedades es el corazón del sistema, permitiendo a usuarios crear, gestionar, buscar y visualizar propiedades inmobiliarias.

---

## 👥 Responsabilidades

### Desarrollador Asignado
- **Principal:** Dev 2 (Backend Dev - Propiedades y Búsqueda)
- **Backup:** Dev 1 (Backend Lead)

### Dependencias
- Módulo de Autenticación (user info)
- Módulo de Storage (upload fotos)
- Módulo de Publicaciones (vigencia)
- Módulo de Cache (Redis)

---

## 📦 Funcionalidades

### 1. Creación de Propiedades

**Endpoint:** `POST /api/propiedades`

**Autenticación:** Requerida (cualquier usuario registrado)

**Request Body:**
```json
{
  "titulo": "Hermosa casa en venta",
  "descripcion": "Bonita casa con 3 habitaciones, ideal para familia...",
  "tipo_propiedad": "casa",
  "operacion": "venta",
  "precio": 85000,
  "moneda": "USD",
  "direccion": "Calle Principal #123",
  "ciudad": "Santa Cruz",
  "zona": "Equipetrol",
  "latitud": -17.7641,
  "longitud": -63.1825,
  "metraje_total": 250.5,
  "metraje_construido": 180.0,
  "numero_habitaciones": 3,
  "numero_banos": 2,
  "numero_garages": 2,
  "antiguedad_anos": 5,
  "tiene_piscina": true,
  "tiene_jardin": true,
  "tiene_terraza": false,
  "tiene_servicio": true,
  "tiene_seguridad": true,
  "es_amueblado": false,
  "acepta_mascotas": true,
  "categorias": ["residencial", "lujo"]
}
```

**Validaciones:**
- `titulo`: Required, min 10 caracteres, max 200
- `descripcion`: Required, min 50 caracteres
- `tipo_propiedad`: Required, debe ser: casa, departamento, terreno, local, oficina, galpon, hacienda, lote
- `operacion`: Required, debe ser: venta, alquiler, anticretico
- `precio`: Required, mayor a 0
- `moneda`: Default "USD", opciones: USD, BOB
- `ciudad`: Required, min 3 caracteres
- `latitud`: Opcional, range -90 a 90
- `longitud`: Opcional, range -180 a 180

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "titulo": "Hermosa casa en venta",
    "slug": "hermosa-casa-en-venta-123",
    "estado": "borrador",
    "fotos": [],
    "created_at": "2026-04-09T15:30:00Z"
  },
  "message": "Propiedad creada exitosamente"
}
```

**Behavior:**
- Crea propiedad con estado `borrador`
- Genera slug único automáticamente
- No incluye contacto (se usa el del usuario creador)
- Audit log: Acción "crear" en entidad "propiedad"

---

### 2. Upload de Fotos

**Endpoint:** `POST /api/propiedades/:id/fotos`

**Autenticación:** Requerida (solo dueño de propiedad)

**Request:** `multipart/form-data`

```
foto1: [file]
foto2: [file]
foto3: [file]
orden: [1,2,3] // opcional, array de órdenes
```

**Validaciones:**
- Formatos permitidos: JPEG, PNG, WEBP
- Tamaño máximo: 5MB por foto
- Mínimo recomendado: 3 fotos (WARNING si menos)
- Máximo permitido: 7 fotos (ERROR si más)
- Dimensiones mínimas: 800x600px

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "fotos": [
      {
        "id": 1,
        "url": "https://cdn.ejemplo.com/propiedades/123/1_casa.jpg",
        "url_thumbnail": "https://cdn.ejemplo.com/propiedades/123/1_casa_thumb.jpg",
        "orden": 1,
        "es_portada": true
      },
      {
        "id": 2,
        "url": "https://cdn.ejemplo.com/propiedades/123/2_cocina.jpg",
        "url_thumbnail": "https://cdn.ejemplo.com/propiedades/123/2_cocina_thumb.jpg",
        "orden": 2,
        "es_portada": false
      }
    ],
    "warning": "Recomendamos agregar al menos 3 fotos para mejor visibilidad"
  },
  "message": "Fotos subidas exitosamente"
}
```

**Behavior:**
- Redimensiona imágenes a máximo 1920x1080px
- Genera thumbnail de 300x300px
- Optimiza compresión (calidad 85%)
- Primera foto es marcada como portada automáticamente
- Sube a Cloudflare R2
- Invalida cache de propiedad

---

### 3. Marcar Foto como Portada

**Endpoint:** `PUT /api/propiedades/:id/fotos/:foto_id/portada`

**Autenticación:** Requerida (solo dueño de propiedad)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": 2,
    "es_portada": true
  },
  "message": "Foto marcada como portada"
}
```

**Behavior:**
- Desmarca otras fotos de la propiedad
- Marca la seleccionada como portada

---

### 4. Reordenar Fotos

**Endpoint:** `PUT /api/propiedades/:id/fotos/reordenar`

**Autenticación:** Requerida (solo dueño de propiedad)

**Request Body:**
```json
{
  "orden": [3, 1, 2]
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Fotos reordenadas exitosamente"
}
```

---

### 5. Eliminar Foto

**Endpoint:** `DELETE /api/propiedades/:id/fotos/:foto_id`

**Autenticación:** Requerida (solo dueño de propiedad)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Foto eliminada exitosamente"
}
```

**Behavior:**
- Elimina foto de Cloudflare R2
- Si era portada, marca la siguiente como portada
- Si era la última foto, no hay portada

---

### 6. Listado de Propiedades (Búsqueda)

**Endpoint:** `GET /api/propiedades`

**Autenticación:** Opcional

**Query Params:**
```
?page=1
&limit=20
&tipo=casa,departamento
&operacion=venta,alquiler
&precio_min=10000
&precio_max=200000
&ciudad=Santa Cruz
&zona=Equipetrol
&habitaciones_min=2
&habitaciones_max=4
&banos_min=1
&banos_max=3
&metraje_min=100
&metraje_max=300
&tiene_piscina=true
&tiene_jardin=true
&ordenar=precio_asc // opciones: precio_asc, precio_desc, mas_recientes, mas_visitados
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "titulo": "Hermosa casa en venta",
      "descripcion": "Bonita casa con 3 habitaciones...",
      "precio": 85000,
      "moneda": "USD",
      "tipo_propiedad": "casa",
      "operacion": "venta",
      "ciudad": "Santa Cruz",
      "zona": "Equipetrol",
      "metraje_construido": 180,
      "numero_habitaciones": 3,
      "numero_banos": 2,
      "numero_garages": 2,
      "tiene_piscina": true,
      "tiene_jardin": true,
      "foto_portada": "https://cdn.ejemplo.com/propiedades/123/1_casa_thumb.jpg",
      "destacado": true,
      "numero_visitas": 125,
      "created_at": "2026-04-09T15:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Behavior (para NO autenticados):**
- Solo retorna propiedades con estado `activa`
- NO incluye datos de contacto del agente
- Caching: 15 minutos en Redis

**Behavior (para autenticados):**
- Incluye datos de contacto si la propiedad está activa
- Dueño de propiedad: ve todas sus propiedades sin importar estado

**Cache Strategy:**
- Key: `properties:list:{hash_of_filters}`
- TTL: 15 minutos
- Invalidar al: crear/actualizar/eliminar propiedad

---

### 7. Detalle de Propiedad

**Endpoint:** `GET /api/propiedades/:id`

**Autenticación:** Opcional

**Response (200 OK):**

**Para NO autenticados:**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "titulo": "Hermosa casa en venta",
    "descripcion": "Bonita casa con 3 habitaciones...",
    "precio": 85000,
    "moneda": "USD",
    "tipo_propiedad": "casa",
    "operacion": "venta",
    "estado": "activa",
    "direccion": "Calle Principal #123",
    "ciudad": "Santa Cruz",
    "zona": "Equipetrol",
    "latitud": -17.7641,
    "longitud": -63.1825,
    "google_maps_url": "https://maps.google.com/?q=-17.7641,-63.1825",
    "metraje_total": 250.5,
    "metraje_construido": 180.0,
    "numero_habitaciones": 3,
    "numero_banos": 2,
    "numero_garages": 2,
    "antiguedad_anos": 5,
    "tiene_piscina": true,
    "tiene_jardin": true,
    "tiene_terraza": false,
    "tiene_servicio": true,
    "tiene_seguridad": true,
    "es_amueblado": false,
    "acepta_mascotas": true,
    "fotos": [
      {
        "url": "https://cdn.ejemplo.com/propiedades/123/1_casa.jpg",
        "url_thumbnail": "https://cdn.ejemplo.com/propiedades/123/1_casa_thumb.jpg",
        "descripcion": "Fachada principal",
        "es_portada": true
      }
    ],
    "categorias": [
      {
        "id": 1,
        "nombre": "Residencial",
        "slug": "residencial"
      }
    ],
    "numero_visitas": 125,
    "numero_contactos": 15,
    "destacado": true,
    "created_at": "2026-04-09T15:30:00Z",
    "contacto": null
  }
}
```

**Para autenticados:**
```json
{
  "success": true,
  "data": {
    // ... todos los campos anteriores ...
    "contacto": {
      "nombre": "Juan Pérez",
      "telefono": "+591 77712345",
      "whatsapp": "+591 77712345",
      "email": "juan@example.com",
      "mostrar_telefono": true,
      "mostrar_whatsapp": true,
      "mostrar_email": false
    }
  }
}
```

**Behavior:**
- Incrementa contador `numero_visitas`
- Cache: 1 hora para autenticados, 15 min para no autenticados
- Para dueño: muestra todos los estados y edición

---

### 8. Actualizar Propiedad

**Endpoint:** `PUT /api/propiedades/:id`

**Autenticación:** Requerida (solo dueño)

**Request Body:** Mismo que crear

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "titulo": "Hermosa casa en venta (Actualizada)",
    // ... todos los campos ...
    "updated_at": "2026-04-09T16:00:00Z"
  },
  "message": "Propiedad actualizada exitosamente"
}
```

**Behavior:**
- Actualiza solo campos enviados (PATCH-like)
- Si propiedad está `activa`, puede requerir reaprobación (configurable)
- Invalida cache
- Audit log con cambios before/after

---

### 9. Eliminar Propiedad

**Endpoint:** `DELETE /api/propiedades/:id`

**Autenticación:** Requerida (solo dueño)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Propiedad eliminada exitosamente"
}
```

**Behavior:**
- Soft delete (marca `deleted_at`)
- No elimina fotos de R2 inmediatamente (cleanup job)
- Desactiva publicación si existe
- Invalida cache

---

### 10. Propiedades Destacadas

**Endpoint:** `GET /api/propiedades/destacadas`

**Autenticación:** No requerida

**Query Params:**
```
?limit=10
&ciudad=Santa Cruz
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "titulo": "Hermosa casa en venta",
      "precio": 85000,
      "moneda": "USD",
      "foto_portada": "https://cdn.ejemplo.com/...",
      "destacado": true,
      "numero_visitas": 1250
    }
  ]
}
```

**Behavior:**
- Retorna propiedades con `destacado=true` y `estado=activa`
- Ordenado por `numero_visitas` DESC
- Cache: 1 hora

---

### 11. Propiedades Relacionadas

**Endpoint:** `GET /api/propiedades/:id/relacionadas`

**Autenticación:** No requerida

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": 124,
      "titulo": "Casa similar cerca",
      "precio": 90000,
      "moneda": "USD",
      "foto_portada": "https://cdn.ejemplo.com/...",
      "ciudad": "Santa Cruz"
    }
  ]
}
```

**Behavior:**
- Busca propiedades con mismo `tipo_propiedad`, `operacion`, `ciudad`
- Rango de precio ±20%
- Max 4 propiedades
- Excluye la propiedad actual
- Cache: 1 hora

---

## 🔍 Lógica de Búsqueda

### PostgreSQL Full-Text Search (MVP)

```sql
-- Búsqueda por texto en título y descripción
SELECT * FROM propiedades
WHERE to_tsvector('spanish', titulo || ' ' || descripcion) @@ to_tsquery('spanish', 'casa & piscina')
AND estado = 'activa'
ORDER BY destacado DESC, created_at DESC;
```

### Índices de Búsqueda

```sql
-- Índice compuesto principal
CREATE INDEX idx_propiedades_busqueda ON propiedades(
    estado, tipo_propiedad, operacion, ciudad, destacado, created_at
) WHERE estado = 'activa';

-- Índice para rangos de precios
CREATE INDEX idx_propiedades_precio ON propiedades(precio, estado) WHERE estado = 'activa';

-- Full-text search
CREATE INDEX idx_propiedades_fts ON propiedades
USING GIN(to_tsvector('spanish', titulo || ' ' || descripcion));
```

### Filtros Implementados

1. **Tipo de propiedad:** Multi-selección (casa, departamento, terreno...)
2. **Operación:** Multi-selección (venta, alquiler, anticrético)
3. **Rango de precios:** precio_min, precio_max
4. **Ubicación:** ciudad, zona (autocomplete)
5. **Características:** habitaciones, baños, garages (rangos)
6. **Metraje:** metraje_min, metraje_max
7. **Amenities:** boolean filters (piscina, jardín, etc.)
8. **Ordenamiento:** precio_asc, precio_desc, mas_recientes, mas_visitados

---

## 📊 Métricas y Analytics

### Contadores
- `numero_visitas`: Incrementa en cada GET /api/propiedades/:id
- `numero_contactos`: Incrementa cuando alguien hace clic en "Contactar"

### Reportes (Admin)
- Propiedades más visitadas
- Propiedades por tipo
- Propiedades por ciudad
- Propiedades por estado
- Tasa de conversión (vistas → contactos)

---

## 🗄️ Base de Datos

### Tabla: propiedades

Ver documento `03_MODELO_DE_DATOS.md` para schema completo.

### Índices Críticos
```sql
CREATE INDEX idx_propiedades_estado ON propiedades(estado);
CREATE INDEX idx_propiedades_tipo ON propiedades(tipo_propiedad);
CREATE INDEX idx_propiedades_operacion ON propiedades(operacion);
CREATE INDEX idx_propiedades_ciudad ON propiedades(ciudad);
CREATE INDEX idx_propiedades_precio ON propiedades(precio);
CREATE INDEX idx_propiedades_destacado ON propiedades(destacado);
CREATE INDEX idx_propiedades_user_id ON propiedades(user_id);
CREATE INDEX idx_propiedades_slug ON propiedades(slug);
```

---

## 🧪 Tests

### Unit Tests (services)
- `TestPropertyService_CreateProperty`
- `TestPropertyService_UpdateProperty`
- `TestPropertyService_DeleteProperty`
- `TestPropertyService_GetProperty`
- `TestPropertyService_ListProperties`
- `TestPropertyService_SearchProperties`

### Integration Tests (API)
- `TestPropertyAPI_CreateProperty`
- `TestPropertyAPI_UploadPhotos`
- `TestPropertyAPI_UpdateProperty`
- `TestPropertyAPI_DeleteProperty`
- `TestPropertyAPI_ListProperties`
- `TestPropertyAPI_GetProperty`

### Test Cases
1. Crear propiedad sin autenticación → 401
2. Crear propiedad con datos inválidos → 400
3. Crear propiedad válida → 201
4. Upload 8 fotos → Error (máximo 7)
5. Upload 2 fotos → Warning (recomendamos 3)
6. Buscar sin filtros → 200 + listado
7. Buscar con filtros → 200 + listado filtrado
8. Actualizar propiedad de otro usuario → 403
9. Eliminar propiedad de otro usuario → 403

---

## 🚀 Implementación

### Archivos a Crear

1. **Model:**
   - `internal/models/property.go`

2. **Repository:**
   - `internal/repository/property_repo.go`

3. **Service:**
   - `internal/services/property_service.go`
   - `internal/services/search_service.go`

4. **Handler:**
   - `internal/handlers/property_handler.go`

5. **DTO:**
   - `internal/dto/property_dto.go`

6. **Validator:**
   - `internal/validators/property_validator.go`

7. **Utils:**
   - `internal/utils/slug.go` (generación de slugs)

### Endpoints Summary

| Método | Endpoint | Auth | Descripción |
|--------|----------|------|-------------|
| POST | /api/propiedades | ✅ | Crear propiedad |
| GET | /api/propiedades | ❌ | Listar propiedades |
| GET | /api/propiedades/:id | ❌ | Detalle de propiedad |
| PUT | /api/propiedades/:id | ✅ | Actualizar propiedad |
| DELETE | /api/propiedades/:id | ✅ | Eliminar propiedad |
| POST | /api/propiedades/:id/fotos | ✅ | Subir fotos |
| PUT | /api/propiedades/:id/fotos/:foto_id/portada | ✅ | Marcar portada |
| PUT | /api/propiedades/:id/fotos/reordenar | ✅ | Reordenar fotos |
| DELETE | /api/propiedades/:id/fotos/:foto_id | ✅ | Eliminar foto |
| GET | /api/propiedades/destacadas | ❌ | Propiedades destacadas |
| GET | /api/propiedades/:id/relacionadas | ❌ | Propiedades relacionadas |

---

## ⚠️ Consideraciones Especiales

### Performance
- Usar eager loading (GORM Preload) para fotos y categorías
- Paginar siempre (max 50 items)
- Cache agresivo para listados
- Select fields específicos cuando sea posible

### Seguridad
- Validar que lat/lng estén en rangos válidos
- Sanitizar HTML en descripción
- Validar tamaño de archivos (magic bytes, no solo extensión)
- Rate limiting en upload de fotos

### SEO
- Generar slug único automáticamente
- Meta título y descripción
- Open Graph tags
- Schema.org markup

---

## ✅ Criterios de Aceptación

- [ ] Usuario puede crear propiedad con todos los campos
- [ ] Sistema valida campos obligatorios correctamente
- [ ] Sistema avisa si carga menos de 3 fotos (no bloquea)
- [ ] Sistema bloquea si carga más de 7 fotos
- [ ] Usuario puede subir fotos y se redimensionan correctamente
- [ ] Usuario puede marcar foto como portada
- [ ] Usuario puede reordenar fotos
- [ ] Usuario puede eliminar fotos
- [ ] Búsqueda funciona con todos los filtros
- [ ] Búsqueda es performante (< 1s)
- [ ] No autenticados NO ven datos de contacto
- [ ] Autenticados SÍ ven datos de contacto
- [ ] Dueño puede editar sus propiedades
- [ ] Dueño puede eliminar sus propiedades
- [ ] Contador de visitas se incrementa correctamente
- [ ] Cache funciona correctamente
- [ ] Todos los endpoints tienen tests

---

**Última actualización:** 2026-04-09
**Versión:** 1.0.0
**Asignado a:** Dev 2 (Backend Dev - Propiedades)
