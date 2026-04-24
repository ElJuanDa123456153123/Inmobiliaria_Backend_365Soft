# Carpetas Necesarias para el Backend

## 📁 Estructura de Carpetas

El backend necesita las siguientes carpetas para funcionar correctamente:

```
Inmobiliaria_Backend_365Soft/
└── uploads/
    └── imagenes/          ← Aquí se guardan las imágenes de las propiedades
```

## 🚀 Creación de Carpetas

### Windows (CMD)
```bash
mkdir uploads
mkdir uploads\imagenes
```

### Windows (PowerShell)
```powershell
New-Item -ItemType Directory -Path "uploads\imagenes" -Force
```

### Linux/Mac
```bash
mkdir -p uploads/imagenes
```

## ⚠️ Problemas Comunes

### 1. Error: "Imagen no encontrada" (404)

**Causa:** La carpeta `uploads/imagenes` no existe o está vacía.

**Solución:**
1. Verificar que la carpeta existe
2. Volver a publicar la propiedad con imágenes

**Verificación:**
```bash
# Windows
dir uploads\imagenes

# Linux/Mac
ls -la uploads/imagenes
```

### 2. Error: "Error al crear directorio de uploads"

**Causa:** El backend no tiene permisos para crear carpetas.

**Solución:**
1. Crear la carpeta manualmente (ver arriba)
2. Dar permisos de escritura a la carpeta `uploads`

### 3. Las imágenes están en la BD pero no se ven

**Causa:** Los registros existen en `propiedades_imagenes` pero los archivos físicos no.

**Verificación:**
```sql
-- Ver cuántas imágenes hay en la BD
SELECT COUNT(*) FROM propiedades_imagenes;

-- Ver las rutas de las imágenes
SELECT image_url FROM propiedades_imagenes LIMIT 10;
```

**Solución:**
- Volver a subir las imágenes editando cada propiedad

## 📊 Estadísticas de Almacenamiento

Una imagen promedio: 200 KB - 500 KB
100 propiedades (3 fotos c/u): ~60 MB - 150 MB
1000 propiedades (3 fotos c/u): ~600 MB - 1.5 GB

## 🔍 Verificación Rápida

**Para verificar que todo funciona:**

1. **Backend corriendo:**
   ```bash
   npm run dev
   ```

2. **Carpeta existe:**
   ```bash
   dir uploads\imagenes
   ```

3. **Probar subida:**
   - Crear una propiedad de prueba
   - Subir 1-2 imágenes
   - Verificar que aparecen en `uploads/imagenes`

4. **Probar visualización:**
   - Abrir el navegador en: `http://localhost:3001/api/imagenes/[nombre-imagen]`
   - Deberías ver la imagen

## 💡 Buenas Prácticas

1. **Backup:** Haz backup regular de la carpeta `uploads/imagenes`
2. **Limpieza:** Elimina imágenes de propiedades borradas/anticréticas
3. **Optimización:** Las imágenes se optimizan automáticamente (máx 1920px)
4. **Monitoreo:** Verifica el tamaño de la carpeta regularmente

## 🛠️ Solución de Problemas

**Si las imágenes no se ven:**
1. Verifica que el backend esté corriendo
2. Verifica que la carpeta `uploads/imagenes` exista
3. Verifica que los archivos existan en la carpeta
4. Revisa la consola del backend para errores
5. Revisa la consola del navegador (F12) para errores 404

**Si necesitas más ayuda:**
- Revisa los logs del backend (consola donde corre `npm run dev`)
- Los errores ahora muestran mensajes detallados con la ruta y el problema específico
