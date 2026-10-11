package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPropiedadesDisponibles_SinPlan(t *testing.T) {
	s := Subscription{}
	assert.Equal(t, 0, s.PropiedadesDisponibles())
}

func TestPropiedadesDisponibles_TableDriven(t *testing.T) {
	cases := []struct {
		name   string
		limite int
		usadas int
		want   int
	}{
		{"sin_usar", 5, 0, 5},
		{"parcialmente_usada", 5, 3, 2},
		{"una_restante", 5, 4, 1},
		{"limite_alcanzado", 5, 5, 0},
		{"limite_excedido", 5, 8, 0},
		{"plan_gratitulo_un_publicacion", 1, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Subscription{
				Plan:              &Plan{LimitePropiedades: tc.limite},
				PropiedadesUsadas: tc.usadas,
			}
			assert.Equal(t, tc.want, s.PropiedadesDisponibles())
		})
	}
}

func TestEstaVencida_TableDriven(t *testing.T) {
	ahora := time.Now()
	futuro := ahora.Add(24 * time.Hour)
	pasado := ahora.Add(-24 * time.Hour)

	cases := []struct {
		name string
		fin  *time.Time
		want bool
	}{
		{"indefinida_nil", nil, false},
		{"vigente", &futuro, false},
		{"vencida", &pasado, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Subscription{FechaFin: tc.fin}
			assert.Equal(t, tc.want, s.EstaVencida())
		})
	}
}

func TestPuedePublicar_TableDriven(t *testing.T) {
	ahora := time.Now()
	futuro := ahora.Add(24 * time.Hour)
	pasado := ahora.Add(-24 * time.Hour)
	plan := &Plan{LimitePropiedades: 3}

	cases := []struct {
		name   string
		estado string
		fin    *time.Time
		usadas int
		want   bool
	}{
		{"activa_con_cupo", "ACTIVA", &futuro, 1, true},
		{"activa_sin_cupo", "ACTIVA", &futuro, 3, false},
		{"activa_cupo_excedido", "ACTIVA", &futuro, 5, false},
		{"cancelada_con_cupo", "CANCELADA", &futuro, 0, false},
		{"vencida_con_cupo", "ACTIVA", &pasado, 0, false},
		{"indefinida_activa_con_cupo", "ACTIVA", nil, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Subscription{
				Plan:              plan,
				Estado:            tc.estado,
				FechaFin:          tc.fin,
				PropiedadesUsadas: tc.usadas,
			}
			assert.Equal(t, tc.want, s.PuedePublicar())
		})
	}
}
