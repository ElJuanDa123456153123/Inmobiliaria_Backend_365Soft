package services

import (
	"strings"
	"testing"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/stretchr/testify/assert"
)

// U7 - Helpers puros del servicio de integración bancaria (Banco MC4).
// Se prueban las funciones que no requieren red: validación de códigos
// de respuesta del banco, cálculo del vencimiento del QR, generación de
// alias únicos y formateo de fechas para la API.

func newBancoServiceForTest() *BancoQrService {
	return NewBancoQrService(&config.Config{})
}

// TestValidarRespuestaBanco - Códigos exitosos vs. erróneos del banco
func TestValidarRespuestaBanco_TableDriven(t *testing.T) {
	cases := []struct {
		name   string
		codigo string
		want   bool
	}{
		{"codigo_exitoso_0000", "0000", true},
		{"codigo_exitoso_OK", "OK", true},
		{"codigo_error_1212_alias_invalido", "1212", false},
		{"codigo_error_9999", "9999", false},
		{"codigo_vacio", "", false},
		{"codigo_ok_minuscula_invalido", "ok", false},
		{"codigo_con_espacios", " 0000", false},
	}
	s := newBancoServiceForTest()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, s.ValidarRespuestaBanco(tc.codigo))
		})
	}
}

// TestCalcularFechaVencimiento - Vencimiento del QR según minutos configurados
func TestCalcularFechaVencimiento(t *testing.T) {
	s := newBancoServiceForTest()
	now := time.Now()

	t.Run("qr_estandar_15_minutos", func(t *testing.T) {
		venc := s.CalcularFechaVencimiento(15)
		assert.True(t, venc.After(now.Add(14*time.Minute)), "debe ser > 14 min")
		assert.True(t, venc.Before(now.Add(16*time.Minute)), "debe ser < 16 min")
	})

	t.Run("vencimiento_cero_es_ahora", func(t *testing.T) {
		venc := s.CalcularFechaVencimiento(0)
		assert.WithinDuration(t, now, venc, time.Minute)
	})

	t.Run("minutos_negativos_en_el_pasado", func(t *testing.T) {
		venc := s.CalcularFechaVencimiento(-5)
		assert.True(t, venc.Before(now), "debe estar en el pasado")
	})
}

// TestGenerarAliasUnico - Formato y unicidad del alias QR365I...
func TestGenerarAliasUnico(t *testing.T) {
	s := newBancoServiceForTest()

	t.Run("prefijo_estandar", func(t *testing.T) {
		alias := s.GenerarAliasUnico(nil)
		assert.True(t, strings.HasPrefix(alias, "QR365I"), "debe iniciar con QR365I: %s", alias)
	})

	t.Run("solo_digitos_tras_el_prefijo", func(t *testing.T) {
		alias := s.GenerarAliasUnico(nil)
		resto := strings.TrimPrefix(alias, "QR365I")
		for _, c := range resto {
			assert.True(t, c >= '0' && c <= '9', "solo dígitos esperados: %s", alias)
		}
	})

	t.Run("alias_con_property_id_nil_y_definido", func(t *testing.T) {
		id := uint(42)
		conID := s.GenerarAliasUnico(&id)
		sinID := s.GenerarAliasUnico(nil)
		assert.NotEqual(t, conID, sinID)
		assert.Contains(t, conID, "42")
		assert.True(t, strings.HasPrefix(conID, "QR365I"))
	})

	t.Run("unicidad_en_lote", func(t *testing.T) {
		vistos := make(map[string]bool, 500)
		for i := 0; i < 500; i++ {
			alias := s.GenerarAliasUnico(nil)
			assert.False(t, vistos[alias], "alias duplicado: %s", alias)
			vistos[alias] = true
		}
	})
}

// TestFormatearFecha - Formato dd/mm/yyyy exigido por la API del banco
func TestFormatearFecha(t *testing.T) {
	s := newBancoServiceForTest()

	t.Run("fecha_doble_digito", func(t *testing.T) {
		assert.Equal(t, "30/08/2026", s.formatearFecha(time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)))
	})

	t.Run("rellena_ceros", func(t *testing.T) {
		assert.Equal(t, "05/01/2026", s.formatearFecha(time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)))
	})
}
