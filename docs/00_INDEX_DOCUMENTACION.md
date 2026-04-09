# 📚 Documentación del Sistema - SaaS Inmobiliario 365Soft

## 🎯 Visión General

Sistema inmobiliario B2B/B2C para publicación y búsqueda de propiedades en Bolivia con modelo de negocio SaaS.

**Stack Tecnológico:**
- Backend: Go + Fiber + PostgreSQL + GORM
- Frontend: Next.js
- Pagos: Banco Bisa (QR)
- Storage: Cloudflare R2
- Cache: Redis

---

## 📑 Estructura de Documentación

Esta documentación está organizada por módulos para facilitar el trabajo del equipo de desarrollo.

### 📋 Documentos Principales

| Documento | Descripción | Audiencia |
|-----------|-------------|-----------|
| [01_REQUERIMIENTOS_FUNCIONALES.md](./01_REQUERIMIENTOS_FUNCIONALES.md) | Requerimientos funcionales completos | Todo el equipo |
| [02_REQUERIMIENTOS_NO_FUNCIONALES.md](./02_REQUERIMIENTOS_NO_FUNCIONALES.md) | Requerimientos no funcionales | Equipo backend + DevOps |
| [03_MODELO_DE_DATOS.md](./03_MODELO_DE_DATOS.md) | Modelo de datos completo | Backend + DBA |
| [04_API_ENDPOINTS.md](./04_API_ENDPOINTS.md) | Endpoints de la API | Backend + Frontend |
| [05_ARQUITECTURA.md](./05_ARQUITECTURA.md) | Arquitectura técnica | Backend |
| [06_FLUJOS_DE_NEGOCIO.md](./06_FLUJOS_DE_NEGOCIO.md) | Diagramas de flujo | Todo el equipo |
| [07_PLAN_DE_PRUEBAS.md](./07_PLAN_DE_PRUEBAS.md) | Estrategia de pruebas | QA + Backend |
| [08_DEPLOYMENT.md](./08_DEPLOYMENT.md) | Guía de deployment | DevOps |

### 📁 Documentos por Módulo

#### Módulo de Autenticación
- [modules/auth/REQUERIMIENTOS.md](./modules/auth/REQUERIMIENTOS.md)
- [modules/auth/API.md](./modules/auth/API.md)
- [modules/auth/DIAGRAMA.md](./modules/auth/DIAGRAMA.md)

#### Módulo de Propiedades
- [modules/propiedades/REQUERIMIENTOS.md](./modules/propiedades/REQUERIMIENTOS.md)
- [modules/propiedades/API.md](./modules/propiedades/API.md)
- [modules/propiedades/DIAGRAMA.md](./modules/propiedades/DIAGRAMA.md)

#### Módulo de Publicaciones
- [modules/publicaciones/REQUERIMIENTOS.md](./modules/publicaciones/REQUERIMIENTOS.md)
- [modules/publicaciones/API.md](./modules/publicaciones/API.md)
- [modules/publicaciones/DIAGRAMA.md](./modules/publicaciones/DIAGRAMA.md)

#### Módulo de Pagos (Banco Bisa)
- [modules/pagos/REQUERIMIENTOS.md](./modules/pagos/REQUERIMIENTOS.md)
- [modules/pagos/API.md](./modules/pagos/API.md)
- [modules/pagos/INTEGRACION_BISA.md](./modules/pagos/INTEGRACION_BISA.md)

#### Módulo CRM
- [modules/crm/REQUERIMIENTOS.md](./modules/crm/REQUERIMIENTOS.md)
- [modules/crm/API.md](./modules/crm/API.md)
- [modules/crm/DIAGRAMA.md](./modules/crm/DIAGRAMA.md)

#### Módulo Admin
- [modules/admin/REQUERIMIENTOS.md](./modules/admin/REQUERIMIENTOS.md)
- [modules/admin/API.md](./modules/admin/API.md)
- [modules/admin/CONFIGURACION.md](./modules/admin/CONFIGURACION.md)

---

## 🚀 Guía Rápida de Inicio

### Para el Scrum Master
1. Leer [`01_REQUERIMIENTOS_FUNCIONALES.md`](./01_REQUERIMIENTOS_FUNCIONALES.md) - Visión completa del negocio
2. Leer [`06_FLUJOS_DE_NEGOCIO.md`](./06_FLUJOS_DE_NEGOCIO.md) - Entender los procesos
3. Usar los documentos por módulo para asignar tareas al equipo

### Para Desarrolladores Backend
1. Leer [`03_MODELO_DE_DATOS.md`](./03_MODELO_DE_DATOS.md) - Estructura de BD
2. Leer [`04_API_ENDPOINTS.md`](./04_API_ENDPOINTS.md) - Contrato de API
3. Leer [`05_ARQUITECTURA.md`](./05_ARQUITECTURA.md) - Patrones y estructura
4. Profundizar en los documentos de SU módulo asignado

### Para Desarrolladores Frontend
1. Leer [`01_REQUERIMIENTOS_FUNCIONALES.md`](./01_REQUERIMIENTOS_FUNCIONALES.md) - Comprender el negocio
2. Leer [`04_API_ENDPOINTS.md`](./04_API_ENDPOINTS.md) - Integración con backend
3. Revisar diagramas en [`06_FLUJOS_DE_NEGOCIO.md`](./06_FLUJOS_DE_NEGOCIO.md)

---

## 📊 Roadmap del Proyecto

### Fase 1: Fundación (Semanas 1-2)
- Estructura base del proyecto
- Configuración de infraestructura
- Modelos de datos core

### Fase 2: Módulo de Propiedades (Semanas 3-5)
- CRUD completo de propiedades
- Upload de fotos
- Búsqueda y filtrado

### Fase 3: Módulo de Publicaciones y Pagos (Semanas 6-8)
- Sistema de publicaciones
- Integración Banco Bisa
- Gestión de vigencias

### Fase 4: Módulo CRM (Semanas 9-10)
- Gestión de clientes
- Interacciones
- Dashboard

### Fase 5: Módulo Admin y Configuración (Semanas 11-12)
- Panel administrativo
- Configuración del sistema
- Estadísticas

---

## 👥 Organización del Equipo

### Estrategia de Trabajo para 3 Personas

El proyecto está diseñado para minimizar conflictos y maximizar paralelismo:

#### Dev 1: Backend Lead - Core y Pagos
**Responsabilidades:**
- Módulo de Autenticación
- Módulo de Publicaciones
- Módulo de Pagos (Banco Bisa)
- Arquitectura general
- Code review del equipo

**Módulos críticos:**
- `auth/`
- `publicaciones/`
- `pagos/`

#### Dev 2: Backend Dev - Propiedades y Búsqueda
**Responsabilidades:**
- Módulo de Propiedades
- Sistema de búsqueda y filtros
- Upload de archivos (Cloudflare R2)
- Cache (Redis)

**Módulos críticos:**
- `propiedades/`
- `storage/`
- `search/`

#### Dev 3: Backend Dev - CRM y Admin
**Responsabilidades:**
- Módulo CRM
- Módulo Admin
- Sistema de notificaciones
- Workers y background jobs

**Módulos críticos:**
- `crm/`
- `admin/`
- `workers/`

### Reglas de Convivencia

1. **Contratos primero:** Antes de implementar, definir API endpoints en documento compartido
2. **Branches por feature:** Cada módulo en branch separado
3. **Integración continua:** Merges diarios a `develop` para detectar conflictos temprano
4. **Code review:** Todo código debe ser revisado por al menos otro dev
5. **Documentación viva:** Actualizar docs junto con el código

---

## 🔧 Convenciones del Código

### Estructura de Archivos
```
internal/
├── models/           # Modelos de datos (GORM)
├── handlers/         # Controladores HTTP (Fiber)
├── services/         # Lógica de negocio
├── repository/       # Acceso a datos
├── middleware/       # Middleware de Fiber
├── dto/             # Request/Response objects
├── validators/      # Validaciones
└── utils/           # Utilidades
```

### Patrones de Arquitectura
- **Repository Pattern:** Separar lógica de negocio de acceso a datos
- **Service Layer:** Contener la lógica de negocio
- **DTO Pattern:** Controlar exposición de datos en API
- **Middleware Chain:** Auth → Roles → Audit → Handler

### Nomenclatura
- **Archivos:** `snake_case.go`
- **Variables/Funciones:** `PascalCase` para exportadas, `camelCase` para privadas
- **Constantes:** `PascalCase`
- **Base de datos:** `snake_case`

---

## 📞 Soporte y Comunicación

### Canales de Comunicación
- **Dudas técnicas:** Discord/Slack #backend-dev
- **Revisions:** GitHub Pull Requests
- **Emergencias:** Llamar al Backend Lead

### Documentación en Vivo
Esta documentación se mantiene actualizada con los cambios del proyecto. Cualquier modificación en el alcance debe reflejarse aquí.

---

**Última actualización:** 2026-04-09
**Versión:** 1.0.0
**Estado:** Planeación
