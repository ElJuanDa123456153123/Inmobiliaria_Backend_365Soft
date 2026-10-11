package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeListPropertiesRequest_ValoresPorDefecto - Entradas inválidas se normalizan a los defaults
func TestNormalizeListPropertiesRequest_ValoresPorDefecto(t *testing.T) {
	req := ListPropertiesRequest{
		Page:    0,     // inválido → 1
		Limit:   500,   // inválido (>100) → 20
		OrderBy: "foo", // inválido → fecha
		Order:   "ascendente", // inválido → desc
		Estado:  "pendiente",  // inválido → activo
	}

	NormalizeListPropertiesRequest(&req)

	assert.Equal(t, 1, req.Page)
	assert.Equal(t, 20, req.Limit)
	assert.Equal(t, "fecha", req.OrderBy)
	assert.Equal(t, "desc", req.Order)
	assert.Equal(t, "activo", req.Estado)
}

// TestNormalizeListPropertiesRequest_LimitesInferiores - limit < 1 también se normaliza
func TestNormalizeListPropertiesRequest_LimitesInferiores(t *testing.T) {
	req := ListPropertiesRequest{Page: 3, Limit: 0, OrderBy: "precio", Order: "asc", Estado: "vendido"}

	NormalizeListPropertiesRequest(&req)

	assert.Equal(t, 3, req.Page, "una página válida no debe modificarse")
	assert.Equal(t, 20, req.Limit)
	assert.Equal(t, "precio", req.OrderBy, "un ordenamiento válido no debe modificarse")
	assert.Equal(t, "asc", req.Order, "una dirección válida no debe modificarse")
	assert.Equal(t, "vendido", req.Estado, "un estado válido no debe modificarse")
}

// TestNormalizeListPropertiesRequest_LimiteMaximoValido - limit == 100 es válido y no se modifica
func TestNormalizeListPropertiesRequest_LimiteMaximoValido(t *testing.T) {
	req := ListPropertiesRequest{Page: 2, Limit: 100, OrderBy: "vistas", Order: "desc", Estado: "inactivo"}

	NormalizeListPropertiesRequest(&req)

	assert.Equal(t, 100, req.Limit)
	assert.Equal(t, "vistas", req.OrderBy)
}

// TestBuildListPropertyParams_FiltrosCombinados - Todos los filtros combinados se trasladan a los params del repositorio
func TestBuildListPropertyParams_FiltrosCombinados(t *testing.T) {
	permuta := true
	financia := false

	req := ListPropertiesRequest{
		TipoOperacion:  "venta",
		TipoPropiedad:  "casa",
		Zona:           "Calle 5, Zona Norte",
		Estado:         "activo",
		Moneda:         "USD",
		PrecioMin:      50000,
		PrecioMax:      150000,
		MetrosMin:      80,
		MetrosMax:      200,
		Dormitorios:    "3",
		Banos:          "2",
		Garaje:         "1",
		AceptaPermuta:  &permuta,
		Financia:       &financia,
		Condominio:     &financia,
		AptoOficina:    &financia,
		ViviendaSocial: &financia,
		SearchText:     "amoblado",
		OrderBy:        "precio",
		Order:          "asc",
		Page:           3,
		Limit:          10,
	}

	params := BuildListPropertyParams(req)

	assert.Equal(t, "venta", params.TipoOperacion)
	assert.Equal(t, "casa", params.TipoPropiedad)
	assert.Equal(t, "Calle 5, Zona Norte", params.Zona)
	assert.Equal(t, "activo", params.Estado)
	assert.Equal(t, "USD", params.Moneda)
	assert.Equal(t, 50000.0, params.PrecioMin)
	assert.Equal(t, 150000.0, params.PrecioMax)
	assert.Equal(t, 80.0, params.MetrosMin)
	assert.Equal(t, 200.0, params.MetrosMax)
	assert.Equal(t, "3", params.Dormitorios)
	assert.Equal(t, "2", params.Banos)
	assert.Equal(t, "1", params.Garaje)
	assert.NotNil(t, params.AceptaPermuta)
	assert.True(t, *params.AceptaPermuta)
	assert.NotNil(t, params.Financia)
	assert.False(t, *params.Financia)
	assert.Equal(t, "amoblado", params.SearchText)
	assert.Equal(t, "precio", params.OrderBy)
	assert.Equal(t, "asc", params.Order)
	assert.Equal(t, 10, params.Limit)
}

// TestBuildListPropertyParams_Offset - El offset se calcula como (page-1) * limit
func TestBuildListSearchParams_Offset(t *testing.T) {
	cases := []struct {
		name           string
		page           int
		limit          int
		expectedOffset int
	}{
		{"pagina_1_comienza_en_cero", 1, 20, 0},
		{"pagina_2", 2, 20, 20},
		{"pagina_3_limit_10", 3, 10, 20},
		{"pagina_5_limit_50", 5, 50, 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := BuildListPropertyParams(ListPropertiesRequest{Page: tc.page, Limit: tc.limit})
			assert.Equal(t, tc.expectedOffset, params.Offset)
		})
	}
}

// TestBuildListPropertyParams_RequestVacio - Un request sin filtros genera params neutros
func TestBuildListPropertyParams_RequestVacio(t *testing.T) {
	params := BuildListPropertyParams(ListPropertiesRequest{Page: 1, Limit: 20})

	assert.Empty(t, params.TipoOperacion)
	assert.Empty(t, params.TipoPropiedad)
	assert.Empty(t, params.Zona)
	assert.Empty(t, params.Moneda)
	assert.Equal(t, 0.0, params.PrecioMin)
	assert.Equal(t, 0.0, params.PrecioMax)
	assert.Nil(t, params.AceptaPermuta)
	assert.Nil(t, params.Financia)
	assert.Empty(t, params.SearchText)
	assert.Equal(t, 0, params.Offset)
}
