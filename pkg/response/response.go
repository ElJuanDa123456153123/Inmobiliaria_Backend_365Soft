package response

import (
	"fmt"

	apperrors "github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// Success - Respuesta exitosa
func Success(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// SuccessWithMessage - Respuesta exitosa con mensaje
func SuccessWithMessage(data interface{}, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
		"message": message,
	}
}

// Error - Respuesta de error
func Error(err error) map[string]interface{} {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    string(appErr.Code),
				"message": appErr.Message,
				"details": appErr.Details,
			},
		}
	}

	// Error genérico
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeInternal),
			"message": "Error interno del servidor",
		},
	}
}

// ErrorWithMessage - Error con mensaje customizado
func ErrorWithMessage(code apperrors.ErrorCode, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(code),
			"message": message,
		},
	}
}

// ValidationError - Error de validación
func ValidationError(field, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeValidation),
			"message": fmt.Sprintf("%s: %s", field, message),
		},
	}
}

// ValidationErrors - Error de validación con detalles
func ValidationErrors(details []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeValidation),
			"message": "Error de validación en los datos enviados",
			"details": details,
		},
	}
}

// NotFound - Recurso no encontrado
func NotFound(resource string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeNotFound),
			"message": fmt.Sprintf("%s no encontrado", resource),
		},
	}
}

// Unauthorized - No autorizado
func Unauthorized() map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeUnauthorized),
			"message": "No autorizado para realizar esta acción",
		},
	}
}

// Forbidden - Prohibido
func Forbidden() map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    string(apperrors.ErrCodeForbidden),
			"message": "No tienes permisos para realizar esta acción",
		},
	}
}

// FromError - Convierte error a respuesta JSON
func FromError(err error) map[string]interface{} {
	return Error(err)
}

// Paginated - Respuesta con paginación
func Paginated(data interface{}, page, limit, total int64) map[string]interface{} {
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return Success(map[string]interface{}{
		"data": data,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}

// Message - Respuesta con solo mensaje
func Message(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": message,
	}
}
