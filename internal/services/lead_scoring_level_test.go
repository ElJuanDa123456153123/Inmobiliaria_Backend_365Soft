package services

import "testing"

// Pruebas unitarias del umbral de niveles del lead scoring
// (internal/services/lead_scoring_service.go, función calculateLevel).
// Los criterios completos del score (presupuesto, zona, interacciones, actividad real,
// origen y decaimiento) dependen de persistencia y se validan mediante la suite de
// integración (ver matriz de aceptación I4-12 e I4-13).

func TestNivelesLeadScoring(t *testing.T) {
	s := &LeadScoringService{}
	casos := []struct {
		score    int
		esperado string
	}{
		{0, "BAJO"},
		{29, "BAJO"},
		{30, "MEDIO"},
		{59, "MEDIO"},
		{60, "ALTO"},
		{100, "ALTO"},
	}
	for _, c := range casos {
		if got := s.calculateLevel(c.score); got != c.esperado {
			t.Errorf("calculateLevel(%d) = %q, esperado %q", c.score, got, c.esperado)
		}
	}
}
