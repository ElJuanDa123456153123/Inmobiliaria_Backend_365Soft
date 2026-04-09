# 🗄️ Modelo de Datos - SaaS Inmobiliario 365Soft

## 🎯 Visión General

Este documento define el modelo de datos completo del sistema, incluyendo entidades, relaciones, índices y validaciones.

---

## 📊 Diagrama Entidad-Relación (Texto)

```
usuarios (1) ----< (N) propiedades
usuarios (1) ----< (N) clientes
usuarios (1) ----< (N) interacciones
usuarios (1) ----< (N) auditoria

propiedades (1) ----< (3-7) fotos_propiedad
propiedades (N) ----< (M) categorias (muchos a muchos)
propiedades (1) ----< (1) publicaciones

publicaciones (1) ----< (N) pagos

categorias (1) ----< (N) propiedades (muchos a muchos)

clientes (1) ----< (N) interacciones
clientes (N) ----< (1) propiedades (opcional)

pagos ----< (webhook) banco_bisa
```

---

## 📦 Entidades del Modelo

### 1. Usuarios (users)

**Descripción:** Usuarios del sistema (personas naturales, agentes, inmobiliarias, admin)

**Tabla:** `usuarios`

```sql
CREATE TABLE usuarios (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP, -- Soft delete

    -- Información básica
    nombre VARCHAR(100) NOT NULL,
    apellidos VARCHAR(100),
    email VARCHAR(150) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL, -- bcrypt hash
    telefono VARCHAR(20),

    -- Tipo de usuario
    rol VARCHAR(20) NOT NULL DEFAULT 'usuario', -- usuario, agente, inmobiliaria, admin

    -- Información de inmobiliaria
    nombre_inmobiliaria VARCHAR(200),
    nit VARCHAR(20),

    -- Estado
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    verificado BOOLEAN NOT NULL DEFAULT FALSE,
    email_token VARCHAR(255), -- Token de verificación

    -- Avatar
    avatar_url VARCHAR(500),

    -- Google OAuth
    google_id VARCHAR(255) UNIQUE,
    google_picture_url VARCHAR(500),

    CONSTRAINT check_rol CHECK (rol IN ('usuario', 'agente', 'inmobiliaria', 'admin'))
);
```

**Índices:**
- `UNIQUE INDEX` en `email`
- `UNIQUE INDEX` en `google_id`
- `INDEX` en `deleted_at` (soft delete)
- `INDEX` en `rol`
- `INDEX` en `activo`

**Relaciones:**
- Un usuario tiene muchas propiedades
- Un usuario tiene muchos clientes (CRM)
- Un usuario tiene muchas interacciones
- Un usuario tiene muchos logs de auditoría

---

### 2. Propiedades (properties)

**Descripción:** Propiedades inmobiliarias publicadas en el sistema

**Tabla:** `propiedades`

```sql
CREATE TABLE propiedades (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Información básica
    titulo VARCHAR(200) NOT NULL,
    descripcion TEXT NOT NULL,
    tipo_propiedad VARCHAR(50) NOT NULL, -- casa, departamento, terreno, local, oficina, galpon, hacienda, lote
    operacion VARCHAR(50) NOT NULL, -- venta, alquiler, anticretico
    estado VARCHAR(50) NOT NULL DEFAULT 'borrador', -- borrador, pendiente, activa, inactiva, vendida, alquilada, rechazada

    -- Ubicación
    direccion VARCHAR(300),
    ciudad VARCHAR(100) NOT NULL,
    zona VARCHAR(100),
    latitud DECIMAL(10, 8),
    longitud DECIMAL(11, 8),
    google_maps_url VARCHAR(500),

    -- Características principales
    precio DECIMAL(15, 2) NOT NULL,
    moneda VARCHAR(10) NOT NULL DEFAULT 'USD',
    metraje_total DECIMAL(10, 2),
    metraje_construido DECIMAL(10, 2),
    numero_habitaciones INTEGER,
    numero_banos INTEGER,
    numero_garages INTEGER,
    antiguedad_anos INTEGER,

    -- Amenities
    tiene_piscina BOOLEAN NOT NULL DEFAULT FALSE,
    tiene_jardin BOOLEAN NOT NULL DEFAULT FALSE,
    tiene_terraza BOOLEAN NOT NULL DEFAULT FALSE,
    tiene_servicio BOOLEAN NOT NULL DEFAULT FALSE,
    tiene_seguridad BOOLEAN NOT NULL DEFAULT FALSE,
    es_amueblado BOOLEAN NOT NULL DEFAULT FALSE,
    acepta_mascotas BOOLEAN NOT NULL DEFAULT FALSE,

    -- Información del propietario
    user_id INTEGER NOT NULL REFERENCES usuarios(id),
    contacto_nombre VARCHAR(150),
    contacto_telefono VARCHAR(20),
    contacto_email VARCHAR(150),

    -- Visualización
    numero_visitas INTEGER NOT NULL DEFAULT 0,
    numero_contactos INTEGER NOT NULL DEFAULT 0,
    destacado BOOLEAN NOT NULL DEFAULT FALSE,

    -- SEO
    slug VARCHAR(200) UNIQUE NOT NULL,
    meta_titulo VARCHAR(200),
    meta_descripcion VARCHAR(500),

    CONSTRAINT check_tipo_propiedad CHECK (tipo_propiedad IN ('casa', 'departamento', 'terreno', 'local', 'oficina', 'galpon', 'hacienda', 'lote')),
    CONSTRAINT check_operacion CHECK (operacion IN ('venta', 'alquiler', 'anticretico')),
    CONSTRAINT check_estado CHECK (estado IN ('borrador', 'pendiente', 'activa', 'inactiva', 'vendida', 'alquilada', 'rechazada')),
    CONSTRAINT check_precio CHECK (precio > 0)
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `tipo_propiedad`
- `INDEX` en `operacion`
- `INDEX` en `estado`
- `INDEX` en `ciudad`
- `INDEX` en `zona`
- `INDEX` en `slug` (UNIQUE)
- `INDEX` en `user_id`
- `INDEX` en `destacado`
- `INDEX` compuesto: `(estado, tipo_propiedad, operacion)`
- `INDEX` compuesto: `(ciudad, estado)`
- `INDEX` compuesto: `(precio, estado)` para filtros de rango

**Relaciones:**
- Una propiedad pertenece a un usuario
- Una propiedad tiene muchas fotos
- Una propiedad tiene una publicación
- Una propiedad tiene muchas categorías (M:N)

---

### 3. Fotos de Propiedad (property_photos)

**Descripción:** Fotos asociadas a una propiedad

**Tabla:** `fotos_propiedad`

```sql
CREATE TABLE fotos_propiedad (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    property_id INTEGER NOT NULL REFERENCES propiedades(id) ON DELETE CASCADE,
    orden INTEGER NOT NULL DEFAULT 0,
    url VARCHAR(500) NOT NULL,
    url_thumbnail VARCHAR(500),
    descripcion VARCHAR(200),
    es_portada BOOLEAN NOT NULL DEFAULT FALSE
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `property_id`
- `INDEX` en `es_portada`
- `INDEX` compuesto: `(property_id, orden)`

**Relaciones:**
- Una foto pertenece a una propiedad
- ON DELETE CASCADE: Si se borra propiedad, se borran fotos

**Validaciones:**
- Mínimo 3 fotos recomendadas (warning, no hard constraint)
- Máximo 7 fotos (validación en aplicación)
- Una foto marcada como portada por propiedad

---

### 4. Categorías (categories)

**Descripción:** Categorías para clasificar propiedades

**Tabla:** `categorias`

```sql
CREATE TABLE categorias (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    nombre VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    icono VARCHAR(50),
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    orden INTEGER NOT NULL DEFAULT 0
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `UNIQUE INDEX` en `nombre`
- `UNIQUE INDEX` en `slug`
- `INDEX` en `activo`

**Relaciones:**
- Una categoría tiene muchas propiedades (M:N)

---

### 5. Propiedades-Categorías (muchos a muchos)

**Descripción:** Tabla pivote para relación muchos a muchos

**Tabla:** `propiedades_categorias`

```sql
CREATE TABLE propiedades_categorias (
    property_id INTEGER NOT NULL REFERENCES propiedades(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categorias(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (property_id, category_id)
);
```

**Índices:**
- `INDEX` en `property_id`
- `INDEX` en `category_id`

---

### 6. Publicaciones (publications)

**Descripción:** Control de vigencia de publicaciones de propiedades

**Tabla:** `publicaciones`

```sql
CREATE TABLE publicaciones (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    property_id INTEGER NOT NULL UNIQUE REFERENCES propiedades(id) ON DELETE CASCADE,

    -- Fechas
    fecha_inicio TIMESTAMP NOT NULL,
    fecha_fin TIMESTAMP NOT NULL,
    fecha_renovacion TIMESTAMP,

    -- Plan y estado
    plan VARCHAR(20) NOT NULL DEFAULT 'gratis', -- gratis, anual, destacado
    activa BOOLEAN NOT NULL DEFAULT TRUE,
    aprobada BOOLEAN NOT NULL DEFAULT FALSE,
    fecha_aprobacion TIMESTAMP,
    aprobado_por INTEGER REFERENCES usuarios(id),

    -- Renovaciones
    numero_renovaciones INTEGER NOT NULL DEFAULT 0,
    recordatorio_enviado BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT check_plan CHECK (plan IN ('gratis', 'anual', 'destacado'))
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `UNIQUE INDEX` en `property_id`
- `INDEX` en `activa`
- `INDEX` en `aprobada`
- `INDEX` en `recordatorio_enviado`
- `INDEX` en `fecha_fin` (para encontrar las que vencen pronto)

**Relaciones:**
- Una publicación pertenece a una propiedad
- Una publicación tiene muchos pagos

---

### 7. Pagos (payments)

**Descripción:** Registro de pagos por publicaciones

**Tabla:** `pagos`

```sql
CREATE TABLE pagos (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    publication_id INTEGER NOT NULL REFERENCES publicaciones(id),

    -- Información de pago
    monto DECIMAL(10, 2) NOT NULL,
    moneda VARCHAR(10) NOT NULL DEFAULT 'USD',
    metodo VARCHAR(50) NOT NULL, -- qr_bisa, transferencia, tarjeta
    estado VARCHAR(50) NOT NULL DEFAULT 'pendiente', -- pendiente, completado, fallido, expirado, reembolsado

    -- Integración Banco Bisa
    referencia_pago VARCHAR(100) UNIQUE,
    qr_code_url VARCHAR(500),
    transaction_id VARCHAR(200),

    -- Fechas
    fecha_pago TIMESTAMP,
    fecha_expiracion TIMESTAMP,

    -- Notas
    notas TEXT,

    CONSTRAINT check_metodo CHECK (metodo IN ('qr_bisa', 'transferencia', 'tarjeta')),
    CONSTRAINT check_estado CHECK (estado IN ('pendiente', 'completado', 'fallido', 'expirado', 'reembolsado'))
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `publication_id`
- `UNIQUE INDEX` en `referencia_pago`
- `INDEX` en `estado`
- `INDEX` en `fecha_expiracion` (para encontrar expirados)

**Relaciones:**
- Un pago pertenece a una publicación

---

### 8. Clientes (clients) - CRM

**Descripción:** Clientes/leads en el CRM de cada usuario

**Tabla:** `clientes`

```sql
CREATE TABLE clientes (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Información del cliente
    nombre VARCHAR(150) NOT NULL,
    email VARCHAR(150),
    telefono VARCHAR(20),
    whatsapp VARCHAR(20),

    -- Estado y origen
    estado VARCHAR(50) NOT NULL DEFAULT 'nuevo', -- nuevo, contactado, interesado, negociacion, cerrado, perdido
    origen VARCHAR(100), -- web, whatsapp, referencia
    mensaje TEXT,

    -- Propiedad de interés
    property_id INTEGER REFERENCES propiedades(id),

    -- Asignación
    user_id INTEGER NOT NULL REFERENCES usuarios(id),

    -- Seguimiento
    ultimo_contacto TIMESTAMP,
    proximo_contacto TIMESTAMP,
    notas TEXT,

    CONSTRAINT check_estado_cliente CHECK (estado IN ('nuevo', 'contactado', 'interesado', 'negociacion', 'cerrado', 'perdido'))
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `estado`
- `INDEX` en `user_id`
- `INDEX` en `property_id`
- `INDEX` en `proximo_contacto` (para recordatorios)

**Relaciones:**
- Un cliente pertenece a un usuario
- Un cliente puede estar interesado en una propiedad (opcional)
- Un cliente tiene muchas interacciones

---

### 9. Interacciones (interactions) - CRM

**Descripción:** Interacciones con clientes del CRM

**Tabla:** `interacciones`

```sql
CREATE TABLE interacciones (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    client_id INTEGER NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES usuarios(id),

    tipo VARCHAR(50) NOT NULL, -- llamada, whatsapp, email, visita, nota
    notas TEXT,
    fecha TIMESTAMP NOT NULL,

    -- Integración WhatsApp (futuro)
    mensaje_id VARCHAR(200),
    mensaje_enviado BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT check_tipo_interaccion CHECK (tipo IN ('llamada', 'whatsapp', 'email', 'visita', 'nota'))
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `client_id`
- `INDEX` en `user_id`
- `INDEX` en `fecha`

**Relaciones:**
- Una interacción pertenece a un cliente
- Una interacción pertenece a un usuario

---

### 10. Logs de Auditoría (audit_logs)

**Descripción:** Registro de auditoría de todas las acciones

**Tabla:** `audit_logs`

```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Quién realizó la acción
    user_id INTEGER REFERENCES usuarios(id),
    ip_address VARCHAR(50),
    user_agent VARCHAR(500),

    -- Acción realizada
    accion VARCHAR(100) NOT NULL, -- crear, actualizar, eliminar, login, logout, aprobar, rechazar
    entidad VARCHAR(100) NOT NULL, -- usuario, propiedad, cliente, pago
    entidad_id INTEGER,

    -- Cambios
    cambios_anteriores JSONB,
    cambios_nuevos JSONB,

    -- Contexto adicional
    metadata JSONB
);
```

**Índices:**
- `INDEX` en `user_id`
- `INDEX` en `accion`
- `INDEX` en `entidad`
- `INDEX` en `entidad_id`
- `INDEX` en `created_at`
- `INDEX` compuesto: `(user_id, accion, created_at)`
- `INDEX` compuesto: `(entidad, entidad_id, created_at)`

**Partitioning:** Considerar partitioning por `created_at` (mensual o trimestral)

---

### 11. Notificaciones (notifications)

**Descripción:** Notificaciones del sistema para usuarios

**Tabla:** `notificaciones`

```sql
CREATE TABLE notificaciones (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Destinatario
    user_id INTEGER NOT NULL REFERENCES usuarios(id),

    -- Contenido
    tipo VARCHAR(50) NOT NULL, -- renovacion, vencimiento, recordatorio, aprobacion, rechazo
    titulo VARCHAR(200) NOT NULL,
    mensaje TEXT NOT NULL,

    -- Estado
    leida BOOLEAN NOT NULL DEFAULT FALSE,
    fecha_lectura TIMESTAMP,

    -- Acción
    accion_url VARCHAR(500),
    metadata JSONB
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `INDEX` en `user_id`
- `INDEX` en `tipo`
- `INDEX` en `leida`

---

### 12. Configuración del Sistema (system_config)

**Descripción:** Parámetros configurables del sistema

**Tabla:** `configuracion_sistema`

```sql
CREATE TABLE configuracion_sistema (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    clave VARCHAR(100) NOT NULL UNIQUE,
    valor TEXT NOT NULL,
    tipo VARCHAR(20) NOT NULL DEFAULT 'string', -- string, int, bool, json
    categoria VARCHAR(50),
    descripcion TEXT,

    CONSTRAINT check_tipo_config CHECK (tipo IN ('string', 'int', 'bool', 'json'))
);
```

**Índices:**
- `INDEX` en `deleted_at`
- `UNIQUE INDEX` en `clave`
- `INDEX` en `categoria`

**Configuraciones iniciales:**
```sql
INSERT INTO configuracion_sistema (clave, valor, tipo, categoria, descripcion) VALUES
('activar_aprobacion_propiedades', 'true', 'bool', 'publicaciones', 'Admin debe aprobar propiedades antes de publicar'),
('periodo_prueba_gratis', 'true', 'bool', 'pagos', 'Activar período de prueba gratuita de 2 meses'),
('meses_gratis', '2', 'int', 'pagos', 'Número de meses de prueba gratuita'),
('vigencia_publicacion_meses', '12', 'int', 'publicaciones', 'Meses de vigencia de una publicación'),
('dias_aviso_renovacion', '5', 'int', 'publicaciones', 'Días antes de vencer para enviar notificación'),
('fotos_minimo_recomendado', '3', 'int', 'propiedades', 'Mínimo de fotos recomendado'),
('fotos_maximo_permitido', '7', 'int', 'propiedades', 'Máximo de fotos permitido'),
('compartir_personalizado_precio', '50', 'int', 'pagos', 'Precio de función de compartir personalizado (USD)');
```

---

## 🔍 Optimizaciones de Consulta

### Consultas más frecuentes:

**1. Listado de propiedades con filtros**
```sql
-- Con índices compuestos
SELECT p.*, u.nombre, u.telefono
FROM propiedades p
JOIN usuarios u ON p.user_id = u.id
WHERE p.estado = 'activa'
  AND p.tipo_propiedad IN ('casa', 'departamento')
  AND p.operacion = 'venta'
  AND p.precio BETWEEN 10000 AND 50000
  AND p.ciudad = 'Santa Cruz'
ORDER BY p.destacado DESC, p.created_at DESC
LIMIT 20 OFFSET 0;

-- Índice necesario:
CREATE INDEX idx_propiedades_busqueda ON propiedades(
    estado, tipo_propiedad, operacion, ciudad, destacado, created_at
) WHERE estado = 'activa';
```

**2. Propiedades por vencer pronto**
```sql
SELECT p.*, p.fecha_fin
FROM publicaciones pub
JOIN propiedades p ON pub.property_id = p.id
WHERE pub.activa = TRUE
  AND pub.recordatorio_enviado = FALSE
  AND pub.fecha_fin BETWEEN NOW() AND NOW() + INTERVAL '5 days';

-- Índice necesario:
CREATE INDEX idx_publicaciones_vencer ON publicaciones(
    activa, recordatorio_enviado, fecha_fin
) WHERE activa = TRUE;
```

**3. Clientes con próximo contacto**
```sql
SELECT *
FROM clientes
WHERE user_id = ?
  AND proximo_contacto <= NOW()
ORDER BY proximo_contacto ASC;

-- Índice necesario:
CREATE INDEX idx_clientes_proximo_contacto ON clientes(user_id, proximo_contacto);
```

---

## 📊 Estadísticas de Tamaño

### Estimación de registros por año:

| Entidad | Registros/año | Tamaño/registro | Total/año |
|---------|---------------|-----------------|-----------|
| usuarios | 1,000 | 1 KB | 1 MB |
| propiedades | 5,000 | 2 KB | 10 MB |
| fotos_propiedad | 25,000 | 0.3 KB | 7.5 MB |
| publicaciones | 5,000 | 0.5 KB | 2.5 MB |
| pagos | 5,000 | 1 KB | 5 MB |
| clientes | 20,000 | 1 KB | 20 MB |
| interacciones | 50,000 | 0.5 KB | 25 MB |
| audit_logs | 500,000 | 2 KB | 1 GB |

**Total BD (aprox 1 año):** ~1.3 GB + Storage de fotos

---

## 🔒 Restricciones y Validaciones

### A nivel de base de datos (CHECK constraints):

1. **Precio > 0** en propiedades
2. **Email único** en usuarios
3. **Slug único** en propiedades
4. **Rol válido** en usuarios
5. **Estado válido** en propiedades y publicaciones
6. **Tipo válido** en propiedades

### A nivel de aplicación (validaciones Go):

1. **Mínimo 3 fotos recomendado** (warning)
2. **Máximo 7 fotos** (hard validation)
3. **Formato de email válido**
4. **Teléfono con prefijo de país**
5. **Latitud/longitud válidas**
6. **Fechas lógicas** (fecha_fin > fecha_inicio)

---

## 📝 Migraciones

### Orden de creación de tablas:

1. `usuarios`
2. `categorias`
3. `propiedades`
4. `fotos_propiedad`
5. `propiedades_categorias`
6. `publicaciones`
7. `pagos`
8. `clientes`
9. `interacciones`
10. `audit_logs`
11. `notificaciones`
12. `configuracion_sistema`

### Seeders (datos iniciales):

```sql
-- Categorías iniciales
INSERT INTO categorias (nombre, slug, icono, orden) VALUES
('Residencial', 'residencial', 'home', 1),
('Comercial', 'comercial', 'building', 2),
('Industrial', 'industrial', 'factory', 3),
('Terrenos', 'terrenos', 'land', 4);

-- Configuración inicial
-- (ver sección de configuración_sistema arriba)
```

---

**Última actualización:** 2026-04-09
**Versión:** 1.0.0
**Autor:** Scrum Master 365Soft
