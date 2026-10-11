package jobs

import (
	"fmt"
	"time"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
)

// RentJob - Job diario del módulo de alquiler:
// marca cuotas atrasadas, vence contratos finalizados y libera propiedades alquiladas
type RentJob struct {
	DB *config.Database
}

// NewRentJob - Crea un nuevo job de alquiler
func NewRentJob() (*RentJob, error) {
	db, err := config.NewDatabase()
	if err != nil {
		return nil, err
	}

	return &RentJob{
		DB: db,
	}, nil
}

// UpdateOverduePayments - Marca como atrasadas las cuotas/depósitos pendientes cuya fecha de vencimiento ya pasó
func (j *RentJob) UpdateOverduePayments() (int64, error) {
	today := time.Now().Truncate(24 * time.Hour)

	result := j.DB.DB.Model(&models.RentPayment{}).
		Where("estado = ? AND tipo IN ? AND deleted_at IS NULL AND fecha_pago < ?",
			models.PaymentStatusPendiente,
			[]models.PaymentType{models.PaymentTypeCuota, models.PaymentTypeDeposito},
			today).
		Update("estado", models.PaymentStatusAtrasado)

	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ExpireContracts - Marca como vencidos los contratos activos cuya fecha_fin ya pasó
// y libera las propiedades que ya no tienen contratos activos
func (j *RentJob) ExpireContracts() (int64, error) {
	today := time.Now().Truncate(24 * time.Hour)

	// Buscar contratos activos vencidos
	var expired []models.RentalContract
	err := j.DB.DB.
		Preload("Property").
		Where("estado = ? AND fecha_fin < ? AND deleted_at IS NULL",
			models.ContractStatusActivo, today).
		Find(&expired).Error
	if err != nil {
		return 0, err
	}

	if len(expired) == 0 {
		return 0, nil
	}

	// Marcar como vencidos
	var count int64
	for _, contract := range expired {
		result := j.DB.DB.Model(&models.RentalContract{}).
			Where("id = ? AND estado = ?", contract.ID, models.ContractStatusActivo).
			Update("estado", models.ContractStatusVencido)
		if result.Error != nil {
			continue
		}
		if result.RowsAffected > 0 {
			count++

			// 🔔 Notificar a ambas partes
			msg := "El contrato de \"" + contract.Property.Titulo + "\" finalizó el " +
				contract.FechaFin.Format("02/01/2006")
			if contract.TipoContrato == models.ContractTypeAnticretico {
				msg += ". Recuerda registrar la devolución del anticrés"
			}
			j.DB.DB.Create(&models.Notification{
				UserID:     contract.VendedorID,
				PropertyID: &contract.PropertyID,
				Tipo:       "contrato_finalizado",
				Titulo:     "Contrato finalizado",
				Mensaje:    msg,
				IsRead:     false,
			})
			j.DB.DB.Create(&models.Notification{
				UserID:     contract.InquilinoID,
				PropertyID: &contract.PropertyID,
				Tipo:       "contrato_finalizado",
				Titulo:     "Contrato finalizado",
				Mensaje:    msg,
				IsRead:     false,
			})
		}
	}

	// Liberar propiedades que ya no tienen contratos activos
	// (subquery: propiedades con contrato activo no deben tocarse)
	j.DB.DB.Exec(`
		UPDATE propiedades
		SET alquilada = false
		WHERE alquilada = true
		AND deleted_at IS NULL
		AND NOT EXISTS (
			SELECT 1 FROM rental_contracts rc
			WHERE rc.property_id = propiedades.id
			AND rc.estado = 'activo'
			AND rc.deleted_at IS NULL
		)
	`)

	return count, nil
}

// RunOnce - Ejecuta todas las tareas del módulo de alquiler
func (j *RentJob) RunOnce() {
	if overdue, err := j.UpdateOverduePayments(); err != nil {
		fmt.Printf("❌ [RentJob] Error marcando cuotas atrasadas: %v\n", err)
	} else if overdue > 0 {
		fmt.Printf("⏰ [RentJob] %d cuotas marcadas como atrasadas\n", overdue)
	}

	if expired, err := j.ExpireContracts(); err != nil {
		fmt.Printf("❌ [RentJob] Error venciendo contratos: %v\n", err)
	} else if expired > 0 {
		fmt.Printf("📜 [RentJob] %d contratos marcados como vencidos\n", expired)
	}
}

// StartRentScheduler - Inicia el scheduler diario del módulo de alquiler
func (j *RentJob) StartRentScheduler(intervalHours int) {
	fmt.Printf("⏰ [RentJob] Scheduler de alquiler iniciado: ejecutando cada %d horas\n", intervalHours)

	// Ejecutar inmediatamente al inicio
	j.RunOnce()

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		j.RunOnce()
	}
}
