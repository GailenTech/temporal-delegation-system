package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-workflow/internal/activities"
	"temporal-workflow/internal/models"
)

// DelegationWorkflowInput entrada para el workflow de delegación
type DelegationWorkflowInput struct {
	Delegation models.Delegation `json:"delegation"`
}

// DelegationWorkflowResult resultado del workflow de delegación  
type DelegationWorkflowResult struct {
	DelegationID string `json:"delegation_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// DelegationWorkflow gestiona el ciclo de vida completo de una delegación
func DelegationWorkflow(ctx workflow.Context, input DelegationWorkflowInput) (*DelegationWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("DelegationWorkflow started", "delegation_id", input.Delegation.ID)

	// Configurar retry policy para actividades
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second * 1,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Second * 100,
		MaximumAttempts:    3,
	}

	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy:         retryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// 1. Validar la delegación (permisos, usuarios, fechas)
	var validationResult *activities.DelegationValidationResult
	err := workflow.ExecuteActivity(ctx, activities.ValidateDelegation, input.Delegation).Get(ctx, &validationResult)
	if err != nil {
		logger.Error("Failed to validate delegation", "error", err)
		return &DelegationWorkflowResult{
			DelegationID: input.Delegation.ID,
			Status:       "failed",
			Message:      fmt.Sprintf("Validation failed: %v", err),
		}, err
	}

	if !validationResult.IsValid {
		logger.Error("Delegation validation failed", "reasons", validationResult.ValidationErrors)
		return &DelegationWorkflowResult{
			DelegationID: input.Delegation.ID,
			Status:       "invalid",
			Message:      fmt.Sprintf("Invalid delegation: %v", validationResult.ValidationErrors),
		}, nil
	}

	// 2. Activar la delegación inmediatamente si ya es hora, o esperar
	now := workflow.Now(ctx)
	if input.Delegation.StartDate.After(now) {
		logger.Info("Delegation scheduled for future activation", 
			"start_date", input.Delegation.StartDate,
			"current_time", now)
		
		// Esperar hasta la fecha de inicio
		err = workflow.Sleep(ctx, input.Delegation.StartDate.Sub(now))
		if err != nil {
			logger.Error("Failed to wait for start date", "error", err)
			return &DelegationWorkflowResult{
				DelegationID: input.Delegation.ID,
				Status:       "failed",
				Message:      fmt.Sprintf("Failed to schedule activation: %v", err),
			}, err
		}
	}

	// 3. Activar la delegación
	err = workflow.ExecuteActivity(ctx, activities.ActivateDelegation, input.Delegation.ID).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to activate delegation", "error", err)
		return &DelegationWorkflowResult{
			DelegationID: input.Delegation.ID,
			Status:       "failed",
			Message:      fmt.Sprintf("Activation failed: %v", err),
		}, err
	}

	logger.Info("Delegation activated successfully", "delegation_id", input.Delegation.ID)

	// 4. Configurar timer para desactivación automática
	endTime := input.Delegation.EndDate.Sub(workflow.Now(ctx))
	timer := workflow.NewTimer(ctx, endTime)
	
	// 5. Configurar selector para manejar señales y timer
	selector := workflow.NewSelector(ctx)
	
	// Canal para señales de modificación/cancelación
	modifyChannel := workflow.GetSignalChannel(ctx, "modify_delegation")
	cancelChannel := workflow.GetSignalChannel(ctx, "cancel_delegation")

	// Resultado final
	result := &DelegationWorkflowResult{
		DelegationID: input.Delegation.ID,
		Status:       "active",
		Message:      "Delegation is active",
	}

	for {
		selector.AddFuture(timer, func(f workflow.Future) {
			// Timer expirado - desactivar delegación
			logger.Info("Delegation expired, deactivating", "delegation_id", input.Delegation.ID)
			
			err := workflow.ExecuteActivity(ctx, activities.DeactivateDelegation, input.Delegation.ID).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to deactivate expired delegation", "error", err)
				result.Status = "failed"
				result.Message = fmt.Sprintf("Failed to deactivate: %v", err)
			} else {
				result.Status = "expired"
				result.Message = "Delegation expired and deactivated"
			}
		})

		selector.AddReceive(modifyChannel, func(c workflow.ReceiveChannel, more bool) {
			if !more {
				return
			}
			
			var modifySignal activities.ModifyDelegationSignal
			c.Receive(ctx, &modifySignal)
			
			logger.Info("Received delegation modification signal", 
				"delegation_id", input.Delegation.ID,
				"action", modifySignal.Action)

			switch modifySignal.Action {
			case "extend":
				// Extender la delegación
				err := workflow.ExecuteActivity(ctx, activities.ExtendDelegation, 
					input.Delegation.ID, modifySignal.NewEndDate).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to extend delegation", "error", err)
				} else {
					// Actualizar timer - crear nuevo timer y el anterior será ignorado
					newEndTime := modifySignal.NewEndDate.Sub(workflow.Now(ctx))
					timer = workflow.NewTimer(ctx, newEndTime)
					logger.Info("Delegation extended", "new_end_date", modifySignal.NewEndDate)
				}
				
			case "modify_amount":
				// Modificar límite de monto
				err := workflow.ExecuteActivity(ctx, activities.ModifyDelegationAmount, 
					input.Delegation.ID, modifySignal.NewMaxAmount).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to modify delegation amount", "error", err)
				} else {
					logger.Info("Delegation amount modified", "new_amount", modifySignal.NewMaxAmount)
				}
			}
		})

		selector.AddReceive(cancelChannel, func(c workflow.ReceiveChannel, more bool) {
			if !more {
				return
			}
			
			var cancelSignal activities.CancelDelegationSignal
			c.Receive(ctx, &cancelSignal)
			
			logger.Info("Received delegation cancellation signal", 
				"delegation_id", input.Delegation.ID,
				"reason", cancelSignal.Reason)

			// Cancelar delegación inmediatamente
			err := workflow.ExecuteActivity(ctx, activities.DeactivateDelegation, input.Delegation.ID).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to cancel delegation", "error", err)
				result.Status = "failed"
				result.Message = fmt.Sprintf("Failed to cancel: %v", err)
			} else {
				result.Status = "cancelled"
				result.Message = fmt.Sprintf("Delegation cancelled: %s", cancelSignal.Reason)
			}
			
			return // Terminar workflow
		})

		// Ejecutar selector
		selector.Select(ctx)
		
		// Si el resultado no es "active", terminar
		if result.Status != "active" {
			break
		}
	}

	logger.Info("DelegationWorkflow completed", 
		"delegation_id", input.Delegation.ID,
		"status", result.Status)

	return result, nil
}

// DelegationStatusQuery query para obtener el estado actual de una delegación
func DelegationStatusQuery(ctx workflow.Context, delegationID string) (*models.DelegationStatus, error) {
	// Obtener estado actual desde la actividad
	var status *models.DelegationStatus
	err := workflow.ExecuteActivity(ctx, activities.GetDelegationStatus, delegationID).Get(ctx, &status)
	if err != nil {
		return nil, err
	}
	
	return status, nil
}