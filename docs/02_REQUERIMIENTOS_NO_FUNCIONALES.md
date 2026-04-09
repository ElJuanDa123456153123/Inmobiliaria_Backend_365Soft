# ⚡ Requerimientos No Funcionales - SaaS Inmobiliario 365Soft

## 🎯 Visión General

Este documento define los requerimientos técnicos y de calidad que el sistema debe cumplir para garantizar performance, seguridad, escalabilidad y mantenibilidad.

---

## 🚀 Performance

### P1. Tiempos de Respuesta

| Endpoint | Tiempo Máximo | Target | Métrica |
|----------|---------------|--------|---------|
| GET /api/propiedades (listado) | 500ms | 200ms | Percentil 95 |
| GET /api/propiedades/:id (detalle) | 300ms | 100ms | Percentil 95 |
| POST /api/propiedades (crear) | 1s | 500ms | Percentil 95 |
| POST /api/propiedades/:id/fotos (upload) | 3s | 1.5s | Por foto (5MB) |
| GET /api/buscar (búsqueda) | 1s | 300ms | Percentil 95 |
| POST /api/pagos/qr/generar | 2s | 1s | Percentil 95 |

**Justificación:**
- Listados: 500ms máximo para UX aceptable
- Detalle: 300ms para sensación de instantaneidad
- Upload: 3s por foto considerando procesamiento (resize, optimización)
- Búsqueda: 1s máximo con filtros complejos

### P2. Concurrencia

**Requerimientos:**
- **Mínimo:** 100 usuarios simultáneos
- **Óptimo:** 500 usuarios simultáneos
- **Máximo:** 1000 usuarios simultáneos

**Escenarios de carga:**
- Pico de visitas: 500 requests/segundo
- Situación normal: 50 requests/segundo
- Hora pico: 200 requests/segundo

### P3. Throughput

**Capacidad de procesamiento:**
- **Mínimo:** 1,000 transacciones por minuto
- **Óptimo:** 5,000 transacciones por minuto

**Cáculos:**
- Publicación de propiedad: ~10 transacciones (crear, upload fotos, generar pago)
- Búsqueda: ~2 transacciones (consulta + fetch fotos)
- Login: ~3 transacciones (validar, generar token, retornar datos)

### P4. Optimizaciones Obligatorias

**Base de datos:**
- Índices en todos los campos de búsqueda
- Índices compuestos para filtros combinados
- Connection pooling óptimo (máx 50 conexiones)
- Query optimization (evitar N+1)

**Cache:**
- Propiedades populares: 1 hora en Redis
- Listados: 15 minutos en Redis
- Categorías: 24 horas en Redis
- Usuario actual: 30 minutos en Redis

**Imágenes:**
- Redimensionar a máximo 1920x1080px
- Generar thumbnail de 300x300px
- Comprimir con calidad 85%
- Servir en WebP cuando sea posible
- CDN para delivery (Cloudflare)

**API:**
- Paginación obligatoria (máx 50 items por página)
- Comprimir responses con gzip
- Eager loading (GORM Preload) para evitar N+1
- Select fields específicos (no siempre retornar todo)

---

## 🔒 Seguridad

### S1. Autenticación y Autorización

**JWT Tokens:**
- Algoritmo: RS256 (RSA) o HS256 (HMAC)
- Expiración: 24 horas access token
- Refresh token: 7 días
- Firma: Verificar firma en cada request
- Revocación: Blacklist de tokens en Redis

**Contraseñas:**
- Hash: bcrypt con cost factor 12
- Longitud mínima: 8 caracteres
- Requisitos: Mayúscula, minúscula, número, especial
- No se pueden repetir últimas 5 contraseñas
- Expiración: No (a menos que se implemente)

**OAuth Google:**
- Validar token de Google
- Verificar email
- Crear usuario si no existe
- Generar JWT propio

### S2. Validación de Inputs

**Sanitización obligatoria:**
- Todos los inputs de usuario
- Prevenir SQL injection (GORM ya lo hace)
- Prevenir XSS (escapar HTML)
- Prevenir CSRF (tokens en forms)

**Validaciones:**
- Email: Formato válido + unicidad
- Teléfono: Prefijo país + formato
- Fotos: Validar magic bytes (no solo extensión)
- Archivos: Máximo 5MB por foto
- Strings: Longitud máxima en todos los campos
- Números: Rangos válidos

### S3. Datos Sensibles

**Encriptación:**
- Contraseñas: bcrypt (siempre hash)
- Teléfonos: Encriptación AES-256 (opcional)
- Emails: Encriptación AES-256 (opcional)
- Tokens: Nunca exponer en logs

**Logs:**
- NUNCA loggear:
  - Contraseñas
  - Tokens completos
  - Datos de tarjetas (cuando se implemente)
  - Información personal sensible

**API Keys:**
- Banco Bisa: Variables de entorno
- Cloudflare R2: Variables de entorno
- Redis: Variables de entorno
- Nunca commitear al repo

### S4. Headers de Seguridad

**Headers obligatorios en todas las respuestas:**
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### S5. Rate Limiting

**Límites por usuario:**
- Endpoints públicos: 100 requests/minuto
- Endpoints protegidos: 200 requests/minuto
- Login: 5 intentos/15 minutos
- Upload: 10 uploads/minuto

**Rate limiting distribuido:**
- Implementar con Redis
- Bloquear por IP + user ID
- Response: 429 Too Many Requests

### S6. CORS

**Política de CORS:**
- Orígenes permitidos: Configurable en .env
- Methods: GET, POST, PUT, DELETE, PATCH
- Headers: Authorization, Content-Type
- Credentials: true
- Max age: 3600

### S7. HTTPS Obligatorio

**Producción:**
- Todas las conexiones deben ser HTTPS
- HTTP debe redirigir a HTTPS
- Certificado SSL válido (Let's Encrypt)

---

## 📈 Escalabilidad

### E1. Base de Datos

**PostgreSQL:**
- Índices optimizados para búsqueda
- Partitioning por fecha (audit_log)
- Connection pooling: PgBouncer
- Read replicas cuando sea necesario

**Capacidad:**
- **Mínimo:** 10,000 propiedades
- **Óptimo:** 100,000 propiedades
- **Máximo:** 1,000,000 propiedades

### E2. Storage

**Cloudflare R2:**
- Storage ilimitado
- No egress fees (importante para fotos)
- CDN incluido
- Backup automático

**Estimación:**
- 1 propiedad: ~5 fotos × 500KB = 2.5MB
- 10,000 propiedades: ~25GB
- 100,000 propiedades: ~250GB

### E3. Cache

**Redis:**
- Cache distribuido
- Persistencia RDB + AOF
- TTLs configurables
- Cluster para alta disponibilidad

**Usos:**
- Cache de propiedades
- Cache de búsquedas
- Rate limiting
- Sessions
- Blacklist de tokens

### E4. Arquitectura Escalable

**Preparado para:**
- Horizontal scaling (múltiples instancias de API)
- Load balancer (Nginx, HAProxy)
- Microservicios en el futuro
- Multi-tenant (SaaS para múltiples clientes)

---

## 🔧 Disponibilidad

### D1. Uptime

**Meta:**
- **Mínimo:** 99% uptime (3.65 días/año de downtime)
- **Óptimo:** 99.9% uptime (8.76 horas/año de downtime)
- **Excelente:** 99.99% uptime (52.56 minutos/año de downtime)

### D2. Backups

**Base de datos:**
- **Automático:** Backup diario a las 3:00 AM
- **Retención:** 30 días
- **Tipo:** Backup físico + WAL
- **Test de restauración:** Mensual

**Storage (R2):**
- Backup automático por Cloudflare
- Versioning de archivos
- Retention policy: 90 días

**Redis:**
- RDB snapshot cada 24 horas
- AOF log cada 1 segundo
- Retention: 7 días

### D3. Monitoring

**Uptime monitoring:**
- UptimeRobot o Pingdom
- Check cada 1 minuto
- Alerta por email si downtime > 5 minutos

**Error tracking:**
- Sentry para errores de aplicación
- Alerta en tiempo real
- Contexto completo del error

**Logs:**
- ELK stack (Elasticsearch, Logstash, Kibana)
- Logs centralizados
- Búsqueda y análisis

---

## 🧪 Mantenibilidad

### M1. Código Limpio

**Patrones obligatorios:**
- Repository Pattern
- Service Layer
- DTO Pattern
- Dependency Injection

**Convenciones:**
- Archivos: `snake_case.go`
- Variables exportadas: `PascalCase`
- Variables privadas: `camelCase`
- Constantes: `PascalCase`
- Comentarios en español

**Complejidad:**
- Máximo 10 parámetros por función
- Máximo 50 líneas por función
- Máximo 3 niveles de anidación
- Cyclomatic complexity < 10

### M2. Testing

**Cobertura de tests:**
- **Mínimo:** 60% cobertura
- **Óptimo:** 80% cobertura
- **Crítico:** 100% cobertura en services

**Tipos de tests:**
- Unit tests: Lógica de negocio
- Integration tests: API endpoints
- E2E tests: Flujos críticos

**Automatización:**
- Tests en cada PR
- Tests en cada merge a main
- Tests en CI/CD

### M3. Documentación

**Código:**
- Comentarios en funciones complejas
- Godoc para paquetes exportados
- Ejemplos de uso

**API:**
- Swagger/OpenAPI documentation
- Ejemplos de request/response
- Códigos de error documentados

**Arquitectura:**
- Diagramas actualizados
- Decisiones recordadas (ADR)
- Wiki del proyecto

### M4. Configuración

**Environment variables:**
- Todo configurable por .env
- Nunca hardcodear valores
- .env.example con template
- Validación de variables al inicio

**Feature flags:**
- Sistema de feature flags
- Activar/desactivar sin deploy
- Configurable por admin

---

## 🌐 Compatibilidad

### C1. Navegadores Soportados

**Desktop:**
- Chrome/Edge: Últimas 2 versiones
- Firefox: Últimas 2 versiones
- Safari: Últimas 2 versiones

**Mobile:**
- iOS Safari: iOS 13+
- Android Chrome: Android 8+
- Chrome Mobile: Última versión

### C2. Responsive Design

**Breakpoints:**
- Mobile: < 768px
- Tablet: 768px - 1024px
- Desktop: > 1024px

**Requerimientos:**
- Mobile-first approach
- Touch-friendly (botones mínimo 44x44px)
- Optimizado para redes lentas

### C3. API Versioning

**Estrategia:**
- Version por URL: `/api/v1/...`
- Soportar última versión + versión anterior
- Deprecation notice con 6 meses de anticipación

---

## ♿ Accesibilidad

### A1. Estándares

**WCAG 2.1 AA:**
- Contraste de colores mínimo 4.5:1
- Texto escalable hasta 200%
- Navegación por teclado
- Lectores de pantalla compatibles
- Alt text en imágenes

### A2. Internacionalización

**Idiomas soportados:**
- Español (default)
- Inglés (futuro)
- Portugués (futuro)

**Formatos:**
- Moneda: Según configuración
- Fecha: Según configuración
- Número: Según configuración

---

## 🔍 Monitoreo y Métricas

### M1. Métricas de Aplicación

**Prometheus metrics:**
- Requests por segundo
- Tiempo de respuesta (p50, p95, p99)
- Tasa de error
- Uso de CPU
- Uso de memoria
- Conexiones a BD

### M2. Business Metrics

**KPIs:**
- Usuarios registrados por día
- Propiedades publicadas por día
- Pagos completados por día
- Tasa de conversión (registro → publicación)
- Churn rate

### M3. Alerts

**Alertas obligatorias:**
- Downtime > 5 minutos
- Error rate > 5%
- Tiempo de respuesta > 1s (p95)
- CPU > 80% por 5 minutos
- Memoria > 80% por 5 minutos
- Espacio en disco < 20%

---

## 🗂️ Auditoría y Compliance

### A1. Logs de Auditoría

**Registrar:**
- Login/logout
- Crear/actualizar/eliminar propiedades
- Pagos
- Cambios de configuración
- Acciones de admin

**Formato:**
- Timestamp
- User ID
- IP address
- Acción
- Entidad afectada
- Cambios (before/after)

### A2. Retención de Datos

**Logs de aplicación:**
- Retención: 90 días
- Rotación: Diaria
- Compresión: gz

**Logs de auditoría:**
- Retención: 5 años
- Inmutable (no se puede modificar)

**Datos de usuario:**
- Derecho al olvido ( GDPR/LPDP)
- Exportar datos (JSON)
- Eliminar cuenta (soft delete + cleanup en 30 días)

### A3. Legal

**Términos de uso:**
- Aceptación obligatoria al registro
- Versión y fecha de aceptación
- Histórico de cambios

**Política de privacidad:**
- Datos recolectados
- Uso de datos
- Compartir con terceros
- Cookies

---

## 🎯 Matriz de Prioridades

| Requerimiento | Prioridad | MVP | Fase 2 |
|---------------|-----------|-----|--------|
| Tiempos de respuesta < 500ms | P0 | ✅ | |
| HTTPS obligatorio | P0 | ✅ | |
| Rate limiting | P0 | ✅ | |
| Validación de inputs | P0 | ✅ | |
| Backup diario | P0 | ✅ | |
| 99% uptime | P0 | ✅ | |
| 100 usuarios concurrentes | P0 | ✅ | |
| Tests 60% cobertura | P1 | ✅ | |
| 500 usuarios concurrentes | P1 | | ✅ |
| 99.9% uptime | P1 | | ✅ |
| Read replicas BD | P1 | | ✅ |
| WCAG 2.1 AA | P2 | | ✅ |
| Internacionalización | P2 | | ✅ |
| 1000 usuarios concurrentes | P2 | | ✅ |

---

**Última actualización:** 2026-04-09
**Versión:** 1.0.0
**Autor:** Scrum Master 365Soft
