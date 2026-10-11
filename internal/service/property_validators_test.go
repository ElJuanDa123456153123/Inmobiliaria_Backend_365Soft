package service

import (
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func ptr(f float64) *float64 { return &f }

// ---------- ValidarRangoPrecio ----------

// TestValidarRangoPrecio_PreciosValidos - Valores dentro de rango para cada moneda/operación
func TestValidarRangoPrecio_PreciosValidos(t *testing.T) {
	cases := []struct {
		name          string
		precio        float64
		moneda        string
		tipoOperacion string
	}{
		// USD venta: 100 - 100.000.000
		{"venta_usd_minimo", 100, "USD", "venta"},
		{"venta_usd_maximo", 100000000, "USD", "venta"},
		{"venta_usd_intermedio", 150000, "USD", "venta"},
		// USD alquiler: 50 - 50.000
		{"alquiler_usd_minimo", 50, "USD", "alquiler"},
		{"alquiler_usd_maximo", 50000, "USD", "alquiler"},
		{"alquiler_usd_intermedio", 800, "USD", "alquiler"},
		// USD anticrético: 1.000 - 1.000.000
		{"anticretico_usd_minimo", 1000, "USD", "anticretico"},
		{"anticretico_usd_maximo", 1000000, "USD", "anticretico"},
		// BOB venta: 1.000 - 500.000.000
		{"venta_bob_minimo", 1000, "BOB", "venta"},
		{"venta_bob_maximo", 500000000, "BOB", "venta"},
		{"venta_bob_intermedio", 850000, "BOB", "venta"},
		// BOB alquiler: 200 - 200.000
		{"alquiler_bob_minimo", 200, "BOB", "alquiler"},
		{"alquiler_bob_maximo", 200000, "BOB", "alquiler"},
		{"alquiler_bob_intermedio", 3500, "BOB", "alquiler"},
		// BOB anticrético: 5.000 - 5.000.000
		{"anticretico_bob_minimo", 5000, "BOB", "anticretico"},
		{"anticretico_bob_maximo", 5000000, "BOB", "anticretico"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarRangoPrecio(tc.precio, tc.moneda, tc.tipoOperacion)
			assert.NoError(t, err)
		})
	}
}

// TestValidarRangoPrecio_PreciosInvalidos - Valores fuera de rango deben retornar error de validación
func TestValidarRangoPrecio_PreciosInvalidos(t *testing.T) {
	cases := []struct {
		name          string
		precio        float64
		moneda        string
		tipoOperacion string
	}{
		{"venta_usd_bajo_el_minimo", 99.99, "USD", "venta"},
		{"venta_usd_sobre_el_maximo", 100000001, "USD", "venta"},
		{"venta_usd_precio_cero", 0, "USD", "venta"},
		{"venta_usd_precio_negativo", -500, "USD", "venta"},
		{"alquiler_usd_bajo_el_minimo", 49, "USD", "alquiler"},
		{"alquiler_usd_sobre_el_maximo", 50001, "USD", "alquiler"},
		{"anticretico_usd_bajo_el_minimo", 999, "USD", "anticretico"},
		{"anticretico_usd_sobre_el_maximo", 1000001, "USD", "anticretico"},
		{"venta_bob_bajo_el_minimo", 999, "BOB", "venta"},
		{"venta_bob_sobre_el_maximo", 500000001, "BOB", "venta"},
		{"alquiler_bob_bajo_el_minimo", 199, "BOB", "alquiler"},
		{"alquiler_bob_sobre_el_maximo", 200001, "BOB", "alquiler"},
		{"anticretico_bob_bajo_el_minimo", 4999, "BOB", "anticretico"},
		{"anticretico_bob_sobre_el_maximo", 5000001, "BOB", "anticretico"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarRangoPrecio(tc.precio, tc.moneda, tc.tipoOperacion)
			assert.Error(t, err)

			appErr, ok := err.(*errors.AppError)
			assert.True(t, ok, "debe ser un AppError")
			assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
			assert.Contains(t, appErr.Message, "precio")
		})
	}
}

// TestValidarRangoPrecio_MonedaInvalida - Moneda no soportada no produce error de rango (se valida en otro campo)
func TestValidarRangoPrecio_MonedaDesconocida(t *testing.T) {
	err := ValidarRangoPrecio(1000, "EUR", "venta")
	assert.NoError(t, err, "ValidarRangoPrecio solo valida rangos de USD y BOB; otra moneda se rechaza antes (validación de campo moneda)")
}

// ---------- ValidarCoordenadasBolivia ----------

// TestValidarCoordenadasBolivia_CoordenadasValidas - Ciudades y límites exactos del territorio boliviano
func TestValidarCoordenadasBolivia_CoordenadasValidas(t *testing.T) {
	cases := []struct {
		name     string
		latitud  float64
		longitud float64
	}{
		{"La_Paz", -16.4897, -68.1193},
		{"Santa_Cruz_de_la_Sierra", -17.7833, -63.1821},
		{"Cochabamba", -17.3895, -66.1568},
		{"Oruro", -17.9833, -67.15},
		{"Potosi", -19.5833, -65.75},
		{"Tarija", -21.5333, -64.7333},
		{"Trinidad_Beni", -14.8333, -64.9},
		{"Cobija_Pando", -11.0, -68.7333},
		{"limite_latitud_norte_exacta", -9, -63},
		{"limite_latitud_sur_exacta", -23, -63},
		{"limite_longitud_oeste_exacta", -17, -69},
		{"limite_longitud_este_exacta", -17, -57},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarCoordenadasBolivia(ptr(tc.latitud), ptr(tc.longitud))
			assert.NoError(t, err)
		})
	}
}

// TestValidarCoordenadasBolivia_CoordenadasInvalidas - Coordenadas fuera de Bolivia deben fallar
func TestValidarCoordenadasBolivia_CoordenadasInvalidas(t *testing.T) {
	cases := []struct {
		name          string
		latitud       float64
		longitud      float64
		campoEsperado string
	}{
		{"latitud_sobre_el_norte_EEUU", -5, -63, "latitud"},
		{"latitud_bajo_el_sur_Argentina", -25, -63, "latitud"},
		{"latitud_ecuador_positiva", 10, -63, "latitud"},
		{"longitud_sobre_el_este_Brasil", -17, -50, "longitud"},
		{"longitud_bajo_el_oeste_Peru", -17, -75, "longitud"},
		{"longitud_hemisferio_este", -17, 20, "longitud"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidarCoordenadasBolivia(ptr(tc.latitud), ptr(tc.longitud))
			assert.Error(t, err)

			appErr, ok := err.(*errors.AppError)
			assert.True(t, ok, "debe ser un AppError")
			assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
			assert.Contains(t, appErr.Message, tc.campoEsperado)
		})
	}
}

// TestValidarCoordenadasBolivia_Opcionales - Coordenadas nil son válidas (el marcador es opcional)
func TestValidarCoordenadasBolivia_Opcionales(t *testing.T) {
	assert.NoError(t, ValidarCoordenadasBolivia(nil, nil))
	assert.NoError(t, ValidarCoordenadasBolivia(ptr(-17.78), nil))
	assert.NoError(t, ValidarCoordenadasBolivia(nil, ptr(-63.18)))
}
