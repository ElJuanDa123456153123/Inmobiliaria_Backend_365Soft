package handlers

import (
	"testing"
	"time"
)

// Pruebas unitarias de las reglas puras del módulo de alquiler
// (internal/handlers/rental_request_handler.go: generarCuotas y calcularDeposito).

func TestGenerarCuotasContratoDoceMeses(t *testing.T) {
	inicio := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2027, 11, 1, 0, 0, 0, 0, time.UTC)

	cuotas, err := generarCuotas(inicio, fin, 1200)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cuotas) != 12 {
		t.Fatalf("contrato de 12 meses debe generar 12 cuotas, obtuvo %d", len(cuotas))
	}
	// Primera cuota: noviembre 2026; última: octubre 2027
	if cuotas[0].MesCorrespondiente != 11 || cuotas[0].Anio != 2026 {
		t.Errorf("primera cuota = %d/%d, esperado 11/2026", cuotas[0].MesCorrespondiente, cuotas[0].Anio)
	}
	ultima := cuotas[11]
	if ultima.MesCorrespondiente != 10 || ultima.Anio != 2027 {
		t.Errorf("última cuota = %d/%d, esperado 10/2027", ultima.MesCorrespondiente, ultima.Anio)
	}
	for _, c := range cuotas {
		if c.Tipo != "cuota" || c.Estado != "pendiente" || c.Monto != 1200 {
			t.Errorf("cuota %d/%d con datos incorrectos: tipo=%s estado=%s monto=%v",
				c.MesCorrespondiente, c.Anio, c.Tipo, c.Estado, c.Monto)
		}
	}
}

func TestGenerarCuotasTresMeses(t *testing.T) {
	inicio := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

	cuotas, err := generarCuotas(inicio, fin, 500)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cuotas) != 3 {
		t.Fatalf("rango de 3 meses debe generar 3 cuotas, obtuvo %d", len(cuotas))
	}
	meses := []int{cuotas[0].MesCorrespondiente, cuotas[1].MesCorrespondiente, cuotas[2].MesCorrespondiente}
	if meses[0] != 1 || meses[1] != 2 || meses[2] != 3 {
		t.Errorf("meses de cuotas = %v, esperado [1 2 3]", meses)
	}
	// La fecha de vencimiento conserva el día de inicio
	if cuotas[0].FechaPago.Day() != 15 {
		t.Errorf("la cuota debe vencer el día 15, obtuvo %d", cuotas[0].FechaPago.Day())
	}
}

func TestGenerarCuotasRangoVacio(t *testing.T) {
	f := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	if _, err := generarCuotas(f, f, 100); err == nil {
		t.Error("rango sin meses debe retornar error")
	}
	if _, err := generarCuotas(f, f.AddDate(0, 0, -1), 100); err == nil {
		t.Error("fecha_fin anterior a fecha_inicio debe retornar error")
	}
}

func TestGenerarCuotasTopeMaximo(t *testing.T) {
	inicio := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2040, 1, 1, 0, 0, 0, 0, time.UTC) // 30 años

	cuotas, err := generarCuotas(inicio, fin, 100)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cuotas) != 120 {
		t.Errorf("contrato de 30 años debe limitarse a 120 cuotas, obtuvo %d", len(cuotas))
	}
}

func TestCalcularDeposito(t *testing.T) {
	casos := []struct {
		texto        string
		montoMensual float64
		esperado     float64
	}{
		{"1 mes", 1200, 1200},
		{"2 meses", 1000, 2000},
		{"3 MESES ", 500, 1500},
		{"Depósito de 2 meses", 1000, 2000},
		{"15 meses", 100, 1200},    // tope de 12 meses
		{"medio mes", 1000, 0},     // sin número -> 0
		{"", 1000, 0},              // sin depósito declarado
		{"no", 1000, 0},            // explícito sin depósito
		{"sin deposito", 1000, 0},  // variante sin tilde
		{"sin depósito", 1000, 0},  // variante con tilde
	}
	for _, c := range casos {
		if got := calcularDeposito(c.texto, c.montoMensual); got != c.esperado {
			t.Errorf("calcularDeposito(%q, %v) = %v, esperado %v", c.texto, c.montoMensual, got, c.esperado)
		}
	}
}
