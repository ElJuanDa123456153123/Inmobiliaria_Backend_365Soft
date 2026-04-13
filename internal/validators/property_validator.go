package validators

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/dto"
	"github.com/go-playground/validator/v10"
)

// Validator - Wrapper de validator.v10
type Validator struct {
	validate *validator.Validate
}

// NewValidator - Crea una nueva instancia del validador
func NewValidator() *Validator {
	v := validator.New()

	// Registrar validadores custom
	v.RegisterValidation("precio_minimo", validatePrecioMinimo)
	v.RegisterValidation("coordenadas_bolivia", validateCoordenadasBolivia)

	// Registrar funciones de mapeo de nombres de campo
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{
		validate: v,
	}
}

// Validate - Valida una struct y retorna errores formateados
func (v *Validator) Validate(s interface{}) map[string]string {
	if err := v.validate.Struct(s); err != nil {
		return v.formatErrors(err)
	}
	return nil
}

// ValidateWithError - Valida y retorna error detallado
func (v *Validator) ValidateWithError(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		return fmt.Errorf("validación fallida: %w", err)
	}
	return nil
}

// formatErrors - Formatea errores de validación
func (v *Validator) formatErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			errors[field] = v.getErrorMessage(e)
		}
	}

	return errors
}

// getErrorMessage - Retorna mensaje de error amigable
func (v *Validator) getErrorMessage(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s es requerido", field)
	case "min":
		return fmt.Sprintf("%s debe tener mínimo %s caracteres", field, param)
	case "max":
		return fmt.Sprintf("%s debe tener máximo %s caracteres", field, param)
	case "email":
		return fmt.Sprintf("%s debe ser un email válido", field)
	case "oneof":
		return fmt.Sprintf("%s debe ser uno de: %s", field, param)
	case "gt":
		return fmt.Sprintf("%s debe ser mayor que %s", field, param)
	case "gte":
		return fmt.Sprintf("%s debe ser mayor o igual a %s", field, param)
	case "lt":
		return fmt.Sprintf("%s debe ser menor que %s", field, param)
	case "lte":
		return fmt.Sprintf("%s debe ser menor o igual a %s", field, param)
	case "numeric":
		return fmt.Sprintf("%s debe ser numérico", field)
	case "url":
		return fmt.Sprintf("%s debe ser una URL válida", field)
	default:
		return fmt.Sprintf("%s es inválido", field)
	}
}

// ========== VALIDADORES CUSTOM ==========

// validatePrecioMinimo - Valida que el precio sea mínimo según operación
func validatePrecioMinimo(fl validator.FieldLevel) bool {
	precio := fl.Field().Float()

	// Validar que sea mayor a 0
	if precio <= 0 {
		return false
	}

	// Validar rangos razonables (en USD)
	if precio < 100 {
		return false // Mínimo $100 USD
	}

	if precio > 100000000 {
		return false // Máximo $100M USD (previene errores)
	}

	return true
}

// validateCoordenadasBolivia - Valida que coordenadas estén dentro de Bolivia
func validateCoordenadasBolivia(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()

	// Bolivia está aproximadamente entre:
	// Latitud: -9° a -23°
	// Longitud: -57° a -69°

	if lat < -23 || lat > -9 {
		return false
	}

	return true
}

// ========== VALIDADORES ESPECÍFICOS DE PROPIEDAD ==========

// PropertyValidator - Validador especializado para propiedades
type PropertyValidator struct {
	validator *Validator
}

// NewPropertyValidator - Crea un nuevo validador de propiedades
func NewPropertyValidator() *PropertyValidator {
	return &PropertyValidator{
		validator: NewValidator(),
	}
}

// ValidateCreateProperty - Valida creación de propiedad
func (pv *PropertyValidator) ValidateCreateProperty(req *dto.CreatePropertyRequest) map[string]string {
	errors := make(map[string]string)

	// Validar struct básico
	if errs := pv.validator.Validate(req); errs != nil {
		for k, v := range errs {
			errors[k] = v
		}
	}

	// Validaciones de negocio

	// Si tiene número de habitaciones, debe tener al menos 1
	if req.NumeroHabitaciones != nil && *req.NumeroHabitaciones < 1 {
		errors["numero_habitaciones"] = "Si especifica habitaciones, debe ser al menos 1"
	}

	// Si tiene metraje construido, debe ser mayor a 0
	if req.MetrajeConstruido != nil && *req.MetrajeConstruido <= 0 {
		errors["metraje_construido"] = "El metraje construido debe ser mayor a 0"
	}

	// Validar que si tiene piscina, jardín o terraza, tenga metraje construido
	if (req.TienePiscina || req.TieneJardin || req.TieneTerraza) && req.MetrajeConstruido == nil {
		errors["metraje_construido"] = "Debe especificar metraje construido si tiene amenities"
	}

	// Validar longitud si especifica latitud
	if req.Latitud != nil && req.Longitud == nil {
		errors["longitud"] = "Debe especificar longitud junto con latitud"
	}

	// Validar latitud si especifica longitud
	if req.Longitud != nil && req.Latitud == nil {
		errors["latitud"] = "Debe especificar latitud junto con longitud"
	}

	return errors
}

// ValidateUpdateProperty - Valida actualización de propiedad
func (pv *PropertyValidator) ValidateUpdateProperty(req *dto.UpdatePropertyRequest) map[string]string {
	errors := make(map[string]string)

	// Validaciones específicas para update

	// Validar que al menos un campo esté presente
	if req.Titulo == nil && req.Descripcion == nil && req.Precio == nil {
		errors["general"] = "Debe especificar al menos un campo para actualizar"
	}

	// Validar lógica de estado
	if req.Estado != nil {
		estado := *req.Estado
		if estado == "vendida" || estado == "alquilada" {
			// Podríamos agregar validaciones adicionales aquí
		}
	}

	return errors
}

// ValidatePropertyFilter - Valida filtros de búsqueda
func (pv *PropertyValidator) ValidatePropertyFilter(filter *dto.PropertyFilter) map[string]string {
	errors := make(map[string]string)

	// Validar rangos lógicos
	if filter.PrecioMin != nil && filter.PrecioMax != nil {
		if *filter.PrecioMin > *filter.PrecioMax {
			errors["precio"] = "El precio mínimo no puede ser mayor al máximo"
		}
	}

	if filter.MetrajeMin != nil && filter.MetrajeMax != nil {
		if *filter.MetrajeMin > *filter.MetrajeMax {
			errors["metraje"] = "El metraje mínimo no puede ser mayor al máximo"
		}
	}

	if filter.HabitacionesMin != nil && filter.HabitacionesMax != nil {
		if *filter.HabitacionesMin > *filter.HabitacionesMax {
			errors["habitaciones"] = "Las habitaciones mínimas no pueden ser mayores a las máximas"
		}
	}

	if filter.BanosMin != nil && filter.BanosMax != nil {
		if *filter.BanosMin > *filter.BanosMax {
			errors["banos"] = "Los baños mínimos no pueden ser mayores a los máximos"
		}
	}

	// Validar ordenamiento
	validOrderBys := map[string]bool{
		"precio_asc":    true,
		"precio_desc":   true,
		"mas_recientes": true,
		"mas_visitados": true,
		"relevancia":    true,
	}

	if filter.OrderBy != "" && !validOrderBys[filter.OrderBy] {
		errors["order_by"] = "Ordenamiento inválido. Use: precio_asc, precio_desc, mas_recientes, mas_visitados o relevancia"
	}

	return errors
}

// ValidateUploadPhotos - Valida upload de fotos
func (pv *PropertyValidator) ValidateUploadPhotos(req *dto.UploadPhotosRequest) map[string]string {
	errors := make(map[string]string)

	// Validar cantidad de fotos
	if len(req.Files) == 0 {
		errors["files"] = "Debe especificar al menos una foto"
	}

	if len(req.Files) > 7 {
		errors["files"] = "Máximo 7 fotos permitidas"
	}

	// Validar cada archivo
	for i, file := range req.Files {
		fieldName := fmt.Sprintf("files[%d]", i)

		// Validar tamaño
		if file.Size == 0 {
			errors[fieldName] = "El archivo está vacío"
		}

		if file.Size > 5*1024*1024 { // 5MB
			errors[fieldName] = "El archivo excede el tamaño máximo de 5MB"
		}

		// Validar tipo MIME
		validMimeTypes := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/webp": true,
		}

		if !validMimeTypes[file.MimeType] {
			errors[fieldName] = "Tipo de archivo inválido. Solo JPEG, PNG y WEBP"
		}
	}

	// Validar warning si hay menos de 3 fotos
	if len(req.Files) > 0 && len(req.Files) < 3 {
		errors["warning"] = "Recomendamos agregar al menos 3 fotos para mejor visibilidad"
	}

	// Validar que el orden corresponda con la cantidad de archivos
	if len(req.Orden) > 0 && len(req.Orden) != len(req.Files) {
		errors["orden"] = "El array de orden debe tener la misma longitud que archivos"
	}

	return errors
}

// ========== FUNCIONES DE UTILIDAD ==========

// HasErrors - Verifica si hay errores
func HasErrors(errors map[string]string) bool {
	return len(errors) > 0
}

// GetFirstError - Retorna el primer error
func GetFirstError(errors map[string]string) string {
	for _, err := range errors {
		return err
	}
	return ""
}

// FormatErrorsForResponse - Formatea errores para response JSON
func FormatErrorsForResponse(errors map[string]string) []interface{} {
	details := make([]interface{}, 0, len(errors))

	for field, message := range errors {
		details = append(details, map[string]string{
			"field":   field,
			"message": message,
		})
	}

	return details
}
