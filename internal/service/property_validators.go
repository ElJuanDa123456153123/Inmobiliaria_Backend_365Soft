package service

import (
	"github.com/365Soft-Bolivia/inmobiliaria-backend/pkg/errors"
)

// ValidarRangoPrecio - Valida el precio de una propiedad según su moneda y tipo de
// operación (función pura). Rangos de negocio del sistema:
//   - Venta:        USD 100 - 100.000.000 | BOB 1.000 - 500.000.000
//   - Alquiler:     USD 50 - 50.000       | BOB 200 - 200.000
//   - Anticrético:  USD 1.000 - 1.000.000 | BOB 5.000 - 5.000.000
func ValidarRangoPrecio(precio float64, moneda, tipoOperacion string) error {
	if moneda == "USD" {
		if tipoOperacion == "alquiler" {
			// Alquiler en USD (precios mensuales): $50 - $50,000
			if precio < 50 || precio > 50000 {
				return errors.ValidationError("precio", "El precio de alquiler en USD debe estar entre $50 y $50,000 mensuales")
			}
		} else if tipoOperacion == "anticretico" {
			// Anticrético en USD: $1,000 - $1,000,000
			if precio < 1000 || precio > 1000000 {
				return errors.ValidationError("precio", "El monto de anticrético en USD debe estar entre $1,000 y $1,000,000")
			}
		} else {
			// Venta en USD: $100 - $100,000,000
			if precio < 100 || precio > 100000000 {
				return errors.ValidationError("precio", "El precio en USD debe estar entre $100 y $100,000,000")
			}
		}
	} else if moneda == "BOB" {
		if tipoOperacion == "alquiler" {
			// Alquiler en BOB (precios mensuales): 200 - 200,000 Bs
			if precio < 200 || precio > 200000 {
				return errors.ValidationError("precio", "El precio de alquiler en BOB debe estar entre 200 y 200,000 Bs mensuales")
			}
		} else if tipoOperacion == "anticretico" {
			// Anticrético en BOB: 5,000 - 5,000,000 Bs
			if precio < 5000 || precio > 5000000 {
				return errors.ValidationError("precio", "El monto de anticrético en BOB debe estar entre 5,000 y 5,000,000 Bs")
			}
		} else {
			// Venta en BOB: 1,000 - 500,000,000 Bs
			if precio < 1000 || precio > 500000000 {
				return errors.ValidationError("precio", "El precio en BOB debe estar entre 1,000 y 500,000,000 Bs")
			}
		}
	}
	return nil
}

// ValidarCoordenadasBolivia - Valida que las coordenadas geográficas estén dentro del
// territorio boliviano (función pura). Rango aproximado: latitud -9° a -23°,
// longitud -69° a -57°. Si alguna coordenada es nil no se valida (son opcionales).
func ValidarCoordenadasBolivia(latitud, longitud *float64) error {
	if latitud != nil && longitud != nil {
		lat := *latitud
		lon := *longitud
		if lat < -23 || lat > -9 {
			return errors.ValidationError("latitud", "Latitud debe estar dentro de Bolivia (-9° a -23°)")
		}
		if lon < -69 || lon > -57 {
			return errors.ValidationError("longitud", "Longitud debe estar dentro de Bolivia (-69° a -57°)")
		}
	}
	return nil
}
