package jobs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"gorm.io/gorm"
)

// SavedSearchWorker - Worker para verificar coincidencias de búsquedas guardadas
type SavedSearchWorker struct {
	DB *config.Database
}

// NewSavedSearchWorker - Crea un nuevo worker de búsquedas guardadas
func NewSavedSearchWorker() (*SavedSearchWorker, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &SavedSearchWorker{
		DB: db,
	}, nil
}

// CheckSavedSearchMatches - Verifica propiedades creadas recientemente contra todas las búsquedas guardadas activas
func (w *SavedSearchWorker) CheckSavedSearchMatches() error {
	fmt.Printf("🔍 [%s] Iniciando verificación de búsquedas guardadas...\n", time.Now().Format("15:04:05"))

	// 1. Obtener todas las búsquedas guardadas activas
	var activeSearches []models.SavedSearch
	err := w.DB.DB.Where("is_active = ?", true).Find(&activeSearches).Error
	if err != nil {
		fmt.Printf("❌ Error obteniendo búsquedas guardadas: %v\n", err)
		return err
	}

	if len(activeSearches) == 0 {
		fmt.Println("✓ No hay búsquedas guardadas activas para procesar")
		return nil
	}

	fmt.Printf("📋 Procesando %d búsquedas guardadas activas\n", len(activeSearches))

	// 2. Definir límite de tiempo para buscar propiedades nuevas (ej. creadas en las últimas 24 horas para garantizar cobertura)
	timeLimit := time.Now().Add(-24 * time.Hour)

	// Procesar cada búsqueda
	notificationsCreated := 0
	for _, search := range activeSearches {
		var filters map[string]interface{}
		if err := json.Unmarshal(search.Filtros, &filters); err != nil {
			fmt.Printf("❌ Error decodificando filtros para búsqueda ID %d: %v\n", search.ID, err)
			continue
		}

		// Construir la consulta de coincidencia
		query := w.BuildMatchQuery(filters)

		// Filtrar solo propiedades creadas recientemente
		query = query.Where("created_at >= ?", timeLimit)

		var matchingProperties []models.Property
		err := query.Find(&matchingProperties).Error
		if err != nil {
			fmt.Printf("❌ Error buscando propiedades coincidentes para búsqueda ID %d: %v\n", search.ID, err)
			continue
		}

		if len(matchingProperties) == 0 {
			continue
		}

		fmt.Printf("✨ Búsqueda '%s' (Usuario %d): Encontradas %d propiedades potenciales\n", search.Nombre, search.UserID, len(matchingProperties))

		// Crear notificaciones para propiedades que no han sido notificadas aún para esta búsqueda
		for _, prop := range matchingProperties {
			var existingNotif models.Notification
			err := w.DB.DB.Where("saved_search_id = ? AND property_id = ?", search.ID, prop.ID).First(&existingNotif).Error
			
			// Si no existe notificación previa para esta combinación, la creamos
			if err == gorm.ErrRecordNotFound {
				mensaje := fmt.Sprintf("¡Nueva coincidencia! La propiedad '%s' en zona '%s' coincide con tu búsqueda guardada '%s'.", prop.Titulo, prop.Zona, search.Nombre)
				
				notif := models.Notification{
					UserID:        search.UserID,
					SavedSearchID: &search.ID,
					PropertyID:    prop.ID,
					Mensaje:       mensaje,
					IsRead:        false,
				}

				if err := w.DB.DB.Create(&notif).Error; err != nil {
					fmt.Printf("❌ Error creando notificación para usuario %d: %v\n", search.UserID, err)
				} else {
					notificationsCreated++
					fmt.Printf("🔔 Notificación creada con éxito para usuario %d sobre propiedad ID %d\n", search.UserID, prop.ID)
				}
			}
		}
	}

	fmt.Printf("🎉 Verificación completada. %d nuevas notificaciones creadas\n", notificationsCreated)
	return nil
}

// StartWorkerScheduler - Inicia el planificador del worker en segundo plano
func (w *SavedSearchWorker) StartWorkerScheduler(intervalMinutes int) {
	fmt.Printf("⏰ Worker de Búsquedas Guardadas iniciado: Ejecutando verificación cada %d minutos\n", intervalMinutes)

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Ejecutar inmediatamente al inicio
	_ = w.CheckSavedSearchMatches()

	for range ticker.C {
		_ = w.CheckSavedSearchMatches()
	}
}

// BuildMatchQuery - Duplica la lógica de matching de propiedades de forma segura
func (w *SavedSearchWorker) BuildMatchQuery(filters map[string]interface{}) *gorm.DB {
	query := w.DB.DB.Model(&models.Property{}).Where("estado_propiedad = ?", "activo")

	// Operación
	operation := getStringField(filters, "tipo_operacion", "operation")
	if operation != "" && operation != "todas" && operation != "todos" {
		query = query.Where("tipo_operacion = ?", operation)
	}

	// Tipo de propiedad
	propType := getStringField(filters, "tipo_propiedad", "type")
	if propType != "" && propType != "todas" && propType != "todos" {
		query = query.Where("tipo_propiedad = ?", propType)
	}

	// Zona / Dirección / Título
	zona := getStringField(filters, "zona", "search")
	if zona != "" {
		searchTerm := "%" + zona + "%"
		query = query.Where("(zona ILIKE ? OR direccion ILIKE ? OR titulo ILIKE ? OR descripcion ILIKE ?)", searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// Moneda
	moneda := getStringField(filters, "moneda", "")
	if moneda != "" && moneda != "todas" && moneda != "todos" {
		query = query.Where("moneda = ?", moneda)
	}

	// Precios
	if priceMin, err := getFloatField(filters, "precio_min", "priceMin"); err == nil && priceMin > 0 {
		query = query.Where("precio >= ?", priceMin)
	}
	if priceMax, err := getFloatField(filters, "precio_max", "priceMax"); err == nil && priceMax > 0 {
		query = query.Where("precio <= ?", priceMax)
	}

	// Dormitorios (habitaciones)
	dormitorios := getStringField(filters, "dormitorios", "bedrooms")
	if dormitorios != "" && dormitorios != "todas" && dormitorios != "todos" {
		query = query.Where("dormitorios = ?", dormitorios)
	}

	// Baños
	banos := getStringField(filters, "banos", "bathrooms")
	if banos != "" && banos != "todas" && banos != "todos" {
		query = query.Where("banos = ?", banos)
	}

	// Garaje
	garaje := getStringField(filters, "garaje", "")
	if garaje != "" && garaje != "todas" && garaje != "todos" {
		query = query.Where("garaje = ?", garaje)
	}

	return query
}

// Helpers locales para parseo seguro de datos de JSON maps
func getStringField(filters map[string]interface{}, key1, key2 string) string {
	if v, ok := filters[key1]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if key2 != "" {
		if v, ok := filters[key2]; ok && v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getFloatField(filters map[string]interface{}, key1, key2 string) (float64, error) {
	var val interface{}
	var ok bool
	if val, ok = filters[key1]; !ok || val == nil {
		if key2 != "" {
			if val, ok = filters[key2]; !ok || val == nil {
				return 0, fmt.Errorf("key not found")
			}
		} else {
			return 0, fmt.Errorf("key not found")
		}
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if v == "" {
			return 0, nil
		}
		return strconv.ParseFloat(v, 64)
	}
	return 0, fmt.Errorf("unknown type")
}
