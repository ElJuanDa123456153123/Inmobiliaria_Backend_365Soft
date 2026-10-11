package handlers

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/repository"
)

// NormalizeListPropertiesRequest - Normaliza paginación, ordenamiento y estado de la
// búsqueda (función pura: dado un request, siempre aplica los mismos valores por defecto).
func NormalizeListPropertiesRequest(req *ListPropertiesRequest) {
	// Validar página
	if req.Page < 1 {
		req.Page = 1
	}

	// Validar límite (máximo 100, mínimo 1)
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Validar ordenamiento
	validOrderBy := map[string]bool{
		"precio": true, "fecha": true, "vistas": true, "metros": true,
	}
	if !validOrderBy[req.OrderBy] {
		req.OrderBy = "fecha"
	}

	// Validar dirección
	if req.Order != "asc" && req.Order != "desc" {
		req.Order = "desc"
	}

	// Validar estado
	validEstados := map[string]bool{
		"activo": true, "inactivo": true, "vendido": true,
	}
	if req.Estado != "" && !validEstados[req.Estado] {
		req.Estado = "activo"
	}
}

// BuildListPropertyParams - Construye los parámetros de búsqueda del repositorio a
// partir del request ya normalizado (función pura). Calcula el offset de paginación.
func BuildListPropertyParams(req ListPropertiesRequest) repository.ListPropertyParams {
	return repository.ListPropertyParams{
		Estado:         req.Estado,
		TipoOperacion:  req.TipoOperacion,
		TipoPropiedad:  req.TipoPropiedad,
		Zona:           req.Zona,
		Moneda:         req.Moneda,
		PrecioMin:      req.PrecioMin,
		PrecioMax:      req.PrecioMax,
		MetrosMin:      req.MetrosMin,
		MetrosMax:      req.MetrosMax,
		Dormitorios:    req.Dormitorios,
		Banos:          req.Banos,
		Garaje:         req.Garaje,
		AceptaPermuta:  req.AceptaPermuta,
		Financia:       req.Financia,
		Condominio:     req.Condominio,
		AptoOficina:    req.AptoOficina,
		ViviendaSocial: req.ViviendaSocial,
		Destacado:      req.Destacado,
		SearchText:     req.SearchText,
		OrderBy:        req.OrderBy,
		Order:          req.Order,
		Limit:          req.Limit,
		Offset:         (req.Page - 1) * req.Limit,
	}
}
