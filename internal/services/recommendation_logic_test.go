package services

import (
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// Pruebas unitarias del motor de recomendaciones híbrido (internal/services/recommendation_service.go).
// Cubren la construcción del perfil ponderado (favoritos 2.0, vistas 1.0, búsquedas 1.5),
// el score de contenido por factor (zona 30%, tipo 15%, operación 15%, precio 20%,
// dormitorios 10%, baños 10%) y el ordenamiento por score.

func propiedadEjemplo(id uint, zona, tipo, operacion, dorm, banos string, precio float64) models.Property {
	return models.Property{
		ID:            id,
		Zona:          zona,
		TipoPropiedad: models.PropertyType(tipo),
		TipoOperacion: models.OperationType(operacion),
		Dormitorios:   dorm,
		Banos:         banos,
		Precio:        precio,
	}
}

func TestPesosDeSenalDelPerfil(t *testing.T) {
	if pesoFavorito != 2.0 || pesoVista != 1.0 || pesoBusqueda != 1.5 {
		t.Errorf("pesos de señal incorrectos: favorito=%v vista=%v búsqueda=%v", pesoFavorito, pesoVista, pesoBusqueda)
	}
	if pesoContenido != 0.6 || pesoColaborativo != 0.4 {
		t.Errorf("pesos de combinación incorrectos: contenido=%v colaborativo=%v", pesoContenido, pesoColaborativo)
	}
}

func TestPerfilAcumulaPesosPorInteraccion(t *testing.T) {
	p := NewUserProfile(1)
	p.addPropertyPreferences(propiedadEjemplo(10, "Sacaba", "departamento", "alquiler", "2", "1", 1500), pesoFavorito)
	p.addPropertyPreferences(propiedadEjemplo(11, "Sacaba", "departamento", "alquiler", "2", "1", 1500), pesoVista)

	if p.ZoneWeights["Sacaba"] != pesoFavorito+pesoVista {
		t.Errorf("zona Sacaba = %v, esperado %v (favorito+vista)", p.ZoneWeights["Sacaba"], pesoFavorito+pesoVista)
	}
	if p.PropertyTypeWeights[models.PropertyType("departamento")] != 3.0 {
		t.Errorf("peso tipo departamento = %v, esperado 3.0", p.PropertyTypeWeights[models.PropertyType("departamento")])
	}
	// buildUserProfile calcula el promedio ponderado al final del perfilado:
	// (1500*2 + 1500*1) / (2+1) = 1500
	if p.PriceWeightedSum != 4500 || p.PriceWeightTotal != 3 {
		t.Errorf("sumas ponderadas = %v/%v, esperado 4500/3", p.PriceWeightedSum, p.PriceWeightTotal)
	}
}

func TestPerfilAcumulaBusquedasGuardadas(t *testing.T) {
	p := NewUserProfile(1)
	p.addSearchPreferences([]byte(`{"zona":"Calacoto","tipo_propiedad":"casa","tipo_operacion":"venta"}`), pesoBusqueda)

	if p.ZoneWeights["Calacoto"] != pesoBusqueda {
		t.Errorf("peso de zona por búsqueda guardada = %v, esperado %v", p.ZoneWeights["Calacoto"], pesoBusqueda)
	}
	if p.PropertyTypeWeights[models.PropertyType("casa")] != 1.5 {
		t.Errorf("peso de tipo por búsqueda guardada = %v, esperado 1.5", p.PropertyTypeWeights[models.PropertyType("casa")])
	}
}

func TestPerfilIgnoraCamposSinDefinir(t *testing.T) {
	p := NewUserProfile(1)
	p.addPropertyPreferences(propiedadEjemplo(10, "", "", "", "sin-definir", "sin-definir", 0), pesoFavorito)

	if len(p.ZoneWeights) != 0 || len(p.DormitoriosWeights) != 0 || len(p.BanosWeights) != 0 {
		t.Error("campos vacíos o 'sin-definir' no deben acumular peso")
	}
	if p.PriceWeightTotal != 0 {
		t.Error("precio 0 no debe acumular peso")
	}
}

func servicioSinBD() *RecommendationService {
	return &RecommendationService{}
}

func TestScoreContenidoCoincidenciaTotal(t *testing.T) {
	s := servicioSinBD()
	p := NewUserProfile(1)
	p.addPropertyPreferences(propiedadEjemplo(10, "Sacaba", "departamento", "alquiler", "2", "1", 1500), pesoFavorito)
	// buildUserProfile deriva el precio promedio ponderado del historial
	p.AveragePriceViewed = p.PriceWeightedSum / p.PriceWeightTotal

	score, reason, factors := s.calculatePropertyScore(
		propiedadEjemplo(20, "Sacaba", "departamento", "alquiler", "2", "1", 1500), p)

	// Zona 0.30 + Tipo 0.15 + Operación 0.15 + Precio 0.20 + Dormitorios 0.10 + Baños 0.10 = 1.0
	if score < 0.999 || score > 1.001 {
		t.Errorf("score de coincidencia total = %v, esperado 1.0", score)
	}
	if len(factors) != 6 {
		t.Errorf("se esperaban 6 factores, obtuvieron %d: %v", len(factors), factors)
	}
	if reason == "Propiedad recomendada basada en tu historial" {
		t.Error("con factores el reason debe listar los factores")
	}
}

func TestScoreContenidoParcialPorFactor(t *testing.T) {
	s := servicioSinBD()
	p := NewUserProfile(1)
	// Una sola vista (señal 1.0 de 2.0 máxima) en zona Sacaba
	p.addPropertyPreferences(propiedadEjemplo(10, "Sacaba", "departamento", "venta", "3", "2", 1000), pesoVista)

	score, _, factors := s.calculatePropertyScore(
		propiedadEjemplo(20, "Sacaba", "casa", "alquiler", "2", "1", 1500), p)

	// Zona: 0.30 * (1.0/2.0) = 0.15; los demás factores no coinciden
	if score < 0.149 || score > 0.151 {
		t.Errorf("score = %v, esperado 0.15", score)
	}
	if len(factors) != 1 || factors[0] != "Zona preferida" {
		t.Errorf("factores = %v, esperado solo [Zona preferida]", factors)
	}
}

func TestScoreContenidoPrecioFueraDeRango(t *testing.T) {
	s := servicioSinBD()
	p := NewUserProfile(1)
	p.addPropertyPreferences(propiedadEjemplo(10, "Sacaba", "departamento", "alquiler", "2", "1", 1000), pesoFavorito)
	p.AveragePriceViewed = 1000

	// Diferencia del 10%: dentro de +-30% => +0.20
	scoreCercana, _, _ := s.calculatePropertyScore(propiedadEjemplo(20, "Sacaba", "departamento", "alquiler", "2", "1", 1100), p)
	// Diferencia del 45%: dentro de +-50% => +0.10
	scoreMedia, _, _ := s.calculatePropertyScore(propiedadEjemplo(21, "Sacaba", "departamento", "alquiler", "2", "1", 1450), p)
	// Diferencia del 100%: fuera de rango => +0
	scoreLejana, _, _ := s.calculatePropertyScore(propiedadEjemplo(22, "Sacaba", "departamento", "alquiler", "2", "1", 2000), p)

	if scoreCercana-scoreLejana < 0.19 {
		t.Errorf("precio cercano debe aportar 0.20 más que lejana: %v vs %v", scoreCercana, scoreLejana)
	}
	if scoreMedia-scoreLejana < 0.09 {
		t.Errorf("precio a 45%% debe aportar 0.10 más que lejana: %v vs %v", scoreMedia, scoreLejana)
	}
}

func TestOrdenamientoPorScoreDescendente(t *testing.T) {
	recs := []PropertyRecommendation{
		{Score: 0.4}, {Score: 0.9}, {Score: 0.1}, {Score: 0.7},
	}
	sortRecommendations(recs)
	if recs[0].Score != 0.9 || recs[1].Score != 0.7 || recs[2].Score != 0.4 || recs[3].Score != 0.1 {
		t.Errorf("orden incorrecto tras sortRecommendations: %v", recs)
	}
}

func TestPerfilSinActividadNoGeneraSenal(t *testing.T) {
	p := NewUserProfile(1)
	s := servicioSinBD()
	score, reason, factors := s.calculatePropertyScore(propiedadEjemplo(20, "Sacaba", "casa", "venta", "3", "2", 1000), p)
	if score != 0 {
		t.Errorf("sin perfil el score debe ser 0, obtuvo %v", score)
	}
	if reason != "Propiedad recomendada basada en tu historial" {
		t.Error("sin factores el reason debe ser el genérico")
	}
	if len(factors) != 0 {
		t.Error("sin perfil no debe haber factores")
	}
}
